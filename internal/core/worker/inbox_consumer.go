package worker

import (
	"context"
	"fmt"
	"github.com/Victor/VEAM/domain/entity"
	"github.com/Victor/VEAM/domain/port"
	"github.com/Victor/VEAM/domain/registry"
	"github.com/Victor/VEAM/internal/core/service"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type InboxConsumer struct {
	repo     port.Repository
	service  *service.PaymentService
	registry *registry.ProviderRegistry
	baseT      time.Duration
	maxT       time.Duration
	k          int
	maxRetries int
	quit       chan struct{}
}

func NewInboxConsumer(repo port.Repository, svc *service.PaymentService, reg *registry.ProviderRegistry, maxRetries int) *InboxConsumer {
	return &InboxConsumer{
		repo:       repo,
		service:    svc,
		registry:   reg,
		baseT:      500 * time.Millisecond,
		maxT:       30 * time.Second,
		k:          0,
		maxRetries: maxRetries,
		quit:       make(chan struct{}),
	}
}

func (c *InboxConsumer) SetMaxRetries(limit int) {
	c.maxRetries = limit
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

		queueDuration := time.Since(event.CreatedAt).Milliseconds()
		tracer := otel.Tracer("inbox-consumer")
		
		workerCtx, span := tracer.Start(workerCtx, "ProcessInboxEvent", 
			trace.WithAttributes(
				attribute.Int64("messaging.message.time_in_queue_ms", queueDuration),
				attribute.String("event.id", event.ID),
				attribute.String("event.type", event.EventType),
			),
		)

		slog.InfoContext(workerCtx, "[InboxConsumer] Ingressando Phase B (Background Exec)", "event_id", event.ID)

		// Nota: O InboxConsumer orquestra o domínio. Em uma implementação industrial,
		// o fail-fast deveria vir de um breaker global de saúde do domínio.
		// A lógica de resiliência aqui é garantida pela atomicidade do Service.ProcessPayment.
		
		success, workerErr := c.processEvent(workerCtx, event)

		// PHASE C: Finalize (Nova Transação Curta p/ Update de Status)
		if success {
			if err := c.repo.MarkInboxCompleted(ctx, event.ID); err != nil {
				slog.ErrorContext(workerCtx, "[InboxConsumer] Erro na Phase C (Completed)", "error", err, "id", event.ID)
			}
		} else {
			errStr := workerErr.Error()
			if event.RetryCount >= c.maxRetries {
				if err := c.repo.MoveInboxToDLQ(ctx, event.ID, errStr); err != nil {
					slog.ErrorContext(workerCtx, "[InboxConsumer] Erro na Phase C (DLQ)", "error", err, "id", event.ID)
				} else {
					slog.WarnContext(workerCtx, "[InboxConsumer] Inbox Event arquivado na DLQ (Poison Pill)", "id", event.ID, "last_error", errStr)
				}
			} else {
				if err := c.repo.MarkInboxFailed(ctx, event.ID, errStr); err != nil {
					slog.ErrorContext(workerCtx, "[InboxConsumer] Erro na Phase C (Failed)", "error", err, "id", event.ID)
				}
			}
			span.SetAttributes(attribute.String("error.message", errStr))
		}
		span.End()
	}

	return len(events)
}

func (c *InboxConsumer) processEvent(ctx context.Context, event *entity.InboxEvent) (bool, error) {
	providerID, ok := event.Metadata["provider_id"]
	if !ok {
		err := fmt.Errorf("provider_id ausente no metadata")
		slog.ErrorContext(ctx, "[InboxConsumer] provider_id ausente", "id", event.ID)
		return false, err
	}

	adapter, err := c.registry.Get(providerID)
	if err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] adaptador não encontrado para provedor", "provider_id", providerID, "error", err)
		return false, err
	}

	tx, targetStatus, err := adapter.TranslatePayload(ctx, event.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] falha na tradução delegada via adaptador", "provider_id", providerID, "error", err)
		return false, err
	}

	slog.InfoContext(ctx, "[InboxConsumer] Evento traduzido com sucesso via Adaptador", "id", tx.ID, "target_status", targetStatus)

	if err := c.service.ProcessPaymentWithMetadata(ctx, tx, event.Metadata, targetStatus); err != nil {
		slog.WarnContext(ctx, "Falha no domínio via Service", "error", err)
		return false, err
	}

	return true, nil
}

func (c *InboxConsumer) Stop() {
	close(c.quit)
}
