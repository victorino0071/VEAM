package entity_test

import (
	"context"
	"testing"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
)

// mockPolicy simula uma regra de negócio externa (fraude, limites, etc.)
type mockPolicy struct {
	id     string
	err    error
	called bool
}

func (m *mockPolicy) ID() string { return m.id }
func (m *mockPolicy) Evaluate(ctx context.Context, tx *entity.Transaction, target entity.PaymentStatus) error {
	m.called = true
	return m.err
}

// VETOR A & B: Teste de Transições da FSM e Chain of Responsibility
func TestTransaction_TransitionTo(t *testing.T) {
	ctx := context.Background()
	dueDate := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name          string
		initialStatus entity.PaymentStatus
		targetStatus  entity.PaymentStatus
		policies      []entity.TransitionPolicy
		wantErr       bool
		wantEvent     string
	}{
		{
			name:          "Sucesso: PENDING para PAID (Fluxo Padrão)",
			initialStatus: entity.StatusPending,
			targetStatus:  entity.StatusPaid,
			wantErr:       false,
			wantEvent:     "PAYMENT_CONFIRMED",
		},
		{
			name:          "Bloqueio: PAID para FAILED (Violação FSM Padrão)",
			initialStatus: entity.StatusPaid,
			targetStatus:  entity.StatusFailed,
			wantErr:       true,
		},
		{
			name:          "Idempotência: PAID para PAID (Sem Erro, Sem Evento)",
			initialStatus: entity.StatusPaid,
			targetStatus:  entity.StatusPaid,
			wantErr:       false,
			wantEvent:     "",
		},
		{
			name:          "Aborto: Política de Risco Rejeita Transição",
			initialStatus: entity.StatusPending,
			targetStatus:  entity.StatusPaid,
			policies:      []entity.TransitionPolicy{&mockPolicy{id: "fraud_rule", err: entity.ErrIllegalTransition}},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup via RestoreTransaction (Fábrica Estática) para evitar mutação em runtime
			tx := entity.RestoreTransaction(entity.TransactionSnapshot{
				ID:          "tx_123",
				CustomerID:  "cust_1",
				ProviderID:  "prov_1",
				Status:      tt.initialStatus,
				Amount:      100.0,
				DueDate:     dueDate,
			})

			if len(tt.policies) > 0 {
				tx.WithPolicies(tt.policies...)
			}

			eventID := "evt_test_determinism"
			event, err := tx.TransitionTo(ctx, tt.targetStatus, eventID, nil)

			// Validação de Erro
			if (err != nil) != tt.wantErr {
				t.Errorf("TransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Validação de Estado (Se falhou, deve manter o inicial)
			if tt.wantErr && tx.Status() != tt.initialStatus {
				t.Errorf("Estado alterado após erro: got %v, want %v", tx.Status(), tt.initialStatus)
			}

			// Validação de Evento
			if !tt.wantErr && tt.wantEvent != "" {
				if event == nil {
					t.Fatal("Esperava evento de outbox, retornou nil")
				}
				if event.ID != eventID {
					t.Errorf("ID do evento não determinístico: got %s, want %s", event.ID, eventID)
				}
				if event.EventType != tt.wantEvent {
					t.Errorf("Tipo de evento incorreto: got %v, want %v", event.EventType, tt.wantEvent)
				}
			}
			if !tt.wantErr && tt.wantEvent == "" && event != nil {
				t.Error("Não esperava evento para transição idempotente")
			}
		} )
	}
}

// VETOR C: Teste de Barreira de Memória (Deep Copy do Memento)
func TestTransaction_MementoSecurity(t *testing.T) {
	dueDate := time.Now().Add(24 * time.Hour)
	paymentDate := time.Now()
	
	// 1. Setup via RestoreTransaction
	tx := entity.RestoreTransaction(entity.TransactionSnapshot{
		ID:          "tx_mem",
		PaymentDate: &paymentDate,
		DueDate:     dueDate,
	})

	// 2. Exporta Snapshot
	snapshot := tx.ToSnapshot()

	// 3. PROVA DE FOGO: Alterar o valor apontado no Snapshot NÃO pode afetar a Entidade
	newDate := paymentDate.Add(10 * time.Hour)
	*snapshot.PaymentDate = newDate

	if tx.ToSnapshot().PaymentDate.Equal(newDate) {
		t.Errorf("VULNERABILIDADE: Alteração no Snapshot mutou a entidade original (Pointer Leak)")
	}

	// 4. Inverso: RestoreTransaction deve realizar Deep Copy dos dados do Snapshot
	anotherDate := time.Now().Add(20 * time.Hour)
	snapToRestore := entity.TransactionSnapshot{
		ID:          "tx_restore",
		PaymentDate: &anotherDate,
	}
	
	txRestored := entity.RestoreTransaction(snapToRestore)

	// Alterar snapshot original após o restore
	*snapToRestore.PaymentDate = time.Now().Add(50 * time.Hour)

	if txRestored.ToSnapshot().PaymentDate.Equal(*snapToRestore.PaymentDate) {
		t.Errorf("VULNERABILIDADE: Alteração externa no Snapshot após Restore mutou a entidade")
	}
}
