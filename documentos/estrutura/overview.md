# Overview da Arquitetura: Payment Engine (Industrial Sovereign)
**Caminho:** `documentos/estrutura/overview.md`

Este documento é o **Índice Mestre** da biblioteca `github.com/Victor/payment-engine`. Nossa arquitetura evoluiu para um modelo **Industrial Sovereign**, focado em blindagem de estado, desacoplamento de escala e prudência operacional.

---

## 🏗️ Topologia de Exportação

### 1. Camadas Públicas (Consumo Sovereign)
*   🔗 **[Facade: Engine (O Orquestrador)](root_engine_facade.md)**: Ponto de entrada com métodos terminais explícitos.
*   🔗 **[Domain: Entity (Memento & Policy)](domain_entity.md)**: Entidades com **Opaque State** e FSM injetável via de Políticas.
*   🔗 **[Domain: Port (Interfaces)](domain_port.md)**: Contratos hexagonais para repositórios e adaptadores.
*   🔗 **[Adapters: Asaas (Gateway)](adapters_asaas.md)**: Conector de pagamento real.
*   🔗 **[Adapters: Mock (Simulador)](adapters_mockprovider.md)**: Motor de simulação tripla (L1/L2/L3) para caos e testes.

### 2. Operações e Manutenção
*   🔗 **[CLI: Operações (Isolamento DDL)](cli_operations.md)**: Ferramenta administrativa para migrações e auditoria.

### 3. Camada Interna (Blindagem de Core)
*   🔗 **[Core: Service (ACID Transaction)](core_service.md)**: Orquestração de negócio sob isolamento.
*   🔗 **[Core: Worker (Asymmetric Scaling)](core_worker.md)**: Motores de background com métodos de início explícitos.
*   🔗 **[Core: Repository (Snapshot Hydration)](core_repository.md)**: Persistência física via `SKIP LOCKED` e rebuild via `Snapshot`.
*   🔗 **[Core: Resilience (Circuit Breaker)](core_resilience.md)**: Blindagem estatística via EWMA.
*   🔗 **[Core: Telemetry (Observabilidade)](core_telemetry.md)**: Rastreabilidade via OpenTelemetry.

---

## 🚀 Fluxo de Deploy Industrial

O ciclo de vida do motor segue um pipeline de segurança estrito:

1.  **Phase 0 (Provision):** Execução do `payment-engine-cli migrate` via Init Container ou Pipeline.
2.  **Phase 1 (WIRING):** O hospedeiro instala a biblioteca e configura as dependências via `NewEngine`.
3.  **Phase 2 (EXEC):** O hospedeiro decide o papel da instância:
    -   Instâncias de **Ingress** invocam `NewWebhookHandler`.
    -   Instâncias de **Background** invocam `ConsumeInbox` e `RelayOutbox`.

> [!IMPORTANT]
> **Sovereignty Rule:**
> A biblioteca impede a mutação direta de estado. Toda transição financeira deve ser validada por uma `TransitionPolicy` e exportada via `Snapshot`, garantindo que o banco de dados seja apenas um "Memento" do domínio soberano.
