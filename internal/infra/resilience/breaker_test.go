package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	ctx := context.Background()
	config := Config{
		FailureThreshold: 0.5,
		ResetTimeout:     100 * time.Millisecond,
		MinRequests:      4,
	}
	cb := NewCircuitBreaker(config)

	// 1. Inicia CLOSED
	state, _ := cb.GetState(ctx)
	if state != StateClosed {
		t.Errorf("Esperado status CLOSED, obtido: %s", state)
	}

	// 2. Registra 4 falhas (100% erro > τ 0.5)
	for i := 0; i < 4; i++ {
		cb.RecordResult(ctx, errors.New("falha"))
	}

	state, _ = cb.GetState(ctx)
	if state != StateOpen {
		t.Errorf("Esperado status OPEN após falhas, obtido: %s", state)
	}

	// 3. Verifica que Allow() retorna falso
	allowed, _ := cb.Allow(ctx)
	if allowed {
		t.Error("Esperado Allow() falso em estado OPEN")
	}

	// 4. Espera o timeout de reset
	time.Sleep(150 * time.Millisecond)

	// 5. Verifica que mudou para HALF_OPEN no Allow()
	allowed, _ = cb.Allow(ctx)
	if !allowed {
		t.Error("Esperado Allow() verdadeiro em HALF_OPEN após timeout")
	}

	state, _ = cb.GetState(ctx)
	if state != StateHalfOpen {
		t.Errorf("Esperado status HALF_OPEN, obtido: %s", state)
	}

	// 6. Registra sucesso
	cb.RecordResult(ctx, nil)
	state, _ = cb.GetState(ctx)
	if state != StateClosed {
		t.Errorf("Esperado status voltar para CLOSED após sucesso no HALF_OPEN, obtido: %s", state)
	}
}
