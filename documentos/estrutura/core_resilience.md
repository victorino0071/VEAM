# Core: Resilience (Circuit Breaker Industrial)
**Caminho:** `internal/core/resilience`

A blindagem de saída do motor é garantida por um Circuit Breaker implementado com algoritmos de suavização estatística.

## 🛡️ Circuit Breaker com EWMA
Utilizamos a **Média Móvel Exponencialmente Ponderada (EWMA)** para calcular a taxa de falha.
-   **Vantagem:** Diferente de contadores simples, a EWMA dá mais peso aos erros recentes, permitindo que o sistema reaja mais rápido a quedas abruptas de disponibilidade no provedor externo.
-   **Configuração:** O motor permite configurar o threshold de falha (ex: 50% de erro abre o disjuntor) e o tempo de repouso para tentativas de reabertura (Half-Open).

## 🚦 Estados do Disjuntor
1.  **CLOSED:** Operação normal. Tráfego flui.
2.  **OPEN:** Falhas detectadas. O sistema bloqueia requisições externas para proteger o motor e evitar filas infinitas.
3.  **HALF-OPEN:** Teste de recuperação. Envia uma pequena fração do tráfego para verificar se o provedor se estabilizou.
