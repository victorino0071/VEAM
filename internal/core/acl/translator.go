package acl

import (
	"fmt"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
	"github.com/mercadopago/sdk-go/pkg/payment"
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

func MapAsaasStatus(asaasStatus string) entity.PaymentStatus {
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

// MapMercadoPagoStatus converte o status nativo do Mercado Pago para o status de Domínio.
func MapMercadoPagoStatus(mpStatus string) entity.PaymentStatus {
	switch mpStatus {
	case "approved":
		return entity.StatusPaid
	case "pending", "in_process":
		return entity.StatusPending
	case "rejected":
		return entity.StatusFailed
	case "refunded":
		return entity.StatusRefunded
	case "cancelled":
		return entity.StatusCanceled
	default:
		return entity.StatusPending
	}
}

// MercadoPagoWebhookDTO reflete a estrutura de notificação do MercadoPago.
type MercadoPagoWebhookDTO struct {
	Action string                 `json:"action"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

// MercadoPagoPaymentDTO representa os detalhes essenciais dentro de data no webhook.
type MercadoPagoPaymentDTO struct {
	ID                string  `json:"id"`
	PayerID           string  `json:"payer_id"`
	TransactionAmount float64 `json:"transaction_amount"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	DateOfExpiration  string  `json:"date_of_expiration"`
}

// ToDomain converte DTO do Mercado Pago para a Entidade Transaction (antigo).
func (m *MercadoPagoWebhookDTO) ToDomain(providerID string) (*entity.Transaction, error) {
	return nil, fmt.Errorf("deprecated: use Fetch approach")
}

// ToDomainFromMPPayment converte o Payment do SDK do MP para Transaction.
func ToDomainFromMPPayment(mpPayment *payment.Response, providerID string) (*entity.Transaction, error) {
	if mpPayment == nil {
		return nil, fmt.Errorf("mpPayment is nil")
	}

	var dueDate time.Time
	if !mpPayment.DateOfExpiration.IsZero() {
		dueDate = mpPayment.DateOfExpiration
	} else {
		dueDate = time.Now()
	}

	payerID := ""
	if mpPayment.Payer.ID != "" {
		payerID = mpPayment.Payer.ID
	} else if mpPayment.Payer.Email != "" {
		payerID = mpPayment.Payer.Email
	}

	tx := entity.NewTransaction(
		fmt.Sprintf("%d", mpPayment.ID),
		payerID,
		providerID,
		mpPayment.TransactionAmount,
		mpPayment.Description,
		dueDate,
	)

	return tx, nil
}
