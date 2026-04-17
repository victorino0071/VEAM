package handler

import (
	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type WebhookHandler struct {
	repo       port.Repository
	adapter    port.GatewayAdapter
	providerID string
}

func NewWebhookHandler(repo port.Repository, adapter port.GatewayAdapter, providerID string) *WebhookHandler {
	return &WebhookHandler{
		repo:       repo,
		adapter:    adapter,
		providerID: providerID,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Inicia Rastreamento (OpenTelemetry)
	tracer := otel.Tracer("webhook-handler")
	ctx, span := tracer.Start(r.Context(), "ReceiveWebhook")
	defer span.End()

	// 2. W3C Context Injection (Metadata Carrier)
	metadata := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(metadata))
	
	// Validação de Versão Cega (Antifragilidade)
	metadata["schema_version"] = "v1"
	metadata["provider_id"] = h.providerID
	// Idealmente r.Header.Get("Date") ou o campo equivalente do Provedor
	metadata["provider_timestamp"] = r.Header.Get("Date")

	// 3. Verificação de Autorização Delegada (Universal ACL)
	ok, err := h.adapter.ValidateWebhook(r)
	if err != nil || !ok {
		slog.WarnContext(ctx, "Tentativa de acesso não autorizada ou falha na validação", "error", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. Normalização do Payload Delegada
	payload, err := h.adapter.TranslateWebhook(r)
	if err != nil {
		slog.ErrorContext(ctx, "Erro na normalização do webhook", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	webhookID := payload.ExternalID
	eventType := payload.EventType
	body := payload.Payload

	// Previne panic silencioso do google/uuid (vamos usar Must)
	eventUUID := uuid.New().String()

	// 5. Ingestão Cega + Metadata JSONB
	inboxEvent := entity.NewInboxEvent(eventUUID, webhookID, eventType, body, metadata)
	if err := h.repo.SaveInboxEvent(ctx, inboxEvent); err != nil {
		slog.ErrorContext(ctx, "Erro ao persistir Inbox", "error", err)
		http.Error(w, "Erro ao persistir", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "Webhook persistido no Inbox (Mastery)", "webhook_id", webhookID)
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Payload persistido com sucesso.")
}
