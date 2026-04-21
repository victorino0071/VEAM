package entity

import (
	"context"
	"sync"
	"time"
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
	Metadata      map[string]string
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
	metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	mu            sync.RWMutex

	policies []TransitionPolicy // Políticas injetáveis para validação de FSM
}

func (t *Transaction) GetMetadata(key string, fallback string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.metadata == nil {
		return fallback
	}
	if val, ok := t.metadata[key]; ok {
		return val
	}
	return fallback
}

func (t *Transaction) SetMetadata(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.metadata == nil {
		t.metadata = make(map[string]string)
	}
	t.metadata[key] = value
}

func (t *Transaction) Status() PaymentStatus {
	return t.status
}

func (t *Transaction) ToSnapshot() TransactionSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s := TransactionSnapshot{
		ID:            t.ID,
		CustomerID:    t.CustomerID,
		Amount:        t.Amount,
		Currency:      t.Currency,
		Status:        t.status,
		Description:   t.Description,
		DueDate:       t.DueDate,
		ProviderID:    t.ProviderID,
		Metadata:      make(map[string]string, len(t.metadata)),
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}

	for k, v := range t.metadata {
		s.Metadata[k] = v
	}
	if t.PaymentDate != nil {
		d := *t.PaymentDate
		s.PaymentDate = &d
	}
	if t.ConfirmedDate != nil {
		d := *t.ConfirmedDate
		s.ConfirmedDate = &d
	}
	return s
}

// RestoreTransaction é a fábrica estática de reidratação (Memento).
// Diferente de um "ApplySnapshot", esta função garante que a entidade renasça 
// com suas defesas de domínio (DefaultTransitionPolicy) ativas.
func RestoreTransaction(s TransactionSnapshot) *Transaction {
	t := &Transaction{
		ID:            s.ID,
		CustomerID:    s.CustomerID,
		Amount:        s.Amount,
		Currency:      s.Currency,
		status:        s.Status,
		Description:   s.Description,
		DueDate:       s.DueDate,
		ProviderID:    s.ProviderID,
		metadata:      make(map[string]string, len(s.Metadata)),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}

	for k, v := range s.Metadata {
		t.metadata[k] = v
	}
	if s.PaymentDate != nil {
		d := *s.PaymentDate
		t.PaymentDate = &d
	}
	if s.ConfirmedDate != nil {
		d := *s.ConfirmedDate
		t.ConfirmedDate = &d
	}
	
	// Auto-Injeção de Defesa na Restauração (Resolvendo o Paradoxo da Hidratação)
	t.policies = append(t.policies, &DefaultTransitionPolicy{})
	
	return t
}

// WithPolicies anexa regras de transição customizadas à transação.
func (t *Transaction) WithPolicies(policies ...TransitionPolicy) *Transaction {
	t.policies = append(t.policies, policies...)
	return t
}

// TransitionTo avalia a mudança de estado contra todas as políticas injetadas.
func (t *Transaction) TransitionTo(ctx context.Context, newState PaymentStatus, eventID string, metadata map[string]string) (*OutboxEvent, error) {
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

	return NewOutboxEvent(eventID, eventType, []byte(t.ID), metadata), nil
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
		metadata:    make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Garante a política padrão
	tx.policies = append(tx.policies, &DefaultTransitionPolicy{})
	return tx
}
