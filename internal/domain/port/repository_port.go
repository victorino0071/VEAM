package port

import (
	"context"
	"asaas_framework/internal/domain/entity"
)

// Repository define a interface para persistência de dados.
type Repository interface {
	SaveTransaction(ctx context.Context, tx *entity.Transaction) error
	GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error)
	
	SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error
	SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error

	// Metodos de Fila (Postgres SKIP LOCKED)
	FetchNextPendingInbox(ctx context.Context, limit int) ([]*entity.InboxEvent, error)
	FetchNextPendingOutbox(ctx context.Context, limit int) ([]*entity.OutboxEvent, error)
	
	// ExecuteInTransaction envolve operações em uma transação ACID.
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
