# Core: Worker (Motores de Background)
**Caminho:** `internal/core/worker`

Os Workers são os componentes de concorrência que garantem que o motor de pagamentos nunca pare, processando eventos de forma assíncrona e resiliente.

## 📥 Inbox Consumer
Responsável por processar webhooks recebidos.
-   **Phase A (Claim):** Reivindica eventos `PENDING` no banco usando `SELECT FOR UPDATE SKIP LOCKED`.
-   **Phase B (Execute):** Chama o `PaymentService` para executar a lógica de domínio.
-   **Phase C (Finalize):** Marca o evento como `COMPLETED` ou `FAILED` (com suporte a DLQ).
-   **Backoff:** Implementa jitter e backoff exponencial caso não existam mensagens para processar.

## 📤 Outbox Relay
Responsável por despachar notificações para o mundo exterior.
-   **Circuit Breaker Integration:** O Relay é protegido por um disjuntor. Se o destino externo estiver falhando, o Relay para de tentar imediatamente para evitar desperdício de recursos e falhas em cascata.
-   **Rastreabilidade:** Preserva o `TraceID` original em toda a cadeia de despacho.
