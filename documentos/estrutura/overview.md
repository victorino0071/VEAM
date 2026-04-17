# Overview da Arquitetura: Payment Engine (Agnóstico)
**Caminho:** `documentos/estrutura/overview.md`

Este documento serve como o **Guia Mestre (Índice)** para toda a documentação da base de código do Payment Engine. Nossa arquitetura evoluiu de um integrador específico (Asaas) para um motor de pagamentos altamente modular e agnóstico, seguindo estritos preceitos de **Arquitetura Hexagonal (Ports & Adapters)**, **Clean Architecture** e garantias de **Resiliência Assíncrona de Larga Escala (Inbox/Outbox)**.

O sistema opera de forma autossuficiente e "Zero Config". Execuções são disparadas nativamente via `go run main.go`, o qual utiliza o padrão **Builder Pattern** para configurar automaticamente telemetria, conexões com Postgres Local (com migrações embutidas), e o despertar de _Workers_ em Background.

---

## 1. O Coração e Orquestração (`internal/app`)
Esta camada é o ponto de entrada e coordenação. Ela não contém regras de negócio puro, mas sabe para quem pedir as coisas.

*   🔗 **[Engine (O Orquestrador Principal)](internal_app_engine.md)**
    *   Como o Builder Pattern centraliza a configuração do sistema e esconde a complexidade do bootstrapping.
*   🔗 **[Handlers (Recepção Universal)](internal_app_handler.md)**
    *   Como os Webhooks de qualquer provedor são recebidos e gravados com rastreabilidade total (Trace IDs).
*   🔗 **[ACL (Universal Webhook Translator)](internal_app_acl.md)**
    *   A barreira de imunidade. Transforma DTOs de gateways externos em Entidades de Domínio blindadas.
*   🔗 **[Workers (Background Engines)](internal_app_worker.md)**
    *   O maquinário de concorrência que executa o processamento Inbox/Outbox com retry exponencial.
*   🔗 **[Services (Coordinators)](internal_app_service.md)**
    *   O regente que dita as transações ACID e coordena os domínios.

---

## 2. A Camada Cérebro (`internal/domain`)
O isolamento é completo aqui. Sem imports de Banco de Dados ou HTTP.

*   🔗 **[Entities (Entidades Clássicas)](internal_domain_entity.md)**
    *   As definições de `Transaction`, `Customer` e o novo suporte a `ProviderID`.
*   🔗 **[Payment (Máquina de Estados)](internal_domain_payment.md)**
    *   Garantia de integridade contábil através de uma FSM financeira.
*   🔗 **[Ports (As Fronteiras Lógicas)](internal_domain_port.md)**
    *   Interfaces contratuais que definem como o mundo exterior deve se comportar.

---

## 3. A Camada de Infraestrutura (`internal/infra`)
Implementações concretas e sustentação física.

*   🔗 **[Repository & Migrations (Persistência)](internal_infra_repository.md)**
    *   Uso de `SELECT FOR UPDATE SKIP LOCKED` e automação de esquema via **[Migrações Versicionadas](internal_infra_repository_migration.md)**.
*   🔗 **[Gateway & Adaptadores (Outbound APIs)](internal_infra_gateway.md)**
    *   Suporte a múltiplos provedores através de um Registry dinâmico.
*   🔗 **[Resilience (Disjuntor/Circuit Breaker)](internal_infra_resilience.md)**
    *   Blindagem matemática (`EWMA`) contra falhas de rede.
*   🔗 **[Telemetry (Observabilidade)](internal_infra_telemetry.md)**
    *   Instrumentação via OpenTelemetry.

---

## 4. Ferramentas de Suporte e Testes (`pkg` & `scripts`)
*   🔗 **[Mocking Engine (O Simulador Industrial)](pkg_testing_mock.md)**
    *   Nosso provedor de mocks com árvore de resolução tripla (L1, L2, L3) para testes unitários e de caos.

---

> [!TIP]
> **O Ciclo de Vida do Motor:** O sistema inicia via `app.NewEngine()`. O `.WithAutoMigrate()` garante que o schema SQL esteja pronto. O `.RegisterProvider()` acopla gateways. Ao dar `Start()`, os workers acordam. A internet entra pelos `Handlers`, é traduzida pela `ACL`, processada pelo `Service`, e se houver erro externo, o `CircuitBreaker` protege a saída.
