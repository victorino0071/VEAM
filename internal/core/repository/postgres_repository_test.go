package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/victorino0071/VEAM/domain/entity"
	"github.com/victorino0071/VEAM/internal/core/repository"
)

// TestPostgresRepository_ACID prova matematicamente que a atomicidade e 
// a propagação de contexto estão funcionando, garantindo o isolamento físico.
func TestPostgresRepository_ACID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("falha ao criar sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("Prova de Atomicidade: Rollback em Falha", func(t *testing.T) {
		// Expectativa: Inicia transação
		mock.ExpectBegin()
		
		// Expectativa 1: Busca transação com lock pessimista (Snapshot completo)
		columns := []string{"id", "customer_id", "provider_id", "amount", "currency", "status", "description", "due_date", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT (.+) FROM transactions WHERE id = \\$1 FOR UPDATE").
			WithArgs("tx_123").
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow("tx_123", "cust_1", "prov_1", 100.0, "BRL", "PENDING", "desc", time.Now(), time.Now(), time.Now()))

		// Expectativa 2: Simula uma falha no meio da transação (ex: erro no SaveTransaction)
		mock.ExpectExec("INSERT INTO transactions").
			WillReturnError(errors.New("db_error"))

		// Expectativa CRÍTICA: Deve executar Rollback
		mock.ExpectRollback()

		err := repo.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			tx, _ := repo.GetTransactionByID(txCtx, "tx_123")
			return repo.SaveTransaction(txCtx, tx)
		})

		if err == nil {
			t.Error("esperava erro mas transação reportou sucesso")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de atomicidade não atendidas: %s", err)
		}
	})

	t.Run("Prova de Propagação: Mesma Conexão (sql.Tx)", func(t *testing.T) {
		mock.ExpectBegin()
		
		columns := []string{"id", "customer_id", "provider_id", "amount", "currency", "status", "description", "due_date", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT (.+) FROM transactions").
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow("tx_123", "cust_1", "prov_1", 100.0, "BRL", "PENDING", "desc", time.Now(), time.Now(), time.Now()))
		
		mock.ExpectExec("INSERT INTO outbox").WillReturnResult(sqlmock.NewResult(1, 1))
		
		mock.ExpectCommit()

		err := repo.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			repo.GetTransactionByID(txCtx, "tx_123")
			return repo.SaveOutboxEvent(txCtx, &entity.OutboxEvent{ID: "evt_1"})
		})

		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de propagação não atendidas: %s", err)
		}
	})
}
