package port

import (
	"github.com/Victor/payment-engine/domain/entity"
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
	MarkInboxCompleted(ctx context.Context, id string) error
	MarkInboxFailed(ctx context.Context, id string, errStr string) error
	MoveInboxToDLQ(ctx context.Context, id string, errStr string) error
	
	MarkOutboxCompleted(ctx context.Context, id string) error
	MarkOutboxFailed(ctx context.Context, id string, errStr string) error
	MoveOutboxToDLQ(ctx context.Context, id string, errStr string) error

	// Replay DLQ
	ReplayInboxDLQ(ctx context.Context, id string) error
	ReplayOutboxDLQ(ctx context.Context, id string) error

	// Domínio
	GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error)
	SaveTransaction(ctx context.Context, tx *entity.Transaction) error

	// ExecuteInTransaction envolve operações em uma transação ACID.
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
