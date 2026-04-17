package acl

import (
	"asaas_framework/internal/domain/entity"
	"time"
)

// AsaasWebhookDTO representa o payload raiz enviado pelo Asaas via Webhook.
type AsaasWebhookDTO struct {
	Event   string          `json:"event"`
	Payment AsaasPaymentDTO `json:"payment"`
}

// AsaasPaymentDTO representa o contrato de pagamento da API/Webhook do Asaas.
type AsaasPaymentDTO struct {
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

// ToDomain converte um DTO do Asaas para a entidade core do nosso Domínio.
func (dto *AsaasPaymentDTO) ToDomain() (*entity.Transaction, error) {
	dueDate, _ := time.Parse("2006-01-02", dto.DueDate)
	
	status := mapAsaasStatus(dto.Status)

	return &entity.Transaction{
		ID:          dto.ID,
		CustomerID:  dto.Customer,
		Amount:      dto.Value,
		// Currency: "BRL" removido para evitar corrupção lógica (deixado vazio para Domain resolver)
		Status:      status,
		Description: dto.Description,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func mapAsaasStatus(asaasStatus string) entity.PaymentStatus {
	switch asaasStatus {
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
