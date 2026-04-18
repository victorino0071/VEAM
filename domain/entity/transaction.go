package entity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	StatusPending           PaymentStatus = "PENDING"
	StatusReceived          PaymentStatus = "RECEIVED"
	StatusConfirmed         PaymentStatus = "CONFIRMED"
	StatusPaid              PaymentStatus = "PAID"
	StatusFailed            PaymentStatus = "FAILED"
	StatusCanceled          PaymentStatus = "CANCELED"
	StatusRefunded          PaymentStatus = "REFUNDED"
	StatusRefundInitiated   PaymentStatus = "REFUND_INITIATED"
	StatusChargebackPending PaymentStatus = "CHARGEBACK_PENDING"
	StatusAnomaly           PaymentStatus = "ANOMALY"
)

// TransactionSnapshot representa o estado imutável de uma transação para transporte e persistência.
// Seguindo o padrão Memento para evitar o vazamento de métodos de mutação (rebuilds).
type TransactionSnapshot struct {
	ID            string
	CustomerID    string
	Amount        float64
	Currency      string
	Status        PaymentStatus
	Description   string
	DueDate       time.Time
	PaymentDate   *time.Time
	ConfirmedDate *time.Time
	ProviderID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Transaction struct {
	ID            string
	CustomerID    string
	Amount        float64
	Currency      string
	status        PaymentStatus
	Description   string
	DueDate       time.Time
	PaymentDate   *time.Time
	ConfirmedDate *time.Time
	ProviderID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	policies []TransitionPolicy // Políticas injetáveis para validação de FSM
}

func (t *Transaction) Status() PaymentStatus {
	return t.status
}

func (t *Transaction) ToSnapshot() TransactionSnapshot {
	return TransactionSnapshot{
		ID:            t.ID,
		CustomerID:    t.CustomerID,
		Amount:        t.Amount,
		Currency:      t.Currency,
		Status:        t.status,
		Description:   t.Description,
		DueDate:       t.DueDate,
		PaymentDate:   t.PaymentDate,
		ConfirmedDate: t.ConfirmedDate,
		ProviderID:    t.ProviderID,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

func (t *Transaction) ApplySnapshot(s TransactionSnapshot) {
	t.ID = s.ID
	t.CustomerID = s.CustomerID
	t.Amount = s.Amount
	t.Currency = s.Currency
	t.status = s.Status
	t.Description = s.Description
	t.DueDate = s.DueDate
	t.PaymentDate = s.PaymentDate
	t.ConfirmedDate = s.ConfirmedDate
	t.ProviderID = s.ProviderID
	t.CreatedAt = s.CreatedAt
	t.UpdatedAt = s.UpdatedAt
}

// WithPolicies anexa regras de transição customizadas à transação.
func (t *Transaction) WithPolicies(policies ...TransitionPolicy) *Transaction {
	t.policies = append(t.policies, policies...)
	return t
}

// TransitionTo avalia a mudança de estado contra todas as políticas injetadas.
func (t *Transaction) TransitionTo(ctx context.Context, newState PaymentStatus, metadata map[string]string) (*OutboxEvent, error) {
	// Chain of Responsibility: Todas as políticas devem passar
	for _, p := range t.policies {
		if err := p.Evaluate(ctx, t, newState); err != nil {
			return nil, err
		}
	}

	// Se for exatamente o mesmo, é idempotente no domínio
	if t.status == newState {
		return nil, nil
	}

	// Efeito Colateral: Mudança de Estado
	t.status = newState
	t.UpdatedAt = time.Now()

	// Mapeamento de Eventos (Pode ser abstraído para Policy futuramente se necessário)
	eventType := "PAYMENT_UPDATED"
	if newState == StatusPaid || newState == StatusReceived || newState == StatusConfirmed {
		eventType = "PAYMENT_CONFIRMED"
	} else if newState == StatusFailed {
		eventType = "PAYMENT_FAILED"
	} else if newState == StatusRefundInitiated {
		eventType = "REFUND_STARTED"
	}

	return NewOutboxEvent(uuid.New().String(), eventType, []byte(t.ID), metadata), nil
}

func NewTransaction(id, customerID, providerID string, amount float64, description string, dueDate time.Time) *Transaction {
	tx := &Transaction{
		ID:          id,
		CustomerID:  customerID,
		ProviderID:  providerID,
		Amount:      amount,
		Currency:    "BRL",
		status:      StatusPending,
		Description: description,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Garante a política padrão
	tx.policies = append(tx.policies, &DefaultTransitionPolicy{})
	return tx
}
