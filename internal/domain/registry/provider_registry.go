package registry

import (
	"asaas_framework/internal/domain/port"
	"fmt"
	"sync"
)

// ProviderRegistry gerencia múltiplos adaptadores de gateway de pagamento.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]port.GatewayAdapter
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]port.GatewayAdapter),
	}
}

// Register adiciona um provedor ao registro.
func (r *ProviderRegistry) Register(id string, adapter port.GatewayAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[id] = adapter
}

// Get recupera um adaptador pelo ID do provedor.
func (r *ProviderRegistry) Get(id string) (port.GatewayAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provedor não encontrado: %s", id)
	}
	return adapter, nil
}
