# Operações: Engine CLI
**Caminho:** `cmd/payment-cli`

O `VEAM-cli` é uma ferramenta administrativa independente projetada para gerenciar o estado físico do banco de dados e tarefas de manutenção fora do ciclo de vida da aplicação principal.

---

## 🏗️ Filosofia de Operação
Em um ambiente industrial (Kubernetes, AWS ECS, etc.), o mapeamento de esquema (DDL) não deve ser um efeito colateral da subida da aplicação. 
-   **Evita Contenção:** Impede que 50 nós tentando subir ao mesmo tempo disputem locks de catálogo no Postgres.
-   **Step Determinístico:** A migração torna-se um step explícito no pipeline de CI/CD (Init Container ou Job).

## 🚀 Comandos Disponíveis

### `migrate`
Sincroniza o banco de dados com as versões embutidas no motor.

```bash
# Exemplo de uso via DSN explicito
./VEAM-cli migrate -dsn "postgres://user:pass@localhost:5432/db"

# Exemplo usando variável de ambiente (DATABASE_URL)
DATABASE_URL="..." ./VEAM-cli migrate
```

## 🛠️ Extensibilidade
A CLI pode ser estendida para incluir:
-   `replay-event`: Forçar o reprocessamento de um ID específico do Inbox/Outbox.
-   `cleanup`: Purga de eventos antigos já processados.
-   `status`: Relatório de saúde das filas e latência de processamento.
