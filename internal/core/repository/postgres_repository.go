package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Victor/VEAM/domain/entity"
)

// DBTX abstrai as operações comuns entre *sql.DB e *sql.Tx
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type txKey struct{} // Chave privada para evitar colisões no context

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// exec decide qual executor usar: a transação no contexto ou o pool global (auto-commit).
func (r *PostgresRepository) exec(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return r.db
}

func (r *PostgresRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO inbox (id, external_id, event_type, payload, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id) DO UPDATE 
		SET payload = EXCLUDED.payload, metadata = EXCLUDED.metadata, updated_at = NOW()
		WHERE inbox.updated_at < NOW();`
	_, err = r.exec(ctx).ExecContext(ctx, query, event.ID, event.ExternalID, event.EventType, event.Payload, metadataJSON)
	return err
}

func (r *PostgresRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}

	query := `INSERT INTO outbox (id, event_type, payload, metadata) VALUES ($1, $2, $3, $4)`
	_, err = r.exec(ctx).ExecContext(ctx, query, event.ID, event.EventType, event.Payload, metadataJSON)
	return err
}

func (r *PostgresRepository) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) {
	query := `
		UPDATE inbox
		SET status = 'PROCESSING',
			updated_at = NOW(),
			-- Punição: Se está sendo resgatado do limbo, conta como uma tentativa
			retry_count = CASE WHEN status = 'PROCESSING' THEN retry_count + 1 ELSE retry_count END
		WHERE id IN (
			SELECT id FROM inbox
			WHERE 
				-- Fluxo Normal
				status = 'PENDING'
				OR 
				-- Fluxo de Autocura (Resgate de Órfãos)
				(status = 'PROCESSING' AND updated_at < NOW() - INTERVAL '5 MINUTES')
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, external_id, event_type, payload, metadata, status, retry_count, last_error, created_at;`
	
	rows, err := r.exec(ctx).QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.InboxEvent
	for rows.Next() {
		e := &entity.InboxEvent{}
		var metadataJSON []byte
		var lastError sql.NullString
		err := rows.Scan(&e.ID, &e.ExternalID, &e.EventType, &e.Payload, &metadataJSON, &e.Status, &e.RetryCount, &lastError, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		if lastError.Valid {
			e.LastError = &lastError.String
		}
		if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresRepository) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	query := `
		UPDATE outbox
		SET status = 'PROCESSING',
			updated_at = NOW(),
			-- Punição: Se está sendo resgatado do limbo, conta como uma tentativa
			retry_count = CASE WHEN status = 'PROCESSING' THEN retry_count + 1 ELSE retry_count END
		WHERE id IN (
			SELECT id FROM outbox
			WHERE 
				-- Fluxo Normal
				(status = 'PENDING' AND created_at > NOW() - INTERVAL '48 HOURS')
				OR 
				-- Fluxo de Autocura (Resgate de Órfãos)
				(status = 'PROCESSING' AND updated_at < NOW() - INTERVAL '5 MINUTES')
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, event_type, payload, metadata, status, retry_count, last_error, created_at;`

	rows, err := r.exec(ctx).QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		e := &entity.OutboxEvent{}
		var metadataJSON []byte
		var lastError sql.NullString
		err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &metadataJSON, &e.Status, &e.RetryCount, &lastError, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		if lastError.Valid {
			e.LastError = &lastError.String
		}
		if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresRepository) MarkInboxCompleted(ctx context.Context, id string) error {
	query := `UPDATE inbox SET status = 'COMPLETED', processed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) MarkInboxFailed(ctx context.Context, id string, errStr string) error {
	query := `
		UPDATE inbox
		SET status = 'PENDING',
			retry_count = retry_count + 1,
			last_error = $2,
			updated_at = NOW()
		WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id, errStr)
	return err
}

func (r *PostgresRepository) MoveInboxToDLQ(ctx context.Context, id string, errStr string) error {
	query := `
		UPDATE inbox
		SET status = 'DLQ',
			last_error = $2,
			updated_at = NOW()
		WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id, errStr)
	return err
}

func (r *PostgresRepository) MarkOutboxCompleted(ctx context.Context, id string) error {
	query := `UPDATE outbox SET status = 'COMPLETED', processed_at = NOW() WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) MarkOutboxFailed(ctx context.Context, id string, errStr string) error {
	query := `UPDATE outbox SET status = 'PENDING', retry_count = retry_count + 1, last_error = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id, errStr)
	return err
}

func (r *PostgresRepository) MoveOutboxToDLQ(ctx context.Context, id string, errStr string) error {
	query := `UPDATE outbox SET status = 'DLQ', last_error = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.exec(ctx).ExecContext(ctx, query, id, errStr)
	return err
}

func (r *PostgresRepository) ReplayInboxDLQ(ctx context.Context, id string) error {
	query := `UPDATE inbox SET status = 'PENDING', retry_count = 0, last_error = NULL, updated_at = NOW() WHERE id = $1 AND status = 'DLQ'`
	_, err := r.exec(ctx).ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) ReplayOutboxDLQ(ctx context.Context, id string) error {
	query := `UPDATE outbox SET status = 'PENDING', retry_count = 0, last_error = NULL, updated_at = NOW() WHERE id = $1 AND status = 'DLQ'`
	_, err := r.exec(ctx).ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	query := `SELECT id, customer_id, provider_id, amount, currency, status, description, due_date, created_at, updated_at FROM transactions WHERE id = $1 FOR UPDATE`
	row := r.exec(ctx).QueryRowContext(ctx, query, id)
	
	var s entity.TransactionSnapshot
	err := row.Scan(&s.ID, &s.CustomerID, &s.ProviderID, &s.Amount, &s.Currency, &s.Status, &s.Description, &s.DueDate, &s.CreatedAt, &s.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil 
	}
	if err != nil {
		return nil, err
	}

	// Reconstroi via Snapshot preservando as políticas de domínio e isolamento de memória
	return entity.RestoreTransaction(s), nil
}

func (r *PostgresRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	s := tx.ToSnapshot()
	query := `
		INSERT INTO transactions (id, customer_id, provider_id, amount, currency, status, description, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE 
		SET status = EXCLUDED.status, updated_at = NOW();`
	_, err := r.exec(ctx).ExecContext(ctx, query, s.ID, s.CustomerID, s.ProviderID, s.Amount, s.Currency, s.Status, s.Description, s.DueDate, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *PostgresRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Propaga a transação via Context
	txCtx := context.WithValue(ctx, txKey{}, tx)

	err = fn(txCtx)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit()
}
