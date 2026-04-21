package entity

import "time"

type OutboxEvent struct {
	ID          string
	Metadata    map[string]string // JSONB: W3C Trace Context + Baggage
	EventType   string
	Payload     []byte
	Status      string
	RetryCount  int
	LastError   *string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

func NewOutboxEvent(id, eventType string, payload []byte, metadata map[string]string) *OutboxEvent {
	return &OutboxEvent{
		ID:         id,
		Metadata:   metadata,
		EventType:  eventType,
		Payload:    payload,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}
}

type InboxEvent struct {
	ID          string
	Metadata    map[string]string // JSONB: W3C Trace Context + Baggage
	ExternalID  string            // ID vindo do Asaas
	EventType   string
	Payload     []byte
	Status      string
	RetryCount  int
	LastError   *string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

func NewInboxEvent(id, externalID, eventType string, payload []byte, metadata map[string]string) *InboxEvent {
	return &InboxEvent{
		ID:         id,
		ExternalID: externalID,
		Metadata:   metadata,
		EventType:  eventType,
		Payload:    payload,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}
}
