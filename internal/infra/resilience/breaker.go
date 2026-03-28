package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
	"asaas_framework/internal/domain/port"
)

const (
	StateClosed   = "CLOSED"
	StateOpen     = "OPEN"
	StateHalfOpen = "HALF_OPEN"
)

type Config struct {
	FailureThreshold float64       // τ (tau) - taxa de falha permitida (ex: 0.5)
	ResetTimeout     time.Duration // t_timeout - tempo para ir de OPEN para HALF_OPEN
	MinRequests      int           // Mínimo de requisições para abrir o disjuntor
}

type circuitBreaker struct {
	config      Config
	mu          sync.RWMutex
	state       string
	failures    int
	total       int
	lastFailure time.Time
}

func NewCircuitBreaker(config Config) port.CircuitBreaker {
	return &circuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

func (cb *circuitBreaker) Allow(ctx context.Context) (bool, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.total = 0
			return true, nil
		}
		return false, errors.New("circuit breaker is open")
	}

	return true, nil
}

func (cb *circuitBreaker) RecordResult(ctx context.Context, err error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.total++
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
	}

	failureRate := float64(cb.failures) / float64(cb.total)

	// Lógica Matemática: S_{t+1}
	if cb.total >= cb.config.MinRequests && failureRate > cb.config.FailureThreshold {
		cb.state = StateOpen
	} else if cb.state == StateHalfOpen && err == nil {
		// Sucesso no HalfOpen -> Volta para Closed (Caso Contrário)
		cb.reset()
	} else if cb.state == StateOpen && time.Since(cb.lastFailure) > cb.config.ResetTimeout {
		// Opcional: atualização passiva de estado se necessário, 
		// mas o Allow() já cuida disso.
	}

	return nil
}

func (cb *circuitBreaker) GetState(ctx context.Context) (string, error) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, nil
}

func (cb *circuitBreaker) reset() {
	cb.state = StateClosed
	cb.failures = 0
	cb.total = 0
}
