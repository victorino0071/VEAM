package worker

import (
	"context"
	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/domain/registry"
	"github.com/Victor/payment-engine/internal/core/service"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type InboxConsumer struct {
	repo     port.Repository
	service  *service.PaymentService
	registry *registry.ProviderRegistry
	baseT    time.Duration
	maxT     time.Duration
	k        int
	quit     chan struct{}
}

func NewInboxConsumer(repo port.Repository, svc *service.PaymentService, reg *registry.ProviderRegistry) *InboxConsumer {
	return &InboxConsumer{
		repo:     repo,
		service:  svc,
		registry: reg,
		baseT:    500 * time.Millisecond,
		maxT:     30 * time.Second,
		k:        0,
		quit:     make(chan struct{}),
	}
}

func (c *InboxConsumer) Start(ctx context.Context) {
	for {
		processed := c.consume(ctx)

		if processed == 0 {
			c.k++
		} else {
			c.k = 0
		}

		backoff := time.Duration(int64(c.baseT) << c.k)
		if backoff > c.maxT || backoff <= 0 {
			backoff = c.maxT
		}

		select {
		case <-time.After(backoff):
			continue
		case <-c.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *InboxConsumer) consume(ctx context.Context) int {
	// PHASE A: Claim (Transação Curta + Commit Imediato)
	events, err := c.repo.ClaimInboxEvents(ctx, 10)
	if err != nil || len(events) == 0 {
		return 0
	}

	slog.InfoContext(ctx, "[InboxConsumer] Lote de eventos reivindicado", "count", len(events))

	// PHASE B: Execute Outside DB Transaction
	for _, event := range events {
		// 1. Extrai W3C Context do Metadata JSONB
		propagator := otel.GetTextMapPropagator()
		carrier := propagation.MapCarrier(event.Metadata)
		workerCtx := propagator.Extract(ctx, carrier)

		slog.InfoContext(workerCtx, "[InboxConsumer] Ingressando Phase B (Background Exec)", "event_id", event.ID)

		// Nota: O InboxConsumer orquestra o domínio. Em uma implementação industrial,
		// o fail-fast deveria vir de um breaker global de saúde do domínio.
		// A lógica de resiliência aqui é garantida pela atomicidade do Service.ProcessPayment.
		
		success := c.processEvent(workerCtx, event)

		// PHASE C: Finalize (Nova Transação Curta p/ Update de Status)
		if success {
			if err := c.repo.MarkInboxCompleted(ctx, event.ID); err != nil {
				slog.ErrorContext(workerCtx, "[InboxConsumer] Erro na Phase C (Completed)", "error", err, "id", event.ID)
			}
		} else {
			if err := c.repo.MarkInboxFailed(ctx, event.ID); err != nil {
				slog.ErrorContext(workerCtx, "[InboxConsumer] Erro na Phase C (Failed/DLQ)", "error", err, "id", event.ID)
			}
		}
	}

	return len(events)
}

func (c *InboxConsumer) processEvent(ctx context.Context, event *entity.InboxEvent) bool {
	providerID, ok := event.Metadata["provider_id"]
	if !ok {
		slog.ErrorContext(ctx, "[InboxConsumer] provider_id ausente no metadata", "id", event.ID)
		return false
	}

	adapter, err := c.registry.Get(providerID)
	if err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] adaptador não encontrado para provedor", "provider_id", providerID, "error", err)
		return false
	}

	tx, targetStatus, err := adapter.TranslatePayload(event.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] falha na tradução delegada via adaptador", "provider_id", providerID, "error", err)
		return false
	}

	slog.InfoContext(ctx, "[InboxConsumer] Evento traduzido com sucesso via Adaptador", "id", tx.ID, "target_status", targetStatus)

	if err := c.service.ProcessPaymentWithMetadata(ctx, tx, event.Metadata, targetStatus); err != nil {
		slog.WarnContext(ctx, "Falha no domínio via Service", "error", err)
		return false
	}

	return true
}

func (c *InboxConsumer) Stop() {
	close(c.quit)
}
