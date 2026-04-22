# Core: Migrations (Esquema Industrial)
**Caminho:** `internal/core/migration`

O motor utiliza um sistema de migração versionado para gerenciar a evolução do banco de dados de maneira previsível.

## 🚀 Filosofia CLI-First
Embora o código suporte a execução programática via `EnsureSchema(db)`, em ambientes industriais (Kubernetes, AWS ECS), **não recomendamos** a execução automática ao iniciar a `Engine`. Isso evita contenção de locks DDL em múltiplos pods.
- O padrão recomendado é o uso do **[Engine CLI](cli_operations.md)** como um *Init Container* ou etapa de CI/CD.
## 📦 Mecanismo de Embed
As queries de criação de tabela e índices estão embutidas no binário Go via `embed`. Isso garante que o motor seja auto-contido e não dependa de arquivos SQL externos no ambiente de execução.
-   **Tabelas Principais:** `transactions`, `customers`, `inbox`, `outbox`, `payment_engine_migrations`.
-   **Atomicidade:** As migrações são aplicadas dentro de transações SQL, garantindo que o banco nunca fique em um estado inconsistente caso uma query falhe.
