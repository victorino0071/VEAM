package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_EWMA_Transitions(t *testing.T) {
	ctx := context.Background()
	config := Config{
		FailureThreshold: 0.5,
		ResetTimeout:     100 * time.Millisecond,
		MinRequests:      2,
		Alpha:            1.0, // Reatividade máxima para teste (P_new = Result)
	}
	cb := NewCircuitBreaker(config)

	// 1. Inicia CLOSED
	state, _ := cb.GetState(ctx)
	if state != StateClosed {
		t.Errorf("Esperado status CLOSED, obtido: %s", state)
	}

	// 2. Registro de Falha (P_new = 1.0 > 0.5)
	cb.RecordResult(ctx, errors.New("falha imediata"))
	cb.RecordResult(ctx, errors.New("falha imediata")) // Segunda p/ atingir MinRequests

	state, _ = cb.GetState(ctx)
	if state != StateOpen {
		t.Errorf("Esperado status OPEN após falhas com Alpha=1.0, obtido: %s", state)
	}

	// 3. Verifica Bloqueio
	allowed, _ := cb.Allow(ctx)
	if allowed {
		t.Error("Esperado Allow() falso em estado OPEN")
	}

	// 4. Reconversão para HALF_OPEN após timeout
	time.Sleep(150 * time.Millisecond)
	allowed, _ = cb.Allow(ctx)
	if !allowed {
		t.Error("Esperado Allow() verdadeiro em HALF_OPEN")
	}

	// 5. Sucesso em HALF_OPEN (P_new = 0.0)
	cb.RecordResult(ctx, nil)
	state, _ = cb.GetState(ctx)
	if state != StateClosed {
		t.Errorf("Esperado status CLOSED após sucesso no reset, obtido: %s", state)
	}
}
