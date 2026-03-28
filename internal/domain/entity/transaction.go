package entity

import "time"

type PaymentStatus string

const (
	StatusPending  PaymentStatus = "PENDING"
	StatusPaid     PaymentStatus = "PAID"
	StatusFailed   PaymentStatus = "FAILED"
	StatusCanceled PaymentStatus = "CANCELED"
	StatusRefunded PaymentStatus = "REFUNDED"
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
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewTransaction(id, customerID string, amount float64, description string, dueDate time.Time) *Transaction {
	return &Transaction{
		ID:          id,
		CustomerID:  customerID,
		Amount:      amount,
		Currency:    "BRL", // Default para asaas
		Status:      StatusPending,
		Description: description,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
