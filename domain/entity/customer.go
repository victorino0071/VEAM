package entity

import "time"

type Customer struct {
	ID        string
	Name      string
	Email     string
	Document  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCustomer(id, name, email, document string) *Customer {
	return &Customer{
		ID:        id,
		Name:      name,
		Email:     email,
		Document:  document,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
