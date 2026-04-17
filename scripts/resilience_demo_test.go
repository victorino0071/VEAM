package scripts

import (
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/infra/resilience"
	"asaas_framework/pkg/testing/mock"
	"context"
	"fmt"
	"testing"
	"time"
)

// TestResilienceDemo permite rodar a demonstração de caos e resiliência via 'go test'.
func TestResilienceDemo(t *testing.T) {
	ctx := context.Background()

	// 1. Criar um Mock que SEMPRE falha (100% de erro)
	p := mock.NewMockProvider(1.0, 10*time.Millisecond)

	// 2. Configurar um Circuit Breaker ultra reativo
	cb := resilience.NewCircuitBreaker(resilience.Config{
		FailureThreshold: 0.5, // Abre se 50% das chamadas falharem
		ResetTimeout:     5 * time.Second,
		MinRequests:      3,   // Precisa de pelo menos 3 chamadas para começar a avaliar
		Alpha:            0.8, // Reatividade alta
	})

	fmt.Println("\n=== Iniciando Teste de Stress do Circuit Breaker ===")

	for i := 1; i <= 7; i++ {
		// O Breaker permite a chamada?
		allowed, _ := cb.Allow(ctx)

		if !allowed {
			fmt.Printf("[%d] BLOQUEADO: O Disjuntor abriu para proteger o sistema.\n", i)
			continue
		}

		// Tenta executar (vai falhar pelo mock)
		id, err := p.CreateTransaction(ctx, &entity.Transaction{ID: "tx_caos"})

		// IMPORTANTE: Informar ao Breaker que falhou
		cb.RecordResult(ctx, err)

		fmt.Printf("[%d] EXECUTADO: ID=%s, Erro=%v\n", i, id, err)
		time.Sleep(100 * time.Millisecond)
	}
}
