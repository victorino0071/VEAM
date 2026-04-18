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
-   **Phase B (Execute):** Chama o `PaymentService` para executar a lógica de domínio (validada pelas Políticas).
-   **Phase C (Finalize):** Marca o evento como `COMPLETED` ou `FAILED`.

## 📤 Outbox Relay
Responsável por despachar notificações para o mundo exterior (Outbound).
-   **Circuit Breaker:** Protegido por disjuntor para evitar exaustão de recursos em caso de instabilidade externa.
-   **Rastreabilidade:** Preserva o `TraceID` original capturado no Webhook Input.
