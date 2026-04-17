package entity

import "time"

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
	Status        PaymentStatus
	Description   string
	DueDate       time.Time
	PaymentDate   *time.Time
	ConfirmedDate *time.Time
	ProviderID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewTransaction(id, customerID, providerID string, amount float64, description string, dueDate time.Time) *Transaction {
	return &Transaction{
		ID:          id,
		CustomerID:  customerID,
		ProviderID:  providerID,
		Amount:      amount,
		Currency:    "BRL",
		Status:      StatusPending,
		Description: description,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
