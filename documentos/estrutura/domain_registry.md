# Domínio: Registry (Gestão de Provedores CoW)
**Caminho:** `domain/registry`

O `ProviderRegistry` é o componente responsável por gerenciar o ciclo de vida e a resolução de adaptadores de gateway de forma thread-safe e de alta performance.

## 🚀 Copy-On-Write (CoW)
Diferente de registries que utilizam `sync.Mutex` para proteger todas as operações (incluindo leituras), o nosso motor utiliza a técnica de **Copy-On-Write** via `atomic.Pointer`:

1.  **Leituras (Lock-Free):** O método `Get` carrega o mapa de provedores atomicamente. Em 99% do tempo, as leituras ocorrem sem disputa de lock, garantindo latência mínima no processamento de webhooks e relay.
2.  **Escritas (Sincronizadas):** O método `Register` utiliza um mutex apenas para garantir que duas escritas não ocorram simultaneamente. Ele cria uma cópia do mapa atual, adiciona o novo adaptador e substitui o ponteiro original.

```go
// Exemplo de funcionamento interno
func (r *ProviderRegistry) Register(id string, adapter port.GatewayAdapter) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    oldMap := *r.providers.Load()
    newMap := make(map[string]port.GatewayAdapter, len(oldMap)+1)
    // ... copy and swap ...
    r.providers.Store(&newMap)
}
```

## 🧩 Benefícios
-   **Segurança em Concorrência:** Permite que centenas de workers acessem o registro simultaneamente enquanto um novo provedor é adicionado em runtime.
-   **Imutabilidade:** Uma vez que um mapa é obtido via `Load()`, ele é imutável para aquele contexto, evitando condições de corrida (*race conditions*).

## 🛠️ Uso
O Registry é inicializado automaticamente pela `Engine` e populado via `RegisterProvider`:

```go
engine.RegisterProvider("stripe", stripeAdapter)
```
