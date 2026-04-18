# Core: Migrations (Esquema Automatizado)
**Caminho:** `internal/core/migration`

O motor segue o preceito de **Zero Config**. Ao subir a `Engine`, o esquema do banco de dados é automaticamente sincronizado.

## 📦 //go:embed
As queries de criação de tabela e índices estão embutidas no binário via `embed`. Isso elimina a necessidade de gerenciar arquivos SQL externos no ambiente de produção.
-   **Tabelas Principais:** `transactions`, `customers`, `inbox`, `outbox`.
-   **Índices:** Otimizados para as buscas por status e data de criação usadas pelos workers.
