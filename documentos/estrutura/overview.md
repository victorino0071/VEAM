# Overview da Arquitetura: Asaas Framework
**Caminho:** `documentos/estrutura/overview.md`

Este documento serve como o **Guia Mestre (Índice)** para toda a documentação da base de código do The Asaas Framework. Nossa arquitetura foi montada seguindo estritos preceitos de **Arquitetura Hexagonal (Ports & Adapters)**, **Clean Architecture** e garantias de **Resiliência Assíncrona de Larga Escala (Inbox/Outbox)**.

Caso você seja um desenvolvedor novo realizando o *Onboarding* no projeto, a ordem de leitura recomendada dos manuais infra-citados reflete exatamente como os dados fluem da internet até o núcleo do negócio e, finalmente, para o banco de dados.

---

## 1. A Camada de Entrada e Orquestração (`internal/app`)
Esta camada é de onde os dados externos chegam (via Webhooks ou Polling de mensagens) e como eles são orquestrados e embalados para que a nossa malha de negócio não precise entender o "mundo feio lá fora".

*   🔗 **[Handlers (Recepção HTTP)](internal_app_handler.md)**
    *   Como os Webhooks são recebidos, rastreados (Tracer ID) e gravados secamente (`Blind Ingestion`) em nanossegundos no BD.
*   🔗 **[ACL (Anti-Corruption Layer)](internal_app_acl.md)**
    *   O tradutor. Transforma o que vem no formato JSON/DTO do Asaas em Entidades e Enums blindados que só nós reconhecemos.
*   🔗 **[Workers (Background Engines)](internal_app_worker.md)**
    *   O maquinário pesadíssimo (rodando simultaneamente usando `Exponential Backoff`) que recolhe das filas do Inbox o que foi armazenado às pressas, e executa os fluxos assíncronos.
*   🔗 **[Services (Coordinators)](internal_app_service.md)**
    *   O regente de orquestra. Ele dita sob quais Transações ACID (garantias do BD) chamaremos o negócio e prepararemos saídas para o Outbox.

---

## 2. A Camada Cérebro (`internal/domain`)
O isolamento é completo aqui. Sem imports de Banco de Dados, sem bibliotecas de HTTP. É onde o "Dinheiro" e "Regras de Negócio" de fato pernoitam.

*   🔗 **[Entities (Entidades Clássicas)](internal_domain_entity.md)**
    *   As definições sagradas (POCO - Plain Old Go Objects) de `Transaction`, `Customer`, `Subscription` e de Fila. E seus construtores.
*   🔗 **[Payment (Máquina de Estados)](internal_domain_payment.md)**
    *   Ninguém altera status manual aqui. A entidade deve passar por uma FSM para evitar inconsistências contábeis e transações corrompidas.
*   🔗 **[Ports (As Fronteiras Lógicas)](internal_domain_port.md)**
    *   As interfaces contratuais (Gateways, Repositories). O limite máximo de até onde o Core se estende para exigir comportamento do mundo exterior.

---

## 3. A Camada de Aço e Concreto (`internal/infra`)
O submundo e sustentação física da fundação. Aqui ficam as requisições lentas de internet TCP, SQLs altamente otimizados e cálculo matemático para bloqueio de latência de cloud. Implementam obedientemente os "Ports" exigidos no capitulo anterior.

*   🔗 **[Repository (Persistência Avançada)](internal_infra_repository.md)**
    *   Entenda como resolvemos colisões no banco de dados via _UPSERTS_ (`ON CONFLICT`) e a maravilha que confere concorrência de Cloud gigantesca para nossos Workers não trombarem uns nos outros (`SELECT FOR UPDATE SKIP LOCKED`).
*   🔗 **[Resilience (Disjuntor/Circuit Breaker)](internal_infra_resilience.md)**
    *   Nossa blindagem matemática (`EWMA`) com travas atômicas diretas no processador (`sync/atomic`) que barra trafego hostil no momento que as Redes ou Gateways explodem as métricas.
*   🔗 **[Gateway (Ponto de Fuga para Rede)](internal_infra_gateway.md)**
    *   As conexões HTTP reais com mockings para simular perfeitamente comportamentos de provedores na AWS e afins.
*   🔗 **[Telemetry (Monitoramento Sistêmico)](internal_infra_telemetry.md)**
    *   A orquestração _Singleton_ via pacotadores assíncronos das bibliotecas OpenTelemetry, que preenchem nossas labels cruzadas no Datadog/Grafana para não nos perdemos _em tempo de debugging_.

---

> [!TIP]
> **Fluxo Mestre na Prática:** A internet bate no `Handler`, cai cegamente para DB no pattern Inbox. O `Worker` desperta, recolhe a mensagem, puxa a `ACL` para traduzi-la. A ACL limpa entrega pro `Service`. O Service envolve o BD num _ACID Limit_, pede permissões ao Núcleo (`FSM`), autoriza. O Serviço avisa o Banco (`Repository`),  salva tudo e cria uma ordem de alerta no Pattern de Saida (Outbox). Outro `Worker` acorda, empacota o pedido Outbox, se projeta na internet e bate no `Gateway` ativando travas do `CircuitBreaker`. Se algo der errado e apitar, os painéis do `Telemetry` gritam!
