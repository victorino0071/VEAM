package port

import (
	"asaas_framework/internal/domain/entity"
	"context"
)

type Repository interface {
	// Ingestão
	SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error
	SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error

	// Phase A: Claim (SKIP LOCKED + Commit)
	ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error)
	ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error)

	// Phase C: Finalize (Update Status + Commit)
	FinalizeInboxEvent(ctx context.Context, id string, success bool) error
	FinalizeOutboxEvent(ctx context.Context, id string, success bool) error

	// Domínio
	GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error)
	SaveTransaction(ctx context.Context, tx *entity.Transaction) error

	// ExecuteInTransaction envolve operações em uma transação ACID.
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
