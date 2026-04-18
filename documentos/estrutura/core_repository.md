# Core: Repository (Persistência e Concorrência SQL)
**Caminho:** `internal/core/repository`

A camada de repositório implementa a persistência física sobre o Postgres, utilizando técnicas avançadas para garantir a integridade em ambientes de alta concorrência.

## 🔒 Concorrência via SKIP LOCKED
Para evitar contenção de locks e garantir que múltiplos workers possam operar simultaneamente em um mesmo banco de dados, utilizamos a técnica `SELECT ... FOR UPDATE SKIP LOCKED`.
-   Isso permite que cada réplica do `InboxConsumer` ou `OutboxRelay` reivindique seu próprio lote de eventos sem bloquear as outras.

## 🏗️ Rebuild Pattern
O repositório utiliza o método `entity.RebuildFromRepository` para carregar transações. Isso permite restaurar o estado privado (`status`) sem expor setters públicos que poderiam ser abusados pela camada de aplicação.
