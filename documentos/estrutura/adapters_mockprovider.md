# Adaptores: MockProvider (Motor de Simulação Tripla)
**Caminho:** `documentos/estrutura/adapters_mockprovider.md`

O `MockProvider` (disponível em `adapters/mockprovider`) não é um mock simples, mas um **Simulador Industrial** projetado para sustentar testes unitários, paralelos e de resiliência.

---

## 🏗️ Árvore de Resolução (L1 -> L2 -> L3)

O motor utiliza uma hierarquia de três níveis para decidir a resposta de uma chamada:

### Camada 1: Magic Overrides (L1)
-   **Propósito:** Injeção imediata de comportamento para IDs específicos.
-   **Mecânica:** `mockProvider.RegisterOverride("tx_123", &Response{Status: entity.StatusPaid})`.
-   **Prioridade:** Absoluta. Se um ID está mapeado em L1, o simulador ignora as camadas inferiores.
-   **Thread-Safety:** Protegido por `sync.RWMutex` para suportar `t.Parallel()`.

### Camada 2: Predicados Lógicos (L2)
-   **Propósito:** Definir comportamentos baseados em regras de negócio dinâmicas.
-   **Mecânica:** `mockProvider.Rules = append(rules, func(ctx, tx) *Response { ... })`.
-   **Uso:** Simular que qualquer transação de um `CustomerID` específico falha, ou que valores acima de R$ 10.000 entram em análise.

### Camada 3: Chaos Engine (L3)
-   **Propósito:** Testar a resiliência (Circuit Breaker) e o comportamento sob condições reais de rede.
-   **Mecânica:** `ChaosRate` (ex: 0.05 para 5% de erro) e `JitterBase` (latência simulada).
-   **Fallthrough:** Se nenhuma regra em L1 ou L2 for disparada, o L3 assume o controle, podendo retornar sucesso ou erro de rede simulado.

---

## 🚀 Exemplo de Uso em Teste

```go
// Inicializa simulador com 10% de falha natural
mock := mockprovider.NewAdapter(0.1, 10 * time.Millisecond)

// Injeta erro crítico em uma transação específica (L1)
mock.RegisterOverride("tx_chaos", &mockprovider.Response{
    Err: errors.New("provider_outage"),
})
```

## ✅ Benefícios Arquiteturais
-   **Determinismo:** Garante que comportamentos complexos sejam reproduzíveis.
-   **Desacoplamento:** Testes não dependem de chaves de API reais ou infraestrutura externa.
-   **Chaos Engineering:** Permite injetar erros de rede no meio da pipeline de processamento para validar o `CircuitBreaker`.
