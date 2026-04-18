package mockprovider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Victor/payment-engine/adapters/mockprovider"
	"github.com/Victor/payment-engine/domain/entity"
)

func TestMockProvider_ResolutionHierarchy(t *testing.T) {
	ctx := context.Background()

	// Configura Mock com Caos 100% (L3 falharia sempre)
	mockProvider := mockprovider.NewAdapter(1.0, 0)

	// L2: Regra que aprova transações "VIP"
	mockProvider.Rules = append(mockProvider.Rules, func(ctx context.Context, tx *entity.Transaction) *mockprovider.Response {
		if tx.CustomerID == "customer_vip" {
			return &mockprovider.Response{ExternalID: "ext_vip_ok", Err: nil}
		}
		return nil
	})

	t.Run("L2 deve vencer L3 (Caos)", func(t *testing.T) {
		tx := &entity.Transaction{ID: "tx_1", CustomerID: "customer_vip"}
		id, err := mockProvider.CreateTransaction(ctx, tx)

		if err != nil {
			t.Errorf("L2 deveria ter interceptado a falha do L3: %v", err)
		}
		if id != "ext_vip_ok" {
			t.Errorf("ID incorreto, esperado ext_vip_ok, obtido %s", id)
		}
	})

	t.Run("L1 (Override) deve vencer L2", func(t *testing.T) {
		// Injetamos um override L1 para o mesmo ID
		mockProvider.RegisterOverride("tx_override", &mockprovider.Response{
			ExternalID: "ext_l1_priority",
			Err:        errors.New("forced_l1_error"),
		})

		tx := &entity.Transaction{ID: "tx_override", CustomerID: "customer_vip"}
		id, err := mockProvider.CreateTransaction(ctx, tx)

		if err == nil || err.Error() != "forced_l1_error" {
			t.Errorf("L1 deveria ter prioridade absoluta: erro esperado 'forced_l1_error', obtido %v", err)
		}
		if id != "ext_l1_priority" {
			t.Errorf("ID de L1 esperado, obtido %s", id)
		}
	})

	t.Run("L3 (Chaos) deve atuar se L1 e L2 falharem", func(t *testing.T) {
		tx := &entity.Transaction{ID: "tx_normal", CustomerID: "customer_normal"}
		_, err := mockProvider.CreateTransaction(ctx, tx)

		if err == nil || err.Error() != "chaos_engine: simulated network failure" {
			t.Errorf("L3 deveria ter falhado (100%% chaos): obtido %v", err)
		}
	})
}

func TestMockProvider_ConcurrencySafe(t *testing.T) {
	mockProvider := mockprovider.NewAdapter(0.1, 10*time.Millisecond)

	t.Run("Parallel access to overrides", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 100; i++ {
			go func(val int) {
				mockProvider.RegisterOverride("tx_id", &mockprovider.Response{ExternalID: "ok"})
				_, _ = mockProvider.CreateTransaction(context.Background(), &entity.Transaction{ID: "tx_id"})
			}(i)
		}
	})
}
