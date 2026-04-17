package mock

import (
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"context"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// MockRule (L2) define um predicado determinístico para injetar respostas.
type MockRule func(ctx context.Context, tx *entity.Transaction) *MockResponse

// MockResponse encapsula o resultado de uma operação simulada.
type MockResponse struct {
	ExternalID string
	Status     entity.PaymentStatus
	Err        error
}

// MockProvider é um motor de simulação de gateway de camada tripla (L1, L2, L3).
type MockProvider struct {
	mu        sync.RWMutex
	overrides map[string]*MockResponse // L1: Magic Overrides (ID-based)
	rules     []MockRule               // L2: Predicados Determinísticos
	chaosRate float64                  // L3: Taxa de falha (0.0 - 1.0)
	jitter    time.Duration            // L3: Latência base
}

func NewMockProvider(chaosRate float64, jitter time.Duration) *MockProvider {
	return &MockProvider{
		overrides: make(map[string]*MockResponse),
		rules:     make([]MockRule, 0),
		chaosRate: chaosRate,
		jitter:    jitter,
	}
}

// RegisterOverride (L1) injeta uma resposta imediata para um ID específico. Thread-safe.
func (m *MockProvider) RegisterOverride(id string, resp *MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.overrides[id] = resp
}

// AddRule (L2) adiciona um predicado de avaliação lógica.
func (m *MockProvider) AddRule(rule MockRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, rule)
}

// evaluate percorre a árvore de decisão L1 -> L2 -> L3.
func (m *MockProvider) evaluate(ctx context.Context, tx *entity.Transaction) (string, error) {
	// L1: Magic Overrides
	m.mu.RLock()
	if resp, ok := m.overrides[tx.ID]; ok {
		m.mu.RUnlock()
		return resp.ExternalID, resp.Err
	}
	
	// L2: Predicates
	for _, rule := range m.rules {
		if resp := rule(ctx, tx); resp != nil {
			m.mu.RUnlock()
			return resp.ExternalID, resp.Err
		}
	}
	m.mu.RUnlock()

	// L3: Chaos Engine
	if m.jitter > 0 {
		select {
		case <-time.After(m.jitter):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if rand.Float64() < m.chaosRate {
		return "", errors.New("chaos_engine: simulated network failure")
	}

	return "ext_" + tx.ID, nil
}

// Satisfy GatewayAdapter interface
func (m *MockProvider) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	return m.evaluate(ctx, tx)
}

func (m *MockProvider) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	// Simulação simples para clientes
	return "cus_mock_" + customer.Document, nil
}

func (m *MockProvider) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	return entity.StatusPending, nil
}

func (m *MockProvider) RefundTransaction(ctx context.Context, transactionID string) error {
	return nil
}

func (m *MockProvider) ValidateWebhook(r *http.Request) (bool, error) {
	return true, nil
}

func (m *MockProvider) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return &port.WebhookResponse{}, nil
}
