package acl

import (
	"asaas_framework/internal/domain/entity"
	"time"
)

// WebhookDTO representa o payload raiz enviado pelo provedor via Webhook.
type WebhookDTO struct {
	Event   string            `json:"event"`
	Payment WebhookPaymentDTO `json:"payment"`
}

// WebhookPaymentDTO representa o contrato de pagamento da API/Webhook.
type WebhookPaymentDTO struct {
	ID                string  `json:"id"`
	Customer          string  `json:"customer"`
	Value             float64 `json:"value"`
	NetValue          float64 `json:"netValue"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	DueDate           string  `json:"dueDate"`
	PaymentDate       string  `json:"paymentDate"`
	InvoiceUrl        string  `json:"invoiceUrl"`
}

// ToDomain converte um DTO para a entidade core do nosso Domínio.
func (dto *WebhookPaymentDTO) ToDomain(providerID string) (*entity.Transaction, error) {
	dueDate, _ := time.Parse("2006-01-02", dto.DueDate)
	
	status := mapProviderStatus(dto.Status)

	return &entity.Transaction{
		ID:          dto.ID,
		CustomerID:  dto.Customer,
		ProviderID:  providerID,
		Amount:      dto.Value,
		// Currency: "BRL" removido para evitar corrupção lógica (deixado vazio para Domain resolver)
		Status:      status,
		Description: dto.Description,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func mapProviderStatus(providerStatus string) entity.PaymentStatus {
	switch providerStatus {
	case "RECEIVED":
		return entity.StatusReceived
	case "CONFIRMED":
		return entity.StatusConfirmed
	case "OVERDUE", "FAILED":
		return entity.StatusFailed
	case "REFUNDED":
		return entity.StatusRefunded
	case "REFUND_REQUESTED":
		return entity.StatusRefundInitiated
	default:
		return entity.StatusPending
	}
}
