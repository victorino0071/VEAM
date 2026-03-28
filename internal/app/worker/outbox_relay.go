package worker

import (
	"context"
	"asaas_framework/internal/domain/port"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type OutboxRelay struct {
	repo    port.Repository
	gateway port.GatewayAdapter
	breaker port.CircuitBreaker
	baseT   time.Duration
	maxT    time.Duration
	k       int
	quit    chan struct{}
}

func NewOutboxRelay(repo port.Repository, gateway port.GatewayAdapter, breaker port.CircuitBreaker) *OutboxRelay {
	return &OutboxRelay{
		repo:    repo,
		gateway: gateway,
		breaker: breaker,
		baseT:   500 * time.Millisecond,
		maxT:    30 * time.Second,
		k:       0,
		quit:    make(chan struct{}),
	}
}

func (r *OutboxRelay) Start(ctx context.Context) {
	for {
		// Verificamos o disjuntor reativo antes de qualquer operação
		allowed, _ := r.breaker.Allow(ctx)
		if !allowed {
			slog.WarnContext(ctx, "[OutboxRelay] Breaker aberto: pausando envio", "retry_after", r.baseT)
			time.Sleep(r.baseT)
			continue
		}

		processed := r.consumeBatch(ctx)

		if processed == 0 {
			r.k++
		} else {
			r.k = 0
		}

		backoff := time.Duration(int64(r.baseT) << r.k)
		if backoff > r.maxT || backoff <= 0 {
			backoff = r.maxT
		}

		select {
		case <-time.After(backoff):
			continue
		case <-r.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (r *OutboxRelay) consumeBatch(ctx context.Context) int {
	// PHASE A: Claim (Transação Curta + Commit Imediato)
	events, err := r.repo.ClaimOutboxEvents(ctx, 10)
	if err != nil || len(events) == 0 {
		return 0
	}

	// PHASE B: Execute Outside DB Transaction (Network I/O)
	for _, event := range events {
		// Extração de Contexto W3C
		propagator := otel.GetTextMapPropagator()
		carrier := propagation.MapCarrier(event.Metadata)
		workerCtx := propagator.Extract(ctx, carrier)

		slog.InfoContext(workerCtx, "[OutboxRelay] Enviando evento externo (Phase B)", "event_id", event.ID)

		// Chamada de gateway com Circuit Breaker Reativo
		err := r.gateway.RefundTransaction(workerCtx, event.ID)
		r.breaker.RecordResult(workerCtx, err) // Atualiza EWMA e Estado

		// PHASE C: Finalize (Nova Transação Curta para Status)
		success := (err == nil)
		if err := r.repo.FinalizeOutboxEvent(ctx, event.ID, success); err != nil {
			slog.ErrorContext(workerCtx, "[OutboxRelay] Erro na Phase C (Finalize)", "error", err, "id", event.ID)
		}
	}

	return len(events)
}

func (r *OutboxRelay) Stop() {
	close(r.quit)
}
