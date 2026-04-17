# Documentação de Estrutura: Entidades de Domínio
**Caminho:** `internal/domain/entity`

As entidades são o "coração de dados" do sistema. Elas representam objetos de negócio reais com identidade própria e estado. No `asaas_framework`, as entidades são mantidas como objetos simples (POCO - Plain Old Go Objects) para que o domínio seja puro e não dependa de tags de banco de dados ou frameworks externos.

---

## 1. Visão Geral das Entidades

As entidades estão divididas em três categorias principais:
1.  **Atores**: `Customer` (Quem paga).
2.  **Financeiro**: `Transaction` (Pagamentos únicos) e `Subscription` (Recorrência).
3.  **Sistema/Resiliência**: `InboxEvent` e `OutboxEvent` (Mensageria e consistência).

---

## 2. Detalhamento de Entidades Financeiras

### A. `Transaction` (Pagamento Único)
Esta entidade representa uma movimentação financeira individual.

#### Campos Principais:
*   `ID`: Identificador único interno.
*   `Status`: Estado atual (`PENDING`, `PAID`, `FAILED`, etc.).
*   `Amount`: Valor da transação.
*   `DueDate`: Data de vencimento original.
*   `PaymentDate`: Data em que o pagamento foi efetivamente realizado (opcional).

#### Construtor: `NewTransaction(id, customerID, amount, description, dueDate)`
*   **O que faz:** Inicializa uma transação com os dados mínimos obrigatórios.
*   **Objetivo:** Garantir que toda transação comece com o status `PENDING` e com a moeda padrão (`BRL`), evitando que transações sejam criadas em estados inválidos.

---

### B. `Subscription` (Assinatura Recorrente)
Representa um plano de cobrança automática.

#### Campos Principais:
*   `Cycle`: Frequência (`MONTHLY`, `ANNUALLY`, `QUARTERLY`).
*   `NextDueDate`: Calculada automaticamente com base no ciclo.
*   `Status`: `ACTIVE` ou `INACTIVE`.

#### Construtor: `NewSubscription(id, customerID, amount, cycle)`
*   **O que faz:** Cria uma nova assinatura e calcula a próxima data de vencimento.
*   **Objetivo:** Automatizar a lógica de recorrência inicial. Por padrão, adiciona 1 mês à data atual para o primeiro vencimento.

---

## 3. Detalhamento de Atores

### `Customer` (Cliente)
Representa a conta do usuário que realizará os pagamentos.

#### Campos Principais:
*   `Document`: CPF ou CNPJ (vital para integração com o Asaas).
*   `Email`: Destinatário das notificações de cobrança.

#### Construtor: `NewCustomer(id, name, email, document)`
*   **O que faz:** Cria o perfil básico do cliente.
*   **Objetivo:** Centralizar a validação inicial de dados cadastrais antes da integração externa.

---

## 4. Detalhamento de Resiliência (Events)

O framework utiliza eventos para garantir que a comunicação entre partes do sistema nunca falhe silenciosamente.

### `OutboxEvent` & `InboxEvent`

#### Campos Técnicos:
*   `Metadata`: Mapa de strings usado para **Observabilidade**. Armazena dados de rastreamento (Trace IDs) para que possamos seguir uma transação desde o clique do usuário até o banco de dados.
*   `Payload`: O conteúdo bruto (byte array) da mensagem original.
*   `RetryCount`: Quantas vezes o sistema tentou processar esta mensagem.

#### Construtores: `NewOutboxEvent` / `NewInboxEvent`
*   **O que faz:** Cria a estrutura de mensagem pendente.
*   **Objetivo:** Iniciar o ciclo de vida de uma mensagem com status `PENDING` e contador de retentativas zerado.

---

## 5. Por que esta estrutura é robusta?

1.  **Imutabilidade de Regras Iniciais:** Ao usar construtores (`New...`), o sistema força o preenchimento de campos obrigatórios, eliminando o risco de "Null Pointers" ou entidades incompletas navegando pelo sistema.
2.  **Rastreabilidade Nativa:** A presença de metadados em todos os eventos garante que o sistema esteja pronto para logs estruturados e telemetria avançada desde o dia 1.
3.  **Separação de Responsabilidades:** As entidades focam apenas em "O que os dados são". A lógica de "Como eles mudam" fica na Máquina de Estados (FSM) que já documentamos anteriormente.

---

> [!NOTE]
> **Convenção de Nomenclatura:** Todos os arquivos de entidades residem no pacote `entity`. Isso permite que outras partes do sistema usem `entity.Transaction` ou `entity.Customer`, tornando o código altamente legível e auto-explicativo.
