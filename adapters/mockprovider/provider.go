package mockprovider

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
)

// Response encapsula o resultado de uma operação simulada.
type Response struct {
	ExternalID string
	Status     entity.PaymentStatus
	Err        error
}

// MockRule (L2) define um predicado determinístico para injetar respostas.
type MockRule func(ctx context.Context, tx *entity.Transaction) *Response

// MockConfig é o motor de simulação de gateway de camada tripla (L1, L2, L3).
type MockConfig struct {
	ChaosRate  float64
	JitterBase time.Duration
	Rules      []MockRule // L2: Predicados Determinísticos

	mu        sync.RWMutex
	overrides map[string]*Response // L1: Magic Overrides (ID-based)
}

// NewAdapter inicializa o motor de mocking conforme o blueprint aprovado.
func NewAdapter(chaosRate float64, jitter time.Duration) *MockConfig {
	return &MockConfig{
		overrides: make(map[string]*Response),
		Rules:     make([]MockRule, 0),
		ChaosRate: chaosRate,
		JitterBase: jitter,
	}
}

// RegisterOverride (L1) injeta uma resposta imediata para um ID específico. Thread-safe.
func (m *MockConfig) RegisterOverride(id string, resp *Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.overrides[id] = resp
}

// evaluate percorre a árvore de decisão L1 -> L2 -> L3.
func (m *MockConfig) evaluate(ctx context.Context, tx *entity.Transaction) (string, error) {
	// L1: Magic Overrides
	m.mu.RLock()
	if resp, ok := m.overrides[tx.ID]; ok {
		m.mu.RUnlock()
		return resp.ExternalID, resp.Err
	}

	// L2: Predicates (Rules)
	for _, rule := range m.Rules {
		if resp := rule(ctx, tx); resp != nil {
			m.mu.RUnlock()
			return resp.ExternalID, resp.Err
		}
	}
	m.mu.RUnlock()

	// L3: Chaos Engine
	if m.JitterBase > 0 {
		select {
		case <-time.After(m.JitterBase):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if rand.Float64() < m.ChaosRate {
		return "", errors.New("chaos_engine: simulated network failure")
	}

	return "ext_" + tx.ID, nil
}

// Satisfy GatewayAdapter interface
func (m *MockConfig) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	return m.evaluate(ctx, tx)
}

func (m *MockConfig) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	return "cus_mock_" + customer.Document, nil
}

func (m *MockConfig) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	return entity.StatusPending, nil
}

func (m *MockConfig) RefundTransaction(ctx context.Context, transactionID string, idempotencyKey string) error {
	return nil
}

func (m *MockConfig) ValidateWebhook(r *http.Request) (bool, error) {
	return true, nil
}

func (m *MockConfig) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return &port.WebhookResponse{}, nil
}
