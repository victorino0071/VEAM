package registry

import (
	"github.com/Victor/payment-engine/domain/port"
	"sync/atomic"
	"fmt"
	"sync"
)

// ProviderRegistry gerencia múltiplos adaptadores via Atomic CoW (Copy-On-Write).
type ProviderRegistry struct {
	mu        sync.Mutex // Apenas para sincronizar escritas (registradores)
	providers atomic.Pointer[map[string]port.GatewayAdapter]
}

func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{}
	initial := make(map[string]port.GatewayAdapter)
	r.providers.Store(&initial)
	return r
}

// Register adiciona um provedor ao registro usando Copy-on-Write.
func (r *ProviderRegistry) Register(id string, adapter port.GatewayAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Carrega o mapa atual
	oldMap := *r.providers.Load()
	
	// 2. Cria uma cópia imutável
	newMap := make(map[string]port.GatewayAdapter, len(oldMap)+1)
	for k, v := range oldMap {
		newMap[k] = v
	}
	
	// 3. Adiciona o novo adaptador
	newMap[id] = adapter
	
	// 4. Troca o ponteiro atomicamente
	r.providers.Store(&newMap)
}

// Get recupera um adaptador via leitura atómica pura (Lock-Free).
func (r *ProviderRegistry) Get(id string) (port.GatewayAdapter, error) {
	pMap := r.providers.Load()
	adapter, ok := (*pMap)[id]
	if !ok {
		return nil, fmt.Errorf("provedor não encontrado: %s", id)
	}
	return adapter, nil
}
