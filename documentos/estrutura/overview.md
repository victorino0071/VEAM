# Overview da Arquitetura: Payment Engine (Library Industrial)
**Caminho:** `documentos/estrutura/overview.md`

Este documento é o **Índice Mestre** da biblioteca `github.com/Victor/payment-engine`. Nossa arquitetura foi migrada para um modelo **Library-First**, priorizando a soberania do domínio e a blindagem contra manipulações externas.

---

## 🏗️ Topologia de Exportação
O projeto segue uma hierarquia de visibilidade rigorosa para garantir a integridade do motor:

### 1. Camadas Públicas (Interface de Consumo)
São os componentes que o usuário da biblioteca importa e utiliza diretamente.

*   🔗 **[Facade: Engine (O Orquestrador)](root_engine_facade.md)**
    *   O ponto de entrada na raiz. Builder Pattern para bootstrapping "Zero Config".
*   🔗 **[Domain: Entity (Sovereign FSM)](domain_entity.md)**
    *   Entidades com **Opaque State**. O domínio dita as regras, não a infraestrutura.
*   🔗 **[Domain: Port (Interfaces Contratuais)](domain_port.md)**
    *   Contratos para Repositórios, Gateways e Resiliência.
*   🔗 **[Adapters: Asaas (Gateway Real)](adapters_asaas.md)**
    *   Implementação concreta do provedor Asaas.
*   🔗 **[Adapters: MockProvider (Simulador)](adapters_mockprovider.md)**
    *   Motor de simulação tripla (L1/L2/L3) para testes e caos.

### 2. Camada Interna (Blindagem de Core)
Localizada em `/internal/core`, bloqueada para importações externas ao módulo.

*   🔗 **[Core: Service (Orquestração ACID)](core_service.md)**
*   🔗 **[Core: Worker (Background Engine)](core_worker.md)**
*   🔗 **[Core: Repository (Persistência Skip Locked)](core_repository.md)**
    *   Sustentação física e **[Migrações Automatizadas](core_migration.md)**.
*   🔗 **[Core: ACL (Universal Translator)](core_acl.md)**
*   🔗 **[Core: Resilience (Circuit Breaker EWMA)](core_resilience.md)**
*   🔗 **[Core: Telemetry (Observabilidade)](core_telemetry.md)**

---

## 🚀 Ciclo de Uso Profissional

```go
// 1. Instância o Motor
engine := paymentengine.NewEngine(db).
    WithAutoMigrate().
    RegisterProvider("asaas", asaas.NewAdapter(key, url))

// 2. Inicia o Maquinário
engine.Start(ctx)

// 3. Recebe Eventos via Facade
mux.Handle("/webhook", engine.NewWebhookHandler("asaas"))
```

> [!IMPORTANT]
> **Sovereign Encapsulation:**
> Nenhuma camada externa pode alterar o `status` de uma transação diretamente. Toda mutação deve ocorrer via `TransitionTo`, garantindo integridade contábil e geração de Auditoria (Outbox).
