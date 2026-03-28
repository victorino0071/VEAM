package worker

import (
	"context"
	"asaas_framework/internal/domain/port"
	"fmt"
	"time"
)

type OutboxRelay struct {
	repo    port.Repository
	gateway port.GatewayAdapter
	breaker port.CircuitBreaker
	baseT   time.Duration // T_base
	maxT    time.Duration // T_max
	k       int           // Tentativas vazias consecutivas
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
		// 1. Verifica Circuit Breaker antes de agir
		allowed, _ := r.breaker.Allow(ctx)
		if !allowed {
			fmt.Println("[OutboxRelay] Breaker aberto: aguardando...")
			time.Sleep(r.baseT)
			continue
		}

		// 2. Processa o lote
		processed := r.processBatch(ctx)

		// 3. Calcula T_poll (Backoff Exponencial)
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

func (r *OutboxRelay) processBatch(ctx context.Context) int {
	events, err := r.repo.FetchNextPendingOutbox(ctx, 10)
	if err != nil || len(events) == 0 {
		return 0
	}

	for _, event := range events {
		// Simula envio
		err := r.gateway.RefundTransaction(ctx, event.ID) // Dummy action
		r.breaker.RecordResult(ctx, err)
		
		if err == nil {
			now := time.Now()
			event.ProcessedAt = &now
		}
	}

	return len(events)
}

func (r *OutboxRelay) Stop() {
	close(r.quit)
}
