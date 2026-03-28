package entity

import "time"

type SubscriptionCycle string

const (
	CycleMonthly   SubscriptionCycle = "MONTHLY"
	CycleAnnually  SubscriptionCycle = "ANNUALLY"
	CycleQuarterly SubscriptionCycle = "QUARTERLY"
)

type SubscriptionStatus string

const (
	SubscriptionActive   SubscriptionStatus = "ACTIVE"
	SubscriptionInactive SubscriptionStatus = "INACTIVE"
)

type Subscription struct {
	ID          string
	CustomerID  string
	Amount      float64
	Cycle       SubscriptionCycle
	Status      SubscriptionStatus
	NextDueDate time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewSubscription(id, customerID string, amount float64, cycle SubscriptionCycle) *Subscription {
	return &Subscription{
		ID:          id,
		CustomerID:  customerID,
		Amount:      amount,
		Cycle:       cycle,
		Status:      SubscriptionActive,
		NextDueDate: time.Now().AddDate(0, 1, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
