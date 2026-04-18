# Facade: Engine (O Orquestrador do Motor)
**Caminho:** `engine.go`

O arquivo `engine.go` atua como o **único ponto de entrada público** para orquestrar o motor de pagamentos. Ele foca na fiação de dependências e expõe métodos terminais para controle explícito da execução.

---

## 🏗️ Bootstrapping (Intenção de Execução)

O motor não inicia processos em background automaticamente. O hospedeiro deve decidir o papel (Role) da instância invocando métodos explícitos.

```go
engine := paymentengine.NewEngine(db).
    WithTelemetry("service-name").
    RegisterProvider("asaas", adapter)
```

### 🚨 Aviso sobre Migrações
O método `WithAutoMigrate()` foi removido do fluxo de runtime para prevenir contenção de locks DDL em ambientes escalados horizontalmente. As migrações devem ser executadas via **[Engine CLI](cli_operations.md)**.

---

## 🚦 Métodos Terminais (Execution Roles)

O motor suporta topologias de execução assimétricas:

### 1. Modo API (Passivo)
Para pods que apenas recebem webhooks e disponibilizam serviços.
-   **`NewWebhookHandler(providerID)`**: Retorna um `http.Handler` pronto para ingestão.

### 2. Modo Worker (Ativo/Background)
Para pods dedicados ao processamento pesado.
-   **`ConsumeInbox(ctx)`**: Inicia o loop de processamento do Inbox (bloqueante).
-   **`RelayOutbox(ctx)`**: Inicia o despachante de eventos externos (bloqueante).

### 3. Modo Monolítico
-   **`Start(ctx)`**: Helper que dispara os workers em goroutines separadas (útil para desenvolvimento local).

---

## 🔒 Encapsulamento
A `Engine` oculta o diretório `internal/core`, garantindo que o hospedeiro utilize apenas as APIs de alto nível e as entidades de domínio, preservando a integridade das transações ACID.
