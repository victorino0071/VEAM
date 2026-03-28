### FASE 1: NÚCLEO, DDD E GARANTIAS DE TRANSAÇÃO (Semanas 1-2)

#### 1.1. Modelagem (Mantido)

Entidades como `Transaction` e `Subscription` permanecem isoladas.

#### 1.2. FSM Rígida (Modificado)

A transição de estados da FSM não deve ocorrer apenas em memória.
Deve ser atrelada a transações de banco de dados (ACID).

#### 1.3. Padrão Outbox/Inbox (Novo)

* Elimine canais em memória para eventos de domínio.
* Qualquer transição de estado da FSM que exija comunicação externa deve:

  * gravar um evento numa tabela **Outbox**
  * na mesma transação de banco de dados

---

### FASE 2: RESILIÊNCIA DISTRIBUÍDA (Semanas 3-4)

#### 2.1. Idempotência Externa (Modificado)

* Implemente locks distribuídos utilizando:

  * Redis (Redlock), ou
  * tabelas de lock transacionais no PostgreSQL
* Nunca confie na memória local para garantir **Exactly-Once Processing**

#### 2.2. Matemática do Circuit Breaker

* Utilize bibliotecas robustas (ex: `sony/gobreaker`)
* Defina formalmente a máquina de estados do breaker

A transição deve seguir:

S_{t+1} = {
Open      se (E_fail / E_total) > τ
HalfOpen  se t > t_timeout
Closed    caso contrário
}

#### 2.3. Mensageria Persistente (Modificado)

* O endpoint do webhook deve ser uma operação **O(1)** que:

  * escreve o payload cru em:

    * Kafka, RabbitMQ, SQS, ou
    * tabela **Inbox** no PostgreSQL
  * retorna HTTP `202 (Accepted)`

* O Worker Pool deve consumir dessa infraestrutura persistente

**Garantia:**
Falha de hardware não causa perda de dados, pois a mensagem permanece **não confirmada (un-ACKed)**

---

### FASE 3: ADAPTAÇÃO EXTERNA E ACL (Semanas 5-6)

#### 3.1 a 3.3 (Mantidos e Ampliados)

A Camada Anti-Corrupção (ACL):

* recebe payloads da mensageria persistente
* executa verificação criptográfica do gateway
* orquestra a FSM

O decaimento exponencial deve ser aplicado exclusivamente pelo consumidor da fila:

t_retry = (base × 2^n) + random_jitter

---

### FASE 4: TELEMETRIA SISTÉMICA (Semana 7)

#### 4.1. Tracing Distribuído (Novo)

* Integração obrigatória com **OpenTelemetry**
* Toda requisição gera um **TraceID**

O TraceID deve ser:

* propagado via `context.Context`
* salvo no banco de dados
* anexado às mensagens da fila
* injetado nos logs (`log/slog`)

#### 4.2. Documentação Operacional

Adicionar playbooks de recuperação de desastres, como:

* O que fazer se o gateway rotacionar chaves criptográficas de webhook inesperadamente
* Procedimentos de reprocessamento de eventos (Inbox/Outbox)
* Estratégias de fallback em falhas prolongadas de serviços externos
