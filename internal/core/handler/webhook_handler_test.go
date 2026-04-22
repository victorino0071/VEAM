package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/victorino0071/VEAM/domain/entity"
	"github.com/victorino0071/VEAM/domain/port"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

type mockHandlerRepo struct {
	lastEvent  *entity.InboxEvent
	forceError error
}

func (m *mockHandlerRepo) SaveInboxEvent(ctx context.Context, event *entity.InboxEvent) error {
	if m.forceError != nil {
		return m.forceError
	}
	m.lastEvent = event
	return nil
}
func (m *mockHandlerRepo) SaveOutboxEvent(ctx context.Context, event *entity.OutboxEvent) error { return nil }
func (m *mockHandlerRepo) ClaimInboxEvents(ctx context.Context, limit int) ([]*entity.InboxEvent, error) { return nil, nil }
func (m *mockHandlerRepo) ClaimOutboxEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) { return nil, nil }
func (m *mockHandlerRepo) MarkInboxCompleted(ctx context.Context, id string) error { return nil }
func (m *mockHandlerRepo) MarkInboxFailed(ctx context.Context, id string, errStr string) error    { return nil }
func (m *mockHandlerRepo) MoveInboxToDLQ(ctx context.Context, id string, errStr string) error { return nil }
func (m *mockHandlerRepo) MarkOutboxCompleted(ctx context.Context, id string) error { return nil }
func (m *mockHandlerRepo) MarkOutboxFailed(ctx context.Context, id string, errStr string) error    { return nil }
func (m *mockHandlerRepo) MoveOutboxToDLQ(ctx context.Context, id string, errStr string) error { return nil }
func (m *mockHandlerRepo) ReplayInboxDLQ(ctx context.Context, id string) error { return nil }
func (m *mockHandlerRepo) ReplayOutboxDLQ(ctx context.Context, id string) error { return nil }
func (m *mockHandlerRepo) GetTransactionByID(ctx context.Context, id string) (*entity.Transaction, error) { return nil, nil }
func (m *mockHandlerRepo) SaveTransaction(ctx context.Context, tx *entity.Transaction) error { return nil }
func (m *mockHandlerRepo) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type mockHandlerAdapter struct{}

func (m *mockHandlerAdapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) { return "", nil }
func (m *mockHandlerAdapter) CreateTransaction(ctx context.Context, transaction *entity.Transaction) (string, error) { return "", nil }
func (m *mockHandlerAdapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) { return "", nil }
func (m *mockHandlerAdapter) RefundTransaction(ctx context.Context, txID string, idempotencyKey string) error {
	return nil
}
func (m *mockHandlerAdapter) ValidateWebhook(r *http.Request) (bool, error) { return true, nil }
func (m *mockHandlerAdapter) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	return &port.WebhookResponse{
		ExternalID: "ext-webhook-123",
		EventType:  "PAYMENT_CONFIRMED",
		Payload:    []byte(`{"status":"paid"}`),
	}, nil
}
func (m *mockHandlerAdapter) TranslatePayload(ctx context.Context, payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	return nil, "", nil
}
func (m *mockHandlerAdapter) Fingerprint(payload []byte) (string, error) {
	return "static-fingerprint", nil
}


func TestWebhookHandler_OTel_Trace_Propagation(t *testing.T) {
	// 1. Setup OpenTelemetry (Trace Provider + Propagator)
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	repo := &mockHandlerRepo{}
	adapter := &mockHandlerAdapter{}
	h := NewWebhookHandler(repo, adapter, "test-provider")

	// 2. Criar Request com cabeçalho Traceparent (W3C Standard)
	// format: 00-<traceID>-<parentID>-01
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	traceparent := "00-" + traceID + "-00f067aa0ba902b7-01"
	
	req := httptest.NewRequest("POST", "/webhooks/asaas", strings.NewReader("{}"))
	req.Header.Set("traceparent", traceparent)
	
	rr := httptest.NewRecorder()

	// 3. Execução
	h.ServeHTTP(rr, req)

	// 4. Validação de Status
	if rr.Code != http.StatusAccepted {
		t.Errorf("Esperado status 202 Accepted, obtido %d", rr.Code)
	}

	// 5. VALIDAÇÃO DE OBSERVABILIDADE (O Ponto Cego)
	if repo.lastEvent == nil {
		t.Fatal("Evento de Inbox não foi persistido")
	}

	// Verificamos se o TraceID injetado no Header chegou ao metadados do evento persistido
	// O OTel SDK injeta o traceparent no carrier quando chamamos Propagator.Inject
	storedTraceparent := repo.lastEvent.Metadata["traceparent"]
	if !strings.Contains(storedTraceparent, traceID) {
		t.Errorf("TraceID não propagado corretamente para o Metadata. \nEsperado conter: %s \nObtido no Metadata: %s", traceID, storedTraceparent)
	}

	if repo.lastEvent.Metadata["provider_id"] != "test-provider" {
		t.Errorf("ProviderID incorreto: %s", repo.lastEvent.Metadata["provider_id"])
	}
}

func TestWebhookHandler_Fingerprint_Deduplication(t *testing.T) {
	repo := &mockHandlerRepo{}
	adapter := &mockHandlerAdapter{}
	h := NewWebhookHandler(repo, adapter, "test-provider")

	// 1. Primeira Ingestão (Sucesso)
	req1 := httptest.NewRequest("POST", "/webhooks/asaas", strings.NewReader(`{"status":"paid"}`))
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusAccepted {
		t.Errorf("Esperado status 202 na primeira ingestão, obtido %d", rr1.Code)
	}

	// 2. Segunda Ingestão (Mesmo Fingerprint - Colisão no Repo)
	repo.forceError = entity.ErrDuplicateFingerprint
	req2 := httptest.NewRequest("POST", "/webhooks/asaas", strings.NewReader(`{"status":"paid"}`))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	// Validação Crucial: Deve retornar 202 Accepted (Silenciamento) e não 500 ou 409.
	if rr2.Code != http.StatusAccepted {
		t.Errorf("Esperado status 202 na detecção de duplicata (silenciamento), obtido %d", rr2.Code)
	}

	if !strings.Contains(rr2.Body.String(), "Evento já processado") {
		t.Errorf("Mensagem de corpo inesperada: %s", rr2.Body.String())
	}
}
