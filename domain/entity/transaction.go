package entity

import (
	"errors"
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

type Transaction struct {
	ID            string
	CustomerID    string
	Amount        float64
	Currency      string
	status        PaymentStatus // Opaque State: minúsculo
	Description   string
	DueDate       time.Time
	PaymentDate   *time.Time
	ConfirmedDate *time.Time
	ProviderID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (t *Transaction) Status() PaymentStatus {
	return t.status
}

// TransitionTo blinda o status e garante o OutboxEvent atrelado à mesma transação.
func (t *Transaction) TransitionTo(newState PaymentStatus, metadata map[string]string) (*OutboxEvent, error) {
	// Se for exatamente o mesmo, é idempotente no domínio
	if t.status == newState {
		return nil, nil
	}

	switch t.status {
	case StatusPending:
		if newState == StatusConfirmed || newState == StatusReceived || newState == StatusPaid {
			t.status = newState
			return NewOutboxEvent(uuid.New().String(), "PAYMENT_CONFIRMED", []byte(t.ID), metadata), nil
		}
		if newState == StatusFailed {
			t.status = newState
			return NewOutboxEvent(uuid.New().String(), "PAYMENT_FAILED", []byte(t.ID), metadata), nil
		}
	case StatusPaid, StatusReceived, StatusConfirmed:
		if newState == StatusRefundInitiated {
			t.status = newState
			return NewOutboxEvent(uuid.New().String(), "REFUND_STARTED", []byte(t.ID), metadata), nil
		}
	}
	return nil, errors.New("fsm_violation: illegal state transition")
}

func NewTransaction(id, customerID, providerID string, amount float64, description string, dueDate time.Time) *Transaction {
	return &Transaction{
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
}

// RebuildFromRepository permite que a camada de infraestrutura reconstrua o estado
// de uma transação persistida sem violar a soberania da FSM.
func RebuildFromRepository(id, customerID, providerID string, status PaymentStatus, amount float64, description string, dueDate time.Time) *Transaction {
	return &Transaction{
		ID:          id,
		CustomerID:  customerID,
		ProviderID:  providerID,
		status:      status,
		Amount:      amount,
		Description: description,
		DueDate:     dueDate,
	}
}
