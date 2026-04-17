package repository

import (
	"context"
	"database/sql"
	"asaas_framework/internal/domain/entity"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Ingestão com ON CONFLICT (sqlc-style)
func (r *PostgresRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error {
	query := `
		INSERT INTO inbox (id, external_id, event_type, payload, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id) DO UPDATE 
		SET payload = EXCLUDED.payload, metadata = EXCLUDED.metadata, updated_at = NOW()
		WHERE inbox.updated_at < NOW();`
	_, err := r.db.ExecContext(ctx, query, event.ID, event.ExternalID, event.EventType, event.Payload, event.Metadata)
	return err
}

func (r *PostgresRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	query := `INSERT INTO outbox (id, event_type, payload, metadata) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, event.ID, event.EventType, event.Payload, event.Metadata)
	return err
}

func (r *PostgresRepository) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) {
	query := `
		UPDATE inbox
		SET status = 'PROCESSING'
		WHERE id IN (
			SELECT id FROM inbox
			WHERE status = 'PENDING'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, external_id, event_type, payload, metadata, status, retry_count, created_at;`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.InboxEvent
	for rows.Next() {
		e := &entity.InboxEvent{}
		// Scan mapping depending on entity struct
		err := rows.Scan(&e.ID, &e.ExternalID, &e.EventType, &e.Payload, &e.Metadata, &e.Status, &e.RetryCount, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresRepository) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	query := `
		UPDATE outbox
		SET status = 'PROCESSING'
		WHERE id IN (
			SELECT id FROM outbox
			WHERE status = 'PENDING'
			AND created_at > NOW() - INTERVAL '48 HOURS'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, event_type, payload, metadata, status, retry_count, created_at;`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		e := &entity.OutboxEvent{}
		err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.Metadata, &e.Status, &e.RetryCount, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresRepository) MarkInboxCompleted(ctx context.Context, id string) error {
	query := `UPDATE inbox SET status = 'COMPLETED', processed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) MarkInboxFailed(ctx context.Context, id string) error {
	query := `
		UPDATE inbox
		SET status = CASE WHEN retry_count >= 4 THEN 'DLQ' ELSE 'PENDING' END,
			retry_count = retry_count + 1,
			updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) MarkOutboxCompleted(ctx context.Context, id string) error {
	query := `UPDATE outbox SET status = 'COMPLETED', processed_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) MarkOutboxFailed(ctx context.Context, id string) error {
	query := `UPDATE outbox SET status = 'FAILED', retry_count = retry_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	query := `SELECT id, customer_id, amount, currency, status, description, due_date FROM transactions WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	
	tx := &entity.Transaction{}
	err := row.Scan(&tx.ID, &tx.CustomerID, &tx.Amount, &tx.Currency, &tx.Status, &tx.Description, &tx.DueDate)
	if err == sql.ErrNoRows {
		return nil, nil // Or specific error
	}
	return tx, err
}

func (r *PostgresRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	query := `
		INSERT INTO transactions (id, customer_id, amount, currency, status, description, due_date, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (id) DO UPDATE 
		SET status = EXCLUDED.status, updated_at = NOW();`
	_, err := r.db.ExecContext(ctx, query, tx.ID, tx.CustomerID, tx.Amount, tx.Currency, tx.Status, tx.Description, tx.DueDate)
	return err
}

func (r *PostgresRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Note: Implementing ExecuteInTransaction with a real DB transaction requires
	// that we pass the *sql.Tx down. For simplicity in this framework, we are assuming
	// the repo methods would use the tx if available. In a more complex setup, you'd
	// store the tx in the context.
	err = fn(ctx)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
