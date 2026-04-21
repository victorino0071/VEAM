package port

import (
	"context"
	"github.com/Victor/payment-engine/domain/entity"
	"net/http"
)

// GatewayAdapter define a interface para comunicação com o provedor de pagamento (Ex: Asaas)
type GatewayAdapter interface {
	CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error)
	CreateTransaction(ctx context.Context, transaction *entity.Transaction) (string, error)
	GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error)
	RefundTransaction(ctx context.Context, transactionID string) error

	// Webhook Methods (Universal ACL)
	ValidateWebhook(r *http.Request) (bool, error)
	TranslateWebhook(r *http.Request) (*WebhookResponse, error)
	TranslatePayload(payload []byte) (*entity.Transaction, entity.PaymentStatus, error)
}

// WebhookResponse normaliza o que vem da rua para o que o motor entende.
type WebhookResponse struct {
	ExternalID string
	EventType  string
	Payload    []byte
}

// IdempotencyStore define a interface para armazenamento e verificação de chaves de idempotência.
type IdempotencyStore interface {
	IsProcessed(ctx context.Context, key string) (bool, error)
	SaveProcessed(ctx context.Context, key string) error
}

// WebhookHandler define a interface para processamento de notificações externas.
type WebhookHandler interface {
	Handle(ctx context.Context, payload []byte, signature string) error
}
