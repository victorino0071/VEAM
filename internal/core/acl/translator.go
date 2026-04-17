package acl

import (
	"github.com/Victor/payment-engine/domain/entity"
	"time"
)

// WebhookDTO representa o contrato universal de entrada para tradução.
type WebhookDTO struct {
	Event   string          `json:"event"`
	Payment AsaasPaymentDTO `json:"payment"`
}

// AsaasPaymentDTO reflete a estrutura proprietária (exemplo: Asaas).
type AsaasPaymentDTO struct {
	ID          string  `json:"id"`
	Customer    string  `json:"customer"`
	Value       float64 `json:"value"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	DueDate     string  `json:"dueDate"`
}

// ToDomain converte DTOs de terceiros em entidades de domínio blindadas.
func (a *AsaasPaymentDTO) ToDomain(providerID string) (*entity.Transaction, error) {
	dueDate, _ := time.Parse("2006-01-02", a.DueDate)
	
	tx := entity.NewTransaction(
		a.ID,
		a.Customer,
		providerID,
		a.Value,
		a.Description,
		dueDate,
	)

	return tx, nil
}

func mapAsaasStatus(asaasStatus string) entity.PaymentStatus {
	switch asaasStatus {
	case "RECEIVED", "CONFIRMED":
		return entity.StatusReceived
	case "OVERDUE", "FAILED":
		return entity.StatusFailed
	case "REFUNDED":
		return entity.StatusRefunded
	default:
		return entity.StatusPending
	}
}
