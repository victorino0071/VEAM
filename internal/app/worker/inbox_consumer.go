package worker

import (
	"context"
	"asaas_framework/internal/app/acl"
	"asaas_framework/internal/app/service"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"encoding/json"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type InboxConsumer struct {
	repo    port.Repository
	service *service.PaymentService
	baseT   time.Duration
	maxT    time.Duration
	k       int
	quit    chan struct{}
}

func NewInboxConsumer(repo port.Repository, svc *service.PaymentService) *InboxConsumer {
	return &InboxConsumer{
		repo:    repo,
		service: svc,
		baseT:   500 * time.Millisecond,
		maxT:    30 * time.Second,
		k:       0,
		quit:    make(chan struct{}),
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
	var webhook acl.AsaasWebhookDTO
	if err := json.Unmarshal(event.Payload, &webhook); err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] Falha na tradução de Payload JSON (ACL)", "error", err, "id", event.ID)
		return false
	}

	tx, err := webhook.Payment.ToDomain()
	if err != nil {
		slog.ErrorContext(ctx, "[InboxConsumer] Falha no mapeamento para entidade de Domínio", "error", err, "id", event.ID)
		return false
	}

	slog.InfoContext(ctx, "[InboxConsumer] Evento traduzido com sucesso para Domínio", "id", tx.ID, "target_status", tx.Status)

	err = c.service.ProcessPaymentWithMetadata(ctx, tx, event.Metadata, tx.Status)
	if err != nil {
		slog.WarnContext(ctx, "Falha no domínio", "error", err)
		return false
	}

	return true
}

func (c *InboxConsumer) Stop() {
	close(c.quit)
}
