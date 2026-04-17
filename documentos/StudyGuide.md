# Guia de Estudo Pedagógico: Asaas Framework

Este guia foi criado para ajudar você a entender a estrutura e a lógica por trás deste framework de integração resiliente com o Asaas. O projeto utiliza **Clean Architecture** (Arquitetura Limpa) e padrões de **Resiliência** (Outbox/Inbox) para garantir que nenhuma mensagem seja perdida, mesmo em caso de falha.

---

## 1. O Ponto de Partida: O Domínio (`internal/domain`)

Na Arquitetura Limpa, o "coração" do sistema é o domínio. Ele não conhece banco de dados, APIs externas ou webservers. Ele define o **estudo do problema**.

### [entity/outbox.go](file:///c:/Users/Victor/OneDrive/Documentos/projetos/asaas_framework/internal/domain/entity/outbox.go)
Este é o arquivo mais importante para a confiabilidade.
- **OutboxEvent**: Representa um evento que *sai* do seu sistema para o Asaas (ex: criar uma cobrança).
- **InboxEvent**: Representa uma notificação que *entra* no seu sistema vinda do Asaas (ex: um webhook de pagamento recebido).
- **Por que isso existe?** Em vez de enviar um comando direto para o Asaas e torcer para a rede funcionar, nós salvamos o "desejo" no banco de dados primeiro (`PENDING`). Se a rede cair, o dado está seguro lá para ser tentado novamente.

### [port/resilience_port.go](file:///c:/Users/Victor/OneDrive/Documentos/projetos/asaas_framework/internal/domain/port/resilience_port.go)
Aqui definimos as **Portas** (Interfaces).
- A `CircuitBreaker` define *o que* um disjuntor de segurança deve fazer (abre, fecha, permite).
- Note que não há implementação aqui, apenas a "assinatura" do contrato. Isso permite mudar a biblioteca de resiliência no futuro sem tocar no código de negócio.

---

## 2. A Camada de Aplicação (`internal/app`)

Esta camada orquestra o fluxo de dados. Ela usa as entidades do domínio e chama as portas da infraestrutura.

### [worker/outbox_relay.go](file:///c:/Users/Victor/OneDrive/Documentos/projetos/asaas_framework/internal/app/worker/outbox_relay.go)
Este arquivo é o **motor** (relay).
- Ele é um processo de backend (worker) que fica rodando em loop.
- Sua função: Olhar para o banco de dados, encontrar `OutboxEvents` com status `PENDING`, e tentar enviá-los.
- Ele implementa a lógica de: "Se falhou, aumenta o `RetryCount`. Se funcionou, marca como `PROCESSED`".

---

## 3. A Camada de Infraestrutura (`internal/infra`)

Aqui é onde o código "suja as mãos" com detalhes técnicos (PostgreSQL, OpenTelemetry, chamadas HTTP).

### [telemetry/telemetry.go](file:///c:/Users/Victor/OneDrive/Documentos/projetos/asaas_framework/internal/infra/telemetry/telemetry.go)
Configura a observabilidade.
- Usa **OpenTelemetry** para rastrear o caminho de uma requisição.
- Isso é vital em sistemas resilientes para saber *por que* algo falhou e quanto tempo levou.

---

## Passo-a-Passo Sugerido para o seu Estudo

Se você fosse "refazer" o projeto do zero, eu sugeriria esta ordem:

1.  **Defina as Entidades**: Comece criando o `OutboxEvent` e `InboxEvent`. Eles são a unidade básica de troca.
2.  **Crie as Interfaces (Ports)**: Defina como você quer salvar esses eventos (ex: `OutboxRepository`).
3.  **Implemente o Repositório**: Crie a versão em SQL que salva esses eventos no Postgres.
4.  **Crie o Worker**: Implemente o loop que lê do banco e tenta processar.
5.  **Adicione Resiliência**: Implemente o `Circuit Breaker` e `Retries` para que o Worker não "suicide" o sistema se o Asaas estiver fora do ar.

---

> [!TIP]
> **Conceito Chave**: O padrão **Outbox** resolve o problema de "Atomicidade Distribuída". Você garante que a operação no seu banco de dados e o envio para o Asaas ocorram de forma consistente, mesmo que um deles falhe momentaneamente.
