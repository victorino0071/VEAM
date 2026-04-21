package registry

import (
	"fmt"
	"sync"
	"testing"
	"github.com/Victor/payment-engine/domain/port"
)

// mockAdapter para testes do Registro
type mockAdapter struct {
	port.GatewayAdapter
	id string
}

func TestProviderRegistry_Concurrency_StartingGun(t *testing.T) {
	reg := NewProviderRegistry()
	numGoroutines := 100
	readyChan := make(chan struct{})
	var wg sync.WaitGroup

	// 1. Preparar o estouro de concorrência
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-readyChan // Bloqueio até o sinal de partida

			providerID := fmt.Sprintf("provider-%d", id)
			reg.Register(providerID, &mockAdapter{id: providerID})

			// Tenta ler logo em seguida para forçar contenção de RWMutex
			_, _ = reg.Get(providerID)
		}(i)
	}

	// 2. DISPARAR (Starting Gun)
	close(readyChan)
	wg.Wait()

	// 3. Validação Final
	for i := 0; i < numGoroutines; i++ {
		providerID := fmt.Sprintf("provider-%d", i)
		adapter, err := reg.Get(providerID)
		if err != nil {
			t.Errorf("Provedor %s não encontrado após teste de estresse", providerID)
		}
		if adapter.(*mockAdapter).id != providerID {
			t.Errorf("ID incorreto para o provedor %s", providerID)
		}
	}
}
