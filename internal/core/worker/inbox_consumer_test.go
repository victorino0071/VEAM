package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/domain/registry"
	"github.com/Victor/payment-engine/internal/core/service"
)

// MockRepository com RWMutex e Isolamento Físico via Snapshots
type MockRepository struct {
	mu           sync.RWMutex
	transactions map[string]entity.TransactionSnapshot
	outbox       []*entity.OutboxEvent
	completed    []string
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		transactions: make(map[string]entity.TransactionSnapshot),
	}
}

func (m *MockRepository) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap, ok := m.transactions[id]
	if !ok {
		return nil, nil
	}
	return entity.RestoreTransaction(snap), nil
}

func (m *MockRepository) SaveTransaction(ctx context.Context, tx *entity.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[tx.ID] = tx.ToSnapshot() // Armazenamento físico serializado
	return nil
}

func (m *MockRepository) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outbox = append(m.outbox, event)
	return nil
}

func (m *MockRepository) MarkInboxCompleted(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed = append(m.completed, id)
	return nil
}

func (m *MockRepository) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (m *MockRepository) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error         { return nil }
func (m *MockRepository) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) { return nil, nil }
func (m *MockRepository) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	return nil, nil
}
func (m *MockRepository) MarkInboxFailed(ctx context.Context, id string) error     { return nil }
func (m *MockRepository) MarkOutboxCompleted(ctx context.Context, id string) error { return nil }
func (m *MockRepository) MarkOutboxFailed(ctx context.Context, id string) error    { return nil }

// MockAdapter para testar a inversão de dependência (Arquitetura Hexagonal)
type MockAdapter struct{}

func (m *MockAdapter) TranslatePayload(payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	var data struct {
		ID     string  `json:"id" `
		Status string  `json:"status" `
		Value  float64 `json:"value" `
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, "", err
	}

	tx := entity.NewTransaction(data.ID, "cust_1", "mock", data.Value, "test", time.Now())
	
	statusMapping := map[string]entity.PaymentStatus{
		"CONFIRMED": entity.StatusReceived,
		"PENDING":   entity.StatusPending,
	}

	return tx, statusMapping[data.Status], nil
}

func (m *MockAdapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) { return "", nil }
func (m *MockAdapter) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) { return "", nil }
func (m *MockAdapter) GetTransactionState(ctx context.Context, id string) (entity.PaymentStatus, error) {
	return "", nil
}
func (m *MockAdapter) RefundTransaction(ctx context.Context, id string) error { return nil }
func (m *MockAdapter) ValidateWebhook(r *http.Request) (bool, error)       { return true, nil }
func (m *MockAdapter) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return nil, nil
}

func TestInboxConsumer_ProcessEvent_StatusMapping(t *testing.T) {
	repo := NewMockRepository()
	reg := registry.NewProviderRegistry()
	svc := service.NewPaymentService(repo, reg)
	consumer := NewInboxConsumer(repo, svc, reg)

	// REGISTRO OBRIGATÓRIO (Prova de Desacoplamento Hexagonal)
	reg.Register("asaas", &MockAdapter{})

	ctx := context.Background()

	t.Run("Deve_Processar_Pagamento_Confirmado_Corretamente", func(t *testing.T) {
		payload := map[string]interface{}{
			"id":     "pay_123",
			"status": "CONFIRMED",
			"value":  100.0,
		}
		bytes, _ := json.Marshal(payload)

		event := &entity.InboxEvent{
			ID:      "evt_1",
			Payload: bytes,
			Metadata: map[string]string{
				"provider_id": "asaas",
			},
		}

		success := consumer.processEvent(ctx, event)
		if !success {
			t.Fatal("Esperado sucesso no processamento")
		}

		tx, _ := repo.GetTransactionByID(ctx, "pay_123")
		if tx.Status() != entity.StatusReceived {
			t.Errorf("Status esperado RECEIVED, obtido: %s", tx.Status())
		}

		repo.mu.RLock()
		if len(repo.outbox) != 1 {
			t.Errorf("Esperado 1 evento no outbox, obtido: %d", len(repo.outbox))
		}
		repo.mu.RUnlock()
	})

	t.Run("Idempotencia_Nao_Deve_Gerar_Outbox_Se_Status_Nao_Mudar", func(t *testing.T) {
		repo.mu.Lock()
		repo.outbox = nil
		repo.mu.Unlock()

		payload := map[string]interface{}{
			"id":     "pay_999",
			"status": "PENDING",
			"value":  100.0,
		}
		bytes, _ := json.Marshal(payload)

		event := &entity.InboxEvent{
			ID:      "evt_2",
			Payload: bytes,
			Metadata: map[string]string{
				"provider_id": "asaas",
			},
		}

		success := consumer.processEvent(ctx, event)
		if !success {
			t.Fatal("Esperado sucesso")
		}

		repo.mu.RLock()
		if len(repo.outbox) != 0 {
			t.Errorf("Não deveria gerar evento de outbox para status redundante, obtido: %d", len(repo.outbox))
		}
		repo.mu.RUnlock()
	})

	t.Run("Falha_Se_Adaptador_Nao_Registrado", func(t *testing.T) {
		event := &entity.InboxEvent{
			ID:      "evt_3",
			Payload: []byte("{}"),
			Metadata: map[string]string{
				"provider_id": "stripe", // Não registrado!
			},
		}

		success := consumer.processEvent(ctx, event)
		if success {
			t.Error("Deveria ter falhado pois o adaptador stripe não existe no registro")
		}
	})
}
