package worker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/domain/registry"
	"github.com/Victor/payment-engine/internal/core/resilience"
)

type mockGateway struct {
	shouldFail int64 // Atômico
}

func (m *mockGateway) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	return "ext-cust-123", nil
}
func (m *mockGateway) CreateTransaction(ctx context.Context, transaction *entity.Transaction) (string, error) {
	return "ext-tx-123", nil
}
func (m *mockGateway) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	return entity.StatusPaid, nil
}
func (m *mockGateway) RefundTransaction(ctx context.Context, txID string) error {
	if atomic.LoadInt64(&m.shouldFail) == 1 {
		return errors.New("provider failure")
	}
	return nil
}

func (m *mockGateway) ValidateWebhook(r *http.Request) (bool, error) { return true, nil }
func (m *mockGateway) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return &port.WebhookResponse{ExternalID: "ext-1", EventType: "PAYMENT_RECEIVED", Payload: []byte("{}")}, nil
}
func (m *mockGateway) TranslatePayload(payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	return nil, "", nil
}


// Mock de repositório focado apenas no Outbox Relay
type mockRelayRepo struct {
	mu            sync.Mutex
	events        []*entity.OutboxEvent
	completedIDs  []string
	failedIDs     []string
}

func (m *mockRelayRepo) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error { return nil }
func (m *mockRelayRepo) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error { return nil }
func (m *mockRelayRepo) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) {
	return nil, nil
}
func (m *mockRelayRepo) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) {
	return nil, nil
}
func (m *mockRelayRepo) SaveTransaction(ctx context.Context, tx *entity.Transaction) error { return nil }
func (m *mockRelayRepo) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (m *mockRelayRepo) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := limit
	if len(m.events) < limit {
		count = len(m.events)
	}

	batch := m.events[:count]
	m.events = m.events[count:]
	return batch, nil
}

func (m *mockRelayRepo) MarkInboxCompleted(ctx context.Context, id string) error { return nil }
func (m *mockRelayRepo) MarkInboxFailed(ctx context.Context, id string) error    { return nil }

func (m *mockRelayRepo) MarkOutboxCompleted(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completedIDs = append(m.completedIDs, id)
	return nil
}

func (m *mockRelayRepo) MarkOutboxFailed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedIDs = append(m.failedIDs, id)
	return nil
}

func TestOutboxRelay_Sociable_Resilience(t *testing.T) {
	ctx := context.Background()

	// 1. Configuração do Motor de Resiliência Real (Hiper-Agressivo)
	breaker := resilience.NewCircuitBreaker(resilience.Config{
		FailureThreshold: 0.1, // Abre com 10% de falha
		ResetTimeout:     1 * time.Second,
		Alpha:            1.0, // Reatividade imediata
		MinRequests:      1,
	})

	// 2. Setup do Registro e Gateway
	reg := registry.NewProviderRegistry()
	gateway := &mockGateway{}
	reg.Register("provider-1", gateway)

	// 3. Setup do Repositório com 10 eventos
	repo := &mockRelayRepo{
		events: make([]*entity.OutboxEvent, 10),
	}
	for i := 0; i < 10; i++ {
		repo.events[i] = &entity.OutboxEvent{
			ID:        fmt.Sprintf("evt-%d", i),
			EventType: "REFUND_STARTED",
			Payload:   []byte("tx-123"),
			Metadata:  map[string]string{"provider_id": "provider-1"},
		}
	}

	relay := NewOutboxRelay(repo, reg, breaker)

	// --- CENÁRIO A: Backpressure (Shrinking Batch) ---
	// Forçamos uma falha manual no breaker para elevar P(F)
	atomic.StoreInt64(&gateway.shouldFail, 1)

	// Simulamos uma execução manual de batch para ver o encolhimento
	// Primeiro, uma falha real via consumeBatch para o breaker registrar
	relay.consumeBatch(ctx, 1)

	pf, _ := breaker.GetFailureProbability(ctx)
	if pf != 1.0 {
		t.Errorf("Esperado P(F)=1.0 após falha atómica, obtido: %f", pf)
	}

	// --- CENÁRIO B: Intra-Batch Fail-Fast ---
	// Recarregamos o repo
	repo.mu.Lock()
	repo.events = make([]*entity.OutboxEvent, 5)
	for i := 0; i < 5; i++ {
		repo.events[i] = &entity.OutboxEvent{
			ID:        fmt.Sprintf("evt-new-%d", i),
			EventType: "REFUND_STARTED",
			Payload:   []byte("tx-123"),
			Metadata:  map[string]string{"provider_id": "provider-1"},
		}
	}
	repo.mu.Unlock()

	// Executamos um batch de 5. No primeiro evento, o breaker já deve estar OPEN.
	// O Relay deve abortar o processamento dos próximos 4.
	processed := relay.consumeBatch(ctx, 5)

	// Apenas 1 deve ter sido tentado (e falhado), os outros foram abortados pelo Fail-Fast
	if processed != 5 {
		t.Errorf("Deveria ter 'revidicado' 5 eventos, processou %d", processed)
	}

	repo.mu.Lock()
	if len(repo.failedIDs) != 1 {
		t.Errorf("Esperado apenas 1 falha marcada no banco devido ao Fail-Fast, obtido: %d", len(repo.failedIDs))
	}
	repo.mu.Unlock()
}
