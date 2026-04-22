# Core: Handler (Portas de Entrada HTTP)
**Caminho:** `internal/core/handler`

Os Handlers são os pontos onde o tráfego externo (Webhooks) entra no sistema.

## 🛡️ Webhook Handler
Diferente de handlers comuns, este é altamente genérico e antifrágil:
-   **Validação Delegada:** Passa a responsabilidade de autenticação para o `Adapter` correspondente.
- **Passo 4 (Deduplicação):** O Adaptador gera um **Fingerprint** do payload core (ignorando ruídos). O sistema verifica se este hash já existe para o provedor.
- **Passo 5 (Ingestão Cega):** O evento é salvo na tabela `inbox` com status `PENDING`.
-   **Ingestão Cega:** O handler não tenta processar o negócio; ele apenas valida a assinatura e grava o payload bruto no `Inbox`. Isso garante que webhooks sejam aceitos o mais rápido possível (Status 202 Accepted).
-   **W3C Injection:** Injeta o contexto de rastreabilidade do OpenTelemetry no metadata da transação.
