package acl

import (
	"context"
	"net/http"

	"github.com/victorino0071/VEAM/domain/entity"
	"github.com/victorino0071/VEAM/domain/port"
)

// InternalSystemAdapter é uma porta de segurança para processamento endógeno do motor (Sagas).
// Ele intercepta eventos do próprio sistema e converte-os em transições de estado,
// poupando o InboxConsumer de lógicas condicionais poluentes.
type InternalSystemAdapter struct {
	repo port.Repository
}

func NewInternalSystemAdapter(repo port.Repository) port.GatewayAdapter {
	return &InternalSystemAdapter{
		repo: repo,
	}
}

func (a *InternalSystemAdapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	return "", nil
}

func (a *InternalSystemAdapter) CreateTransaction(ctx context.Context, transaction *entity.Transaction) (string, error) {
	return "", nil
}

func (a *InternalSystemAdapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	return "", nil
}

func (a *InternalSystemAdapter) RefundTransaction(ctx context.Context, transactionID string, idempotencyKey string) error {
	return nil
}

func (a *InternalSystemAdapter) ValidateWebhook(r *http.Request) (bool, error) {
	// A via verde interna - não há webhook HMAC do lado de fora, a confiança é absoluta.
	return true, nil
}

func (a *InternalSystemAdapter) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return nil, nil // Não entra via webhook standard
}

// TranslatePayload interpreta o evento do próprio motor (Ex: GATEWAY_REFUND_REJECTED) e converte-o 
// na instrução tática de compensação da FSM.
func (a *InternalSystemAdapter) TranslatePayload(ctx context.Context, payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	txID := string(payload)
	
	// Como a ACL tem visibilidade do repositório, reidrata a transação local
	tx, err := a.repo.GetTransactionByID(ctx, txID)
	if err != nil {
		return nil, "", err
	}
	
	// Força o rollback (RefundFailed)
	return tx, entity.StatusRefundFailed, nil
}
