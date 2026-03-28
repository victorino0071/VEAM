package entity

import (
	"time"
)

// OutboxEvent representa um evento que deve ser enviado para um sistema externo.
type OutboxEvent struct {
	ID          string
	EventType   string
	Payload     []byte
	Metadata    map[string]string
	RetryCount  int
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

// InboxEvent representa um evento bruto recebido de um webhook externo que precisa ser processado.
type InboxEvent struct {
	ID          string
	ExternalID  string // ID original do gateway (ex: Asaas)
	Source      string // Origem do evento (ex: "Asaas")
	Payload     []byte
	Status      string // "PENDING", "PROCESSED", "FAILED"
	RetryCount  int
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

func NewOutboxEvent(id, eventType string, payload []byte) *OutboxEvent {
	return &OutboxEvent{
		ID:        id,
		EventType: eventType,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
}

func NewInboxEvent(id, externalID, source string, payload []byte) *InboxEvent {
	return &InboxEvent{
		ID:         id,
		ExternalID: externalID,
		Source:     source,
		Payload:    payload,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}
}
