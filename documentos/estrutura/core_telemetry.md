# Core: Telemetry (Observabilidade Total)
**Caminho:** `internal/core/telemetry`

A visibilidade é um pilar de primeira classe no Payment Engine. Utilizamos o padrão **OpenTelemetry (OTel)** para instrumentação.

## 🕵️ Rastreabilidade Distribuída
Cada transação ou webhook recebe um `TraceID` único no momento em que toca o sistema.
-   **Propagação:** O TraceID viaja do Handler HTTP, passa pelo Banco de Dados (como metadados JSONB), é recuperado pelo Worker e segue até o despacho final no Gateway.
-   **Diagnóstico:** Permite reconstruir a árvore de eventos completa de um pagamento que falhou em qualquer fase do pipeline.

## 📊 Métricas e Spans
O motor exporta spans detalhados para cada fase crítica:
-   `ReceiveWebhook`
-   `ExecuteDomainFSM`
-   `RepositorySave`
-   `GatewayRequest`
