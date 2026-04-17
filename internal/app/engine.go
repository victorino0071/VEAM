package app

import (
	"context"
	"database/sql"
	"asaas_framework/internal/app/service"
	"asaas_framework/internal/app/worker"
	"asaas_framework/internal/domain/port"
	"asaas_framework/internal/domain/registry"
	"asaas_framework/internal/infra/repository"
	"asaas_framework/internal/infra/resilience"
	"asaas_framework/internal/infra/telemetry"
	"asaas_framework/internal/app/handler"
	"net/http"
	"time"
)

// Engine representa o coração do motor de pagamentos.
type Engine struct {
	Repo     port.Repository
	Registry *registry.ProviderRegistry
	Service  *service.PaymentService
	Consumer *worker.InboxConsumer
	Relay    *worker.OutboxRelay
	Breaker  port.CircuitBreaker
}

// NewEngine inicializa a fundação do motor sobre uma conexão Postgres.
func NewEngine(db *sql.DB) *Engine {
	repo := repository.NewPostgresRepository(db)
	reg := registry.NewProviderRegistry()
	
	// Circuit Breaker Padrão
	cb := resilience.NewCircuitBreaker(resilience.Config{
		FailureThreshold: 0.5,
		ResetTimeout:     10 * time.Second,
		Alpha:            0.2,
		MinRequests:      5,
	})

	svc := service.NewPaymentService(repo, reg)
	consumer := worker.NewInboxConsumer(repo, svc)
	relay := worker.NewOutboxRelay(repo, reg, cb)

	return &Engine{
		Repo:     repo,
		Registry: reg,
		Service:  svc,
		Consumer: consumer,
		Relay:    relay,
		Breaker:  cb,
	}
}

// WithTelemetry atacha a observabilidade no motor.
func (e *Engine) WithTelemetry(serviceName string) *Engine {
	_, _ = telemetry.InitTelemetry(serviceName)
	return e
}

// RegisterProvider acopla um novo gateway ao roteador lógico.
func (e *Engine) RegisterProvider(id string, adapter port.GatewayAdapter) *Engine {
	e.Registry.Register(id, adapter)
	return e
}

// NewWebhookHandler cria um handler configurado para um provedor específico.
func (e *Engine) NewWebhookHandler(providerID string) http.Handler {
	adapter, err := e.Registry.Get(providerID)
	if err != nil {
		panic(err) // No bootstrapping, erros de config devem ser fatais
	}
	return handler.NewWebhookHandler(e.Repo, adapter, providerID)
}

// Start dispara os processos de background (Inbox/Outbox).
func (e *Engine) Start(ctx context.Context) {
	go e.Consumer.Start(ctx)
	go e.Relay.Start(ctx)
}
