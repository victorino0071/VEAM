# Core: Repository (Persistência e Concorrência SQL)
**Caminho:** `internal/core/repository`

A camada de repositório implementa a persistência física sobre o Postgres, utilizando técnicas avançadas para garantir a integridade em ambientes de alta concorrência.

## 🔒 Concorrência via SKIP LOCKED
Para evitar contenção de locks e garantir que múltiplos workers possam operar simultaneamente em um mesmo banco de dados, utilizamos a técnica `SELECT ... FOR UPDATE SKIP LOCKED`.
-   Isso permite que cada réplica do `InboxConsumer` ou `OutboxRelay` reivindique seu próprio lote de eventos sem bloquear as outras.
-   **Nota:** O uso de `SKIP LOCKED` é estrito às tabelas de fila (`inbox`, `outbox`). A hidratação do agregado raiz (`transactions`) durante o processamento via `PaymentService` utiliza bloqueio pessimista padrão (`FOR UPDATE`) para garantir a serialização de atualizações concorrentes no mesmo registro.

## 🏗️ Memento Pattern (Snapshot Hydration)
Diferente de sistemas que expõem campos privados via setters públicos, este repositório utiliza o **Memento Pattern**:
-   O repositório lê os campos físicos do banco e preenche uma struct `entity.TransactionSnapshot`.
-   A entidade é então reidratada via a fábrica estática `entity.RestoreTransaction(snapshot)`.
-   Isso garante que o repositório consiga restaurar o estado soberano (`status`) injetando automaticamente as políticas de proteção, sem permitir mutações posteriores por camadas de aplicação.
