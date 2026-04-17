package handler

import (
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type WebhookHandler struct {
	repo        port.Repository
	accessToken string
}

func NewWebhookHandler(repo port.Repository, accessToken string) *WebhookHandler {
	return &WebhookHandler{
		repo:        repo,
		accessToken: accessToken,
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
	// Idealmente r.Header.Get("Date") ou o campo equivalente do Asaas
	metadata["asaas_timestamp"] = r.Header.Get("Date")

	// 3. Verificação Criptográfica
	token := r.Header.Get("asaas-access-token")
	if token != h.accessToken {
		slog.WarnContext(ctx, "Tentativa de acesso não autorizada")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. Leitura do Payload
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "Erro ao ler body", "error", err)
		http.Error(w, "Can't read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse mínimo para extrair id e event
	var asaasEvt struct {
		ID    string `json:"id"`
		Event string `json:"event"`
	}
	_ = json.Unmarshal(body, &asaasEvt)

	webhookID := asaasEvt.ID
	if webhookID == "" {
		webhookID = r.Header.Get("asaas-event-id") // Fallback
	}
	
	eventType := asaasEvt.Event
	if eventType == "" {
		eventType = "UNKNOWN"
	}

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
