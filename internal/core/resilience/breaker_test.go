package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

func TestCircuitBreaker_DoubleCheckLocking_StartingGun(t *testing.T) {
	ctx := context.Background()
	config := Config{
		FailureThreshold: 0.1,
		ResetTimeout:     50 * time.Millisecond,
		MinRequests:      1,
		Alpha:            1.0,
	}
	cb := NewCircuitBreaker(config)

	// Força o estado OPEN
	cb.RecordResult(ctx, errors.New("falha"))
	
	state, _ := cb.GetState(ctx)
	if state != StateOpen {
		t.Fatalf("Deveria estar OPEN, mas está %s", state)
	}

	// Aguarda o timeout do reset
	time.Sleep(60 * time.Millisecond)

	// Cenário de Estouro: 100 goroutines tentam Allow() simultaneamente
	numGoroutines := 100
	readyChan := make(chan struct{})
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-readyChan // Bloqueio até o sinal de partida
			
			allowed, err := cb.Allow(ctx)
			if allowed && err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failureCount, 1)
			}
		}()
	}

	// DISPARAR
	close(readyChan)
	wg.Wait()

	// VALIDAÇÃO: Padrão Double-Check Locking
	// Apenas UMA goroutine deve ter conseguido o lock para transição p/ HALF_OPEN.
	// As outras devem ter falhado no double-check ("lost race to half-open").
	if successCount != 1 {
		t.Errorf("Esperado que apenas 1 goroutine vencesse a corrida para HALF_OPEN, mas %d venceram", successCount)
	}

	expectedFailures := int64(numGoroutines - 1)
	if failureCount != expectedFailures {
		t.Errorf("Esperado %d falhas na corrida, mas houve %d", expectedFailures, failureCount)
	}

	// Verifica se o estado agora é HALF_OPEN
	state, _ = cb.GetState(ctx)
	if state != StateHalfOpen {
		t.Errorf("Estado deveria ser HALF_OPEN, obtido: %s", state)
	}
}

func TestCircuitBreaker_EWMA_AlphaProgression(t *testing.T) {
	ctx := context.Background()
	// Teste com Alpha suave (0.2)
	cb := NewCircuitBreaker(Config{
		Alpha:            0.2,
		FailureThreshold: 0.5,
		MinRequests:      5,
	})

	// Com Alpha=0.2, uma falha isolada não deve levar a P=1.0 nem abrir o disjuntor imediatamente.
	cb.RecordResult(ctx, errors.New("falha 1"))
	p, _ := cb.GetFailureProbability(ctx)
	
	// P_new = (1-0.2)*0 + 0.2*1 = 0.2
	if p != 0.2 {
		t.Errorf("Esperado P=0.2 após primeira falha com Alpha=0.2, obtido: %f", p)
	}

	state, _ := cb.GetState(ctx)
	if state != StateClosed {
		t.Errorf("Estado deveria continuar CLOSED com P=0.2, obtido: %s", state)
	}

	// Mais 4 falhas
	for i := 0; i < 4; i++ {
		cb.RecordResult(ctx, errors.New("falha"))
	}
	
	// P_new após 5 falhas totais:
	// 1: 0.2
	// 2: 0.8*0.2 + 0.2*1 = 0.36
	// 3: 0.8*0.36 + 0.2*1 = 0.488
	// 4: 0.8*0.488 + 0.2*1 = 0.5904
	// 5: 0.8*0.5904 + 0.2*1 = 0.67232
	
	p, _ = cb.GetFailureProbability(ctx)
	if p < 0.6 {
		t.Errorf("Esperado P > 0.6 após 5 falhas, obtido: %f", p)
	}

	state, _ = cb.GetState(ctx)
	if state != StateOpen {
		t.Errorf("Estado deveria estar OPEN após atingir limiar com 5 requisições, obtido: %s", state)
	}
}
