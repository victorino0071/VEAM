# Core: Worker (Motores de Background)
**Caminho:** `internal/core/worker`

Os Workers são os componentes de concorrência que garantem que o motor de pagamentos nunca pare, processando eventos de forma assíncrona e resiliente. No modelo industrial, sua execução é **desacoplada** do tráfego de entrada.

## 🚀 Topologia de Execução
Ao contrário de bibliotecas passivas, o motor exige que o hospedeiro invoque explicitamente os métodos de processamento:
-   `engine.ConsumeInbox(ctx)`: Aciona o maquinário de entrada.
-   `engine.RelayOutbox(ctx)`: Aciona o maquinário de saída.

## 📥 Inbox Consumer
Responsável por processar webhooks recebidos e salvos no banco.
-   **Phase A (Claim):** Reivindica eventos `PENDING` usando `SELECT FOR UPDATE SKIP LOCKED`.
-   **Phase B (Execute):** Chama o `PaymentService` para executar a lógica de domínio (validada pelas Políticas). O processamento ocorre fora da transação de banco de dados original para evitar locks longos.
-   **Phase C (Finalize):** Marca o evento como `COMPLETED` ou `FAILED`.

### 🔄 Resiliência e Backoff
O worker utiliza **Exponential Backoff** para evitar sobrecarga no banco de dados quando não há mensagens. Em caso de falha no processamento (Phase B):
1.  O `RetryCount` do evento é incrementado.
2.  Se o limite de retentativas (`MaxRetries`) for atingido, o evento é movido para a **DLQ (Dead Letter Queue)**, sendo marcado como uma *Poison Pill* para análise manual.

## 📤 Outbox Relay
Responsável por despachar notificações para o mundo exterior (Outbound).
-   **Circuit Breaker:** Protegido por um disjuntor reativo que monitora a saúde dos gateways via EWMA.
-   **Sliding Window:** O tamanho do lote de despacho diminui proporcionalmente à taxa de falha do provedor.
-   **SAGA Compensation:** Se um gateway rejeita um estorno de forma terminal (ex: 400 Bad Request), o Relay injeta automaticamente um evento de compensação no **Inbox** (`SYSTEM_INTERNAL`) para que o domínio resolva a anomalia.

### ⚓ Rastreabilidade
- **W3C Propagation:** O Relay recupera o `TraceID` original do metadados, garantindo que o rastro de um pagamento possa ser seguido desde o webhook de entrada até a notificação de saída.
