# Guia de Execução: Colocando o Asaas Framework para Rodar
**Caminho:** `documentos/guia_execucao.md`

Este guia detalha os passos necessários para configurar o ambiente, inicializar o banco de dados, expor seu webhook via **ngrok** e iniciar o processamento assíncrono.

---

## 1. Preparação do Banco de Dados (Postgres Local)
Como você optou por não usar Docker, certifique-se de que o seu Postgres local está rodando.

1.  Crie um banco de dados chamado `asaas_db` (ou o nome que preferir).
2.  Execute o script de schema para criar as tabelas de Inbox, Outbox e Transactions:
    *   O arquivo está em: `internal/infra/repository/sql/schema.sql`.

---

## 2. Configuração do Ambiente (.env)
Crie um arquivo `.env` na raiz do projeto (copie o `.env.example`) e preencha as variáveis:

```env
ASAAS_API_KEY=sua_chave_aqui
ASAAS_BASE_URL=https://sandbox.asaas.com/api/v3
DB_HOST=localhost
DB_PORT=5432
DB_USER=seu_usuario
DB_PASS=sua_senha
DB_NAME=asaas_db
WEBHOOK_ACCESS_TOKEN=chave_seguranca_que_voce_definir
HTTP_PORT=8080
```

---

## 3. Expondo o Webhook com ngrok
Para que o Asaas consiga enviar webhooks para o seu computador local, você precisa de um túnel público.

1.  No seu terminal, inicie o ngrok apontando para a porta do projeto (padrão 8080):
    ```bash
    ngrok http 8080
    ```
2.  O ngrok fornecerá uma URL (ex: `https://abcd-123.ngrok-free.app`).
3.  **No Painel do Asaas**: Vá em Configurações de Webhook e configure:
    *   **URL**: `https://abcd-123.ngrok-free.app/webhooks/asaas`
    *   **Token de Autenticação**: O mesmo valor que você colocou em `WEBHOOK_ACCESS_TOKEN`.
    *   **Eventos**: Selecione os eventos que deseja receber (ex: Pagamento Confirmado).

---

## 4. Inicializando a Aplicação
Com tudo configurado, execute os comandos abaixo na raiz do projeto:

1.  **Baixar Dependências**:
    ```bash
    go mod tidy
    ```
2.  **Rodar o Programa**:
    ```bash
    go run cmd/main.go
    ```

---

## 5. Como Validar o Fluxo?

### Passo A: O Recebimento (Ingestão)
Quando o Asaas enviar um evento, você verá um log em JSON no terminal:
`{"msg": "[Handler] Webhook persistido no Inbox", "webhook_id": "..."}`.
Neste momento, o dado já está salvo no seu banco de dados local na tabela `inbox`.

### Passo B: O Processamento (Worker)
Em poucos segundos, o `InboxConsumer` (que roda em background) vai acordar, ler o evento, traduzir via ACL e chamar o domínio. Você verá:
`{"msg": "[InboxConsumer] Evento traduzido com sucesso para Domínio", ...}`.

### Passo C: A Saída (Outbox)
Se o processamento gerar uma ação externa (como um estorno ou confirmação extra), o sistema criará um registro na tabela `outbox`, que será enviado pelo `OutboxRelay`. Você verá no log:
`{"msg": "[OutboxRelay] Evento enviado e confirmado pelo Gateway", ...}`.

---

> [!TIP]
> **Dica de Monitoramento:**
> Mantenha uma aba do seu gerenciador de banco de dados (ex: DBeaver ou pgAdmin) aberta. Você poderá ver as linhas mudando de `PENDING` para `COMPLETED` nas tabelas `inbox` e `outbox` conforme os Workers trabalham.
