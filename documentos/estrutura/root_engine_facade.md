# Facade: Engine (O Orquestrador do Motor)
**Caminho:** `engine.go`

O arquivo `engine.go` na raiz do repositório atua como o **único ponto de entrada público** para orquestrar o motor de pagamentos. Ele implementa o padrão **Facade** para esconder a complexidade da fiação interna (`internal/core`) e o padrão **Builder Pattern** para configuração fluida.

---

## 🏗️ Padrão Builder
A inicialização do motor é projetada para ser semântica e "Zero Config" por padrão, permitindo customizações graduais.

```go
engine := paymentengine.NewEngine(db).
    WithAutoMigrate().                  // Garante esquema SQL pronto
    WithTelemetry("service-name").      // Habilita OpenTelemetry
    RegisterProvider("asaas", adapter) // Acopla gateways
```

### Principais Métodos
-   **`NewEngine(db)`**: Inicializa o núcleo (Repos, Services e Workers) sobre uma conexão Postgres.
-   **`WithAutoMigrate()`**: Executa as migrações versionadas embutidas no binário.
-   **`RegisterProvider(id, adapter)`**: Registra um gateway no Registry dinâmico do motor.
-   **`NewWebhookHandler(providerID)`**: Fábrica de handlers HTTP já configurados com ACL e Repositório.
-   **`Start(ctx)`**: Desperta os Workers de Background (Inbox Consumer e Outbox Relay).

---

## 🔒 Encapsulamento de Core
O `Engine` é o único componente que possui visibilidade total de:
-   `internal/core/worker`
-   `internal/core/service`
-   `internal/core/repository`

Ao expor apenas a struct `Engine` na raiz, impedimos que o consumidor da biblioteca acesse acidentalmente componentes de infraestrutura ou tente manipular transações ACID manualmente fora do `PaymentService`.

---

## 🚦 Ciclo de Vida do Webhook
1.  O `WebhookHandler` (gerado pela Engine) recebe o POST.
2.  Delegada a validação e tradução ao `Adapter` registrado.
3.  O `InboxConsumer` (iniciado pelo `.Start()`) processa o evento via `PaymentService`.
4.  A `FSM` na entidade de domínio decide a transição final.
