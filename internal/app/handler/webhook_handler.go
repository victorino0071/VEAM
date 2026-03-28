package handler

import (
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"fmt"
	"net/http"
	"io/ioutil"
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
	// 1. Verificação Criptográfica (Auth Token)
	token := r.Header.Get("asaas-access-token")
	if token != h.accessToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Leitura do Payload Bruto
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Can't read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. ID do Webhook extraído (idealmente do header ou metadados da requisição se possível)
	// Como o Asaas não envia o ID no header, vamos salvar com um ID único gerado.
	// A idempotência será garantida pela Unique Constraint no worker ou via ID do Asaas no Inbox.
	webhookID := r.Header.Get("asaas-event-id") // Exemplo: Asaas envia um ID de evento

	// 4. Ingestão Cega: Salvamento síncrono do payload bruto sem parsing.
	inboxEvent := entity.NewInboxEvent("id-servidor", webhookID, "Asaas", body)
	if err := h.repo.SaveInboxEvent(r.Context(), inboxEvent); err != nil {
		// Se falhar por Unique Constraint (Duplicado), o repositório retornará nil ou erro específico.
		// ON CONFLICT DO NOTHING garante que não falhe, apenas não insira.
		http.Error(w, "Erro ao persistir", http.StatusInternalServerError)
		return
	}

	// 5. Resposta O(1)
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Payload persistido com sucesso.")
}
