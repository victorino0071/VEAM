package repository

import (
	"context"
	"database/sql"
	"asaas_framework/internal/domain/entity"
	"fmt"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Ingestão com ON CONFLICT (sqlc-style)
func (r *PostgresRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error {
	fmt.Printf("[sqlc] INSERT INTO inbox (id, external_id, metadata) VALUES (%s, %s, %v) ON CONFLICT DO NOTHING\n", 
		event.ID, event.ExternalID, event.Metadata)
	return nil
}

func (r *PostgresRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	fmt.Printf("[sqlc] INSERT INTO outbox (id, metadata) VALUES (%s, %v)\n", event.ID, event.Metadata)
	return nil
}

// Phase A: Claim (SELECT FOR UPDATE SKIP LOCKED + UPDATE status = 'PROCESSING')
func (r *PostgresRepository) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) {
	fmt.Printf("[sqlc] Claiming %d Inbox Events (SKIP LOCKED)\n", limit)
	// mock return
	return []*entity.InboxEvent{}, nil
}

func (r *PostgresRepository) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	fmt.Printf("[sqlc] Claiming %d Outbox Events (SKIP LOCKED)\n", limit)
	return []*entity.OutboxEvent{}, nil
}

// Phase C: Finalize (Update status = 'COMPLETED' or 'FAILED')
func (r *PostgresRepository) FinalizeInboxEvent(ctx context.Context, id string, success bool) error {
	status := "COMPLETED"
	if !success {
		status = "FAILED"
	}
	fmt.Printf("[sqlc] Finalizing Inbox %s to %s\n", id, status)
	return nil
}

func (r *PostgresRepository) FinalizeOutboxEvent(ctx context.Context, id string, success bool) error {
	status := "COMPLETED"
	if !success {
		status = "FAILED"
	}
	fmt.Printf("[sqlc] Finalizing Outbox %s to %s\n", id, status)
	return nil
}

// Domínio
func (r *PostgresRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	return &entity.Transaction{ID: id}, nil
}

func (r *PostgresRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	return nil
}

func (r *PostgresRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	fmt.Println("[sqlc] Starting Transaction Block")
	err := fn(ctx)
	if err != nil {
		fmt.Println("[sqlc] Rollback Transaction")
		return err
	}
	fmt.Println("[sqlc] Commit Transaction")
	return nil
}
