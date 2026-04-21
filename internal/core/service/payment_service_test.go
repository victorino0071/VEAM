package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/registry"
)

// MockRepository com Isolamento Físico e Sincronização Estrita (RWMutex)
type MockRepository struct {
	mu           sync.RWMutex
	transactions map[string]*entity.Transaction
	outbox       []*entity.OutboxEvent
	shouldFail   bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		transactions: make(map[string]*entity.Transaction),
		outbox:       make([]*entity.OutboxEvent, 0),
	}
}

func (m *MockRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tx, ok := m.transactions[id]
	if !ok {
		return nil, nil
	}
	// Importante: Retornar uma cópia (Restore) para simular isolamento de memória do BD
	return entity.RestoreTransaction(tx.ToSnapshot()), nil
}

func (m *MockRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldFail {
		return fmt.Errorf("erro simulação banco")
	}
	m.transactions[tx.ID] = tx
	return nil
}

func (m *MockRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outbox = append(m.outbox, event)
	return nil
}

func (m *MockRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Em um mock de memória, simplificamos: se a função retornar erro, não persistimos mudanças complexas.
	// Para este teste, vamos apenas rodar a função. Se o Repository fosse mais complexo, 
	// usaríamos um 'shadow map' para rollbacks.
	return fn(ctx)
}

// Métodos obrigatórios da interface port.Repository (No-Op para este teste)
func (m *MockRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error           { return nil }
func (m *MockRepository) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error)   { return nil, nil }
func (m *MockRepository) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) { return nil, nil }
func (m *MockRepository) MarkInboxCompleted(ctx context.Context, id string) error                     { return nil }
func (m *MockRepository) MarkInboxFailed(ctx context.Context, id string) error                        { return nil }
func (m *MockRepository) MarkOutboxCompleted(ctx context.Context, id string) error                    { return nil }
func (m *MockRepository) MarkOutboxFailed(ctx context.Context, id string) error                       { return nil }

func TestPaymentService_ProcessPayment_ACID_Memory(t *testing.T) {
	repo := NewMockRepository()
	reg := registry.NewProviderRegistry()
	svc := NewPaymentService(repo, reg)

	ctx := context.Background()
	txID := "tx-123"
	
	// 1. Cenário: Criar transação inicial
	initialTx := entity.RestoreTransaction(entity.TransactionSnapshot{
		ID:     txID,
		Status: entity.StatusPending,
		Amount: 100.0,
	})
	
	metadata := map[string]string{"reason": "webhook_received"}

	// 2. Execução de Sucesso: Transição PENDING -> PAID
	err := svc.ProcessPaymentWithMetadata(ctx, initialTx, metadata, entity.StatusPaid)
	if err != nil {
		t.Fatalf("Erro inesperado no processamento: %v", err)
	}

	// Validação de Persistência
	tx, _ := repo.GetTransactionByID(ctx, txID)
	if tx.Status() != entity.StatusPaid {
		t.Errorf("Status esperado PAID, obtido: %s", tx.Status())
	}

	// Validação de Outbox
	repo.mu.RLock()
	if len(repo.outbox) != 1 {
		t.Errorf("Esperado 1 evento no outbox, obtido: %d", len(repo.outbox))
	} else if repo.outbox[0].EventType != "PAYMENT_CONFIRMED" {
		t.Errorf("Tipo de evento incorreto: %s", repo.outbox[0].EventType)
	}
	repo.mu.RUnlock()

	// 3. Cenário: Falha de Banco (Simular Rollback Lógico)
	repo.shouldFail = true
	// Tentativa de transição PAID -> REFUNDED (deve falhar no SaveTransaction)
	err = svc.ProcessPaymentWithMetadata(ctx, tx, metadata, entity.StatusRefunded)
	if err == nil {
		t.Error("Esperado erro de banco, mas a operação reportou sucesso")
	}

	// O status não deve ter mudado se o serviço propagou o erro corretamente
	txVerify, _ := repo.GetTransactionByID(ctx, txID)
	if txVerify.Status() != entity.StatusPaid {
		t.Errorf("Status deveria continuar PAID após erro de persistência, obtido: %s", txVerify.Status())
	}
}
