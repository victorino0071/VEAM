package resilience

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"asaas_framework/internal/domain/port"
	"log/slog"
)

const (
	StateClosed   = "CLOSED"
	StateOpen     = "OPEN"
	StateHalfOpen = "HALF_OPEN"
)

type Config struct {
	FailureThreshold float64       // τ (tau) - ex: 0.5 (50%)
	ResetTimeout     time.Duration // t_timeout
	Alpha            float64       // α (alpha) - Coeficiente de reatividade (ex: 0.2)
	MinRequests      int           // Mínimo de amostras para reatividade total
}

type circuitBreaker struct {
	config     Config
	mu         sync.RWMutex
	state      string
	pFailure   uint64        // P_t (Atomic float64 via math.Float64bits)
	requestCount uint64      // Mínimo para aquecimento
	lastTransition time.Time
}

func NewCircuitBreaker(config Config) port.CircuitBreaker {
	if config.Alpha == 0 {
		config.Alpha = 0.2 // Default reativo
	}
	return &circuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

func (cb *circuitBreaker) Allow(ctx context.Context) (bool, error) {
	cb.mu.RLock()
	state := cb.state
	lastT := cb.lastTransition
	cb.mu.RUnlock()

	if state == StateOpen {
		if time.Since(lastT) > cb.config.ResetTimeout {
			cb.transition(ctx, StateHalfOpen)
			return true, nil
		}
		return false, errors.New("circuit breaker is open")
	}

	return true, nil
}

func (cb *circuitBreaker) RecordResult(ctx context.Context, err error) error {
	var result float64
	if err != nil {
		result = 1.0
	} else {
		result = 0.0
	}

	// Cálculo da EWMA Atômica: P_new = (1-α)P_old + α(Result)
	for {
		oldBits := atomic.LoadUint64(&cb.pFailure)
		oldP := math.Float64frombits(oldBits)
		
		newP := (1-cb.config.Alpha)*oldP + cb.config.Alpha*result
		newBits := math.Float64bits(newP)

		if atomic.CompareAndSwapUint64(&cb.pFailure, oldBits, newBits) {
			break
		}
	}

	atomic.AddUint64(&cb.requestCount, 1)

	// Análise de Transição baseada em P(F)
	currentP := math.Float64frombits(atomic.LoadUint64(&cb.pFailure))
	totalReq := atomic.LoadUint64(&cb.requestCount)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if totalReq >= uint64(cb.config.MinRequests) && currentP > cb.config.FailureThreshold {
		cb.setState(ctx, StateOpen)
	} else if cb.state == StateHalfOpen && err == nil {
		cb.setState(ctx, StateClosed)
		// Opcional: Reset de P no sucesso total para acelerar recuperação
		atomic.StoreUint64(&cb.pFailure, 0)
	}

	return nil
}

func (cb *circuitBreaker) transition(ctx context.Context, next string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(ctx, next)
}

func (cb *circuitBreaker) setState(ctx context.Context, next string) {
	if cb.state != next {
		slog.WarnContext(ctx, "Circuit Breaker Transição (Mastery)", 
			"from", cb.state, 
			"to", next,
			"p_failure", math.Float64frombits(atomic.LoadUint64(&cb.pFailure)))
		cb.state = next
		cb.lastTransition = time.Now()
	}
}

func (cb *circuitBreaker) GetState(ctx context.Context) (string, error) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, nil
}

func (cb *circuitBreaker) GetFailureProbability(ctx context.Context) (float64, error) {
	return math.Float64frombits(atomic.LoadUint64(&cb.pFailure)), nil
}
