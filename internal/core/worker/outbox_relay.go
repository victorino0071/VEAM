package worker

import (
	"context"
	"errors"
	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/domain/registry"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type OutboxRelay struct {
	repo     port.Repository
	registry *registry.ProviderRegistry
	breaker    port.CircuitBreaker
	baseT      time.Duration
	maxT       time.Duration
	k          int
	maxRetries int
	quit       chan struct{}
}

func NewOutboxRelay(repo port.Repository, reg *registry.ProviderRegistry, breaker port.CircuitBreaker, maxRetries int) *OutboxRelay {
	return &OutboxRelay{
		repo:       repo,
		registry:   reg,
		breaker:    breaker,
		baseT:      500 * time.Millisecond,
		maxT:       30 * time.Second,
		k:          0,
		maxRetries: maxRetries,
		quit:       make(chan struct{}),
	}
}

func (r *OutboxRelay) SetMaxRetries(limit int) {
	r.maxRetries = limit
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

		// Janela deslizante do Outbox (Sliding Window based on EWMA)
		pf, _ := r.breaker.GetFailureProbability(ctx)
		batchLimit := 10
		if pf > 0 {
			shrink := int(10.0 * (1.0 - pf))
			if shrink < 1 {
				shrink = 1
			}
			batchLimit = shrink
		}

		processed := r.consumeBatch(ctx, batchLimit)

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

func (r *OutboxRelay) consumeBatch(ctx context.Context, limit int) int {
	// PHASE A: Claim (Transação Curta + Commit Imediato com limit dinâmico)
	events, err := r.repo.ClaimOutboxEvents(ctx, limit)
	if err != nil {
		slog.ErrorContext(ctx, "[OutboxRelay] Falha crítica ao reivindicar eventos", "error", err)
		return 0
	}
	if len(events) == 0 {
		return 0
	}

	slog.InfoContext(ctx, "[OutboxRelay] Lote de eventos de saída reivindicado", "count", len(events))

	// PHASE B: Execute Outside DB Transaction (Network I/O)
	for _, event := range events {
		workerCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.Metadata))
		
		queueDuration := time.Since(event.CreatedAt).Milliseconds()
		tracer := otel.Tracer("outbox-relay")
		workerCtx, span := tracer.Start(workerCtx, "RelayOutboxEvent", 
			trace.WithAttributes(
				attribute.Int64("messaging.message.time_in_queue_ms", queueDuration),
				attribute.String("event.id", event.ID),
				attribute.String("event.type", event.EventType),
			),
		)

		slog.InfoContext(workerCtx, "[OutboxRelay] Enviando evento externo (Phase B)", "event_id", event.ID)
		
		// Aborta o processamento do resto do lote se o disjuntor abrir no meio (Intra-Batch Fail-Fast)
		if state, _ := r.breaker.GetState(ctx); state == "OPEN" {
			slog.WarnContext(ctx, "[OutboxRelay] Disjuntor abriu durante o lote. Abortando execução residual.")
			break 
		}

		// O Payload contém o ID da Transação do Domínio
		txID := string(event.Payload)
		var err error

		// Recupera o adaptador via Registry usando o rastro do metadados
		providerID := event.Metadata["provider_id"]
		gateway, regErr := r.registry.Get(providerID)
		if regErr != nil {
			slog.ErrorContext(workerCtx, "[OutboxRelay] Falha ao resolver gateway", "error", regErr, "id", event.ID, "provider_id", providerID)
			err = regErr
		} else {
			switch event.EventType {
			case "REFUND_STARTED":
				// Chamada de gateway isolada com Circuit Breaker Reativo, injeta Idempotency Key (event.ID)
				err = gateway.RefundTransaction(workerCtx, txID, event.ID)
				r.breaker.RecordResult(workerCtx, err) // Atualiza EWMA e Estado
				
				if err != nil {
					slog.WarnContext(workerCtx, "[OutboxRelay] Falha ao solicitar estorno no Provedor", "error", err, "tx_id", txID)
					
					if errors.Is(err, entity.ErrTerminalGatewayRejection) {
						slog.InfoContext(workerCtx, "[OutboxRelay] Falha Terminal Detectada. Injetando InboxEvent de Compensação (Saga)", "tx_id", txID)
						compensationEvent := entity.NewInboxEvent(
							event.ID, // reaproveita ID garantindo atomicidade visual no log
							txID,
							"GATEWAY_REFUND_REJECTED",
							[]byte(txID),
							map[string]string{
								"provider_id": "SYSTEM_INTERNAL",
								"original_outbox_id": event.ID,
								"error": err.Error(),
							},
						)
						// Insere e encobre erro original (marcando Outbox relayer como completed porque o erro engatou engine de resolução nativa)
						if inboxErr := r.repo.SaveInboxEvent(ctx, compensationEvent); inboxErr != nil {
							slog.ErrorContext(workerCtx, "Erro grave ao salvar compensação", "error", inboxErr)
						} else {
							err = nil // Absorve falha: fluxo assíncrono agora lida com o Gateway refund rejected no inbox.
						}
					}
				} else {
					slog.InfoContext(workerCtx, "[OutboxRelay] Estorno solicitado e confirmado pelo Gateway", "tx_id", txID)
				}

			case "PAYMENT_CONFIRMED", "PAYMENT_FAILED", "PAYMENT_ANOMALY":
				// Aqui é onde você informaria outro microserviço do seu sistema ou enviaria um email!
				slog.InfoContext(workerCtx, "[OutboxRelay] Evento do Domínio publicado em barramento interno (No-Op Gateway)", "event_type", event.EventType, "tx_id", txID)
				err = nil

			default:
				slog.InfoContext(workerCtx, "[OutboxRelay] Evento ignorado pelo Relay", "event_type", event.EventType)
			}
		}

		// PHASE C: Finalize (Nova Transação Curta para Status)
		if err == nil {
			if err := r.repo.MarkOutboxCompleted(ctx, event.ID); err != nil {
				slog.ErrorContext(workerCtx, "[OutboxRelay] Erro na Phase C (Completed)", "error", err, "id", event.ID)
			}
		} else {
			if event.RetryCount >= r.maxRetries {
				if repoErr := r.repo.MoveOutboxToDLQ(ctx, event.ID, err.Error()); repoErr != nil {
					slog.ErrorContext(workerCtx, "[OutboxRelay] Erro na Phase C (DLQ)", "error", repoErr, "id", event.ID)
				} else {
					slog.WarnContext(workerCtx, "[OutboxRelay] Outbox Event arquivado na DLQ (Poison Pill)", "id", event.ID, "last_error", err.Error())
				}
			} else {
				if repoErr := r.repo.MarkOutboxFailed(ctx, event.ID, err.Error()); repoErr != nil {
					slog.ErrorContext(workerCtx, "[OutboxRelay] Erro na Phase C (Failed)", "error", repoErr, "id", event.ID)
				}
			}
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}
		span.End()
	}

	return len(events)
}

func (r *OutboxRelay) Stop() {
	close(r.quit)
}
