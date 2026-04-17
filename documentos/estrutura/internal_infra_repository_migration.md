# Documentação de Estrutura: Migrações Versionadas (Auto-Setup)
**Caminho:** `internal/infra/repository/migration`

Este componente fornece a automação de infraestrutura necessária para o motor ser "Zero Config". Ele garante que o esquema do banco de dados PostgreSQL esteja sempre sincronizado com as necessidades da versão do código.

---

## 1. O Conceito de Embedded Migrations
*( **Conceito Técnico - Embedded Files (//go:embed):** No Go, podemos incorporar arquivos de texto (como scripts .sql) diretamente dentro do binário final. Isso significa que você não precisa carregar uma pasta de migrações separada junto com seu executável; o código "carrega seu próprio banco" ).*

---

## 2. Visão Geral do `migrator.go`

O motor de migração opera seguindo quatro princípios de robustez industrial:

### A. Controle de Versão (Metadata)
O sistema cria uma tabela auxiliar chamada `asaas_framework_migrations`. Nela, armazenamos apenas a última versão de esquema aplicada com sucesso. Isso evita a fragilidade do simples `IF NOT EXISTS` em atualizações futuras onde novas colunas são adicionadas.

### B. Delta Execution (Mecânica Sequencial)
Os arquivos SQL são lidos do diretório embutido `migrations/`. Eles devem ser numerados de forma estrita (ex: `001_initial.sql`). O migrador carrega todos, mas só executa aqueles cuja numeração é **maior** que a versão registrada no banco de dados.

### C. Atomicidade Transacional
*( **Conceito Técnico - ACID Transaction:** Cada arquivo de migração é executado dentro de uma transação do banco de dados. Se uma única linha do SQL falhar, o banco sofre um ROLLBACK total daquele arquivo, evitando que o banco de dados fique em um estado "meio alterado" e corrompido. )*

---

## 3. O Esquema Core (Inbox/Outbox/Transactions)

As migrações iniciais garantem a criação das três peças fundamentais:
1.  **Inbox:** A fila de entrada (blind ingestion) para resiliência de recepção.
2.  **Outbox:** A fila de saída para resiliência de chamadas a APIs externas.
3.  **Transactions:** A tabela de domínio que agora suporta múltiplos gateways através da coluna `provider_id`.

---

> [!CAUTION]
> **Manutenção de IDs:** Nunca altere o número inicial de um arquivo de migração já aplicado em produção. Para qualquer alteração estrutural futura, crie um novo arquivo com o próximo número sequencial na pasta `migrations/`.
