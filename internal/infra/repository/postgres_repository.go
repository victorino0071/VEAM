package repository

import (
	"context"
	"database/sql"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"fmt"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) port.Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	// Exemplo de SQL: INSERT INTO transactions ... ON CONFLICT (id) DO UPDATE SET status = excluded.status
	fmt.Printf("[Postgres] Saving transaction ID: %s, Status: %s\n", tx.ID, tx.Status)
	return nil
}

func (r *PostgresRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	fmt.Printf("[Postgres] Reading transaction: %s\n", id)
	return &entity.Transaction{ID: id, Status: entity.StatusPending}, nil
}

func (r *PostgresRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	fmt.Printf("[Postgres] Saving Outbox Event: %s\n", event.EventType)
	return nil
}

func (r *PostgresRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error {
	// query := "INSERT INTO inbox (id, webhook_id, payload) VALUES ($1, $2, $3) ON CONFLICT (webhook_id) DO NOTHING"
	fmt.Printf("[Postgres] Saving Inbox Event: %s (ON CONFLICT DO NOTHING)\n", event.ExternalID)
	return nil
}

// FetchNextPendingInbox utiliza FOR UPDATE SKIP LOCKED para concorrência segura.
func (r *PostgresRepository) FetchNextPendingInbox(ctx context.Context, limit int) ([]*entity.InboxEvent, error) {
	// query := "SELECT * FROM inbox WHERE status = 'PENDING' FOR UPDATE SKIP LOCKED LIMIT $1"
	fmt.Println("[Postgres] Fetching Inbox with SKIP LOCKED...")
	return []*entity.InboxEvent{}, nil
}

func (r *PostgresRepository) FetchNextPendingOutbox(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	fmt.Println("[Postgres] Fetching Outbox with SKIP LOCKED...")
	return []*entity.OutboxEvent{}, nil
}

func (r *PostgresRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Inicia transação: tx, _ := r.db.BeginTx(ctx, nil)
	// Passa o contexto com o tx injetado (ou apenas o tx) para a função
	fmt.Println("[Postgres] Starting ACID Transaction...")
	
	err := fn(ctx)
	if err != nil {
		fmt.Println("[Postgres] Rolling back Transaction.")
		return err
	}
	
	fmt.Println("[Postgres] Committing Transaction.")
	return nil
}
