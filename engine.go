package veam

import (
	"context"
	"database/sql"
	"github.com/victorino0071/VEAM/internal/core/service"
	"github.com/victorino0071/VEAM/internal/core/worker"
	"github.com/victorino0071/VEAM/domain/port"
	"github.com/victorino0071/VEAM/domain/registry"
	"github.com/victorino0071/VEAM/internal/core/repository"
	"github.com/victorino0071/VEAM/internal/core/resilience"
	"github.com/victorino0071/VEAM/internal/core/telemetry"
	"github.com/victorino0071/VEAM/internal/core/acl"
	"github.com/victorino0071/VEAM/internal/core/handler"
	"net/http"
	"fmt"
	"time"
)

// Engine representa o coração do motor de pagamentos.
type Engine struct {
	Repo     port.Repository
	Registry *registry.ProviderRegistry
	Service  *service.PaymentService
	Consumer *worker.InboxConsumer
	Relay      *worker.OutboxRelay
	Breaker    port.CircuitBreaker
	db         *sql.DB
	MaxRetries int
}

// NewEngine inicializa a fundação do motor sobre uma conexão Postgres.
func NewEngine(db *sql.DB) *Engine {
	repo := repository.NewPostgresRepository(db)
	reg := registry.NewProviderRegistry()
	
	// Adapter Interno (SAGA)
	reg.Register("SYSTEM_INTERNAL", acl.NewInternalSystemAdapter(repo))
	
	// Circuit Breaker Padrão
	cb := resilience.NewCircuitBreaker(resilience.Config{
		FailureThreshold: 0.5,
		ResetTimeout:     10 * time.Second,
		Alpha:            0.2,
		MinRequests:      5,
	})

	svc := service.NewPaymentService(repo, reg)
	consumer := worker.NewInboxConsumer(repo, svc, reg, 5) // Defaults to 5
	relay := worker.NewOutboxRelay(repo, reg, cb, 5) // Defaults to 5

	return &Engine{
		Repo:     repo,
		Registry: reg,
		Service:  svc,
		Consumer: consumer,
		Relay:      relay,
		Breaker:    cb,
		db:         db,
		MaxRetries: 5,
	}
}

// WithMaxRetries define o limite de tentativas antes do DLQ
func (e *Engine) WithMaxRetries(limit int) *Engine {
	e.MaxRetries = limit
	e.Consumer.SetMaxRetries(limit)
	e.Relay.SetMaxRetries(limit)
	return e
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

// NewWebhookHandler cria um handler configurado para um provedor específico (Modo API).
func (e *Engine) NewWebhookHandler(providerID string) http.Handler {
	adapter, err := e.Registry.Get(providerID)
	if err != nil {
		panic(err)
	}
	return handler.NewWebhookHandler(e.Repo, adapter, providerID)
}

// ConsumeInbox inicia o loop de processamento de eventos recebidos (Modo Worker).
// Este método é bloqueante.
func (e *Engine) ConsumeInbox(ctx context.Context) error {
	e.Consumer.Start(ctx)
	return nil
}

// RelayOutbox inicia o despacho de eventos para o mundo exterior (Modo Worker).
// Este método é bloqueante.
func (e *Engine) RelayOutbox(ctx context.Context) error {
	e.Relay.Start(ctx)
	return nil
}

// Start mantém compatibilidade para rodar ambos em goroutines (Modo Monolítico).
func (e *Engine) Start(ctx context.Context) {
	go e.ConsumeInbox(ctx)
	go e.RelayOutbox(ctx)
}

// ReplayInboxDLQ resgata um evento do purgatório lógico.
func (e *Engine) ReplayInboxDLQ(ctx context.Context, eventID string) error {
	return e.Repo.ReplayInboxDLQ(ctx, eventID)
}

// ReplayOutboxDLQ resgata um evento do purgatório lógico.
func (e *Engine) ReplayOutboxDLQ(ctx context.Context, eventID string) error {
	return e.Repo.ReplayOutboxDLQ(ctx, eventID)
}

// RotateGatewaySecret dispara a rotação de chaves em um adapter compatível.
func (e *Engine) RotateGatewaySecret(providerID string, newSecret string, gracePeriod time.Duration) error {
	adapter, err := e.Registry.Get(providerID)
	if err != nil {
		return err
	}

	type rotatable interface {
		RotateWebhookSecret(newSecret string, gracePeriod time.Duration)
	}

	if r, ok := adapter.(rotatable); ok {
		r.RotateWebhookSecret(newSecret, gracePeriod)
		return nil
	}

	return fmt.Errorf("provedor %s não suporta rotação dinâmica", providerID)
}
