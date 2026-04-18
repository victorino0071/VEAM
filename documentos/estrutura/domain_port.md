# Domínio: Ports (As Fronteiras Lógicas)
**Caminho:** `domain/port`

Os *Ports* definem os contratos que o motor de pagamentos exige que o mundo exterior implemente. Eles são a base da nossa **Arquitetura Hexagonal**.

## 🔌 Portas Principais
-   **`GatewayAdapter`**: Define como criar transações, estornar e validar webhooks em provedores externos.
-   **`Repository`**: Define o contrato de persistência para Inbox, Outbox e Transações (incluindo suporte a ACID Transactions).
-   **`CircuitBreaker`**: Define a interface de controle de fluxo de resiliência.

## ⚖️ Independência Total
As portas permitem que o `core` e o `domain` operem sem saber se estamos usando Postgres ou MongoDB, ou se o pagamento está indo para o Asaas ou enviando Mocks.
