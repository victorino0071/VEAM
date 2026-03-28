### FASE 1: DEFINIÇÃO DE DOMÍNIO E MÁQUINA DE ESTADOS (Semanas 1-2)

O núcleo do sistema não deve saber que o Asaas existe. Esta fase isola as regras de negócio de fatores externos de entropia (falhas de rede, mudanças de API).

#### 1.1. Modelagem de Agregados (DDD)

Defina as entidades núcleo (`Transaction`, `Customer`, `Subscription`) usando structs limpas.

#### 1.2. Implementação da FSM (Finite State Machine)

* Construa um motor de transição de estados para o ciclo de vida do pagamento.

**Critério de Sucesso:**
Transições ilegais (ex: `Failed` → `Paid`) devem resultar em erros gerados em runtime e devem ser idealmente restringidas na lógica de compilação através de interfaces de estado.

#### 1.3. Contratos de Interface

Defina as portas (Ports) que a camada de infraestrutura terá que satisfazer.
Crie as interfaces:

* `GatewayAdapter`
* `IdempotencyStore`
* `WebhookHandler`

---

### FASE 2: MOTORES DE RESILIÊNCIA E CONCORRÊNCIA (Semanas 3-4)

Aqui reside a demonstração do seu domínio sobre o runtime do Go. Este é o escudo do sistema.

#### 2.1. Middleware de Idempotência

* Crie um interceptor que inspeciona ou gera chaves únicas (`Idempotency-Key`) por transação.
* Implemente locks distribuídos (ou em memória) para garantir que duas goroutines processando o mesmo webhook simultaneamente não dupliquem a operação de banco de dados.

#### 2.2. Algoritmos de Retry e Circuit Breaker

* Implemente Exponential Backoff com Jitter para lidar com Rate Limiting do gateway.

Fórmula:
t_retry = (base × 2^n) + random_jitter

* Construa um Circuit Breaker que corta requisições externas imediatamente se a taxa de falha da API ultrapassar um limite crítico pré-definido.

#### 2.3. Worker Pool Assíncrono para Webhooks

* Projete um pool de goroutines consumidoras que leem de um buffered channel.
* Implemente Graceful Shutdown utilizando `context.Context` e `os/signal`.

**Regra crítica:**
O processo não pode ser finalizado até que:

* o canal de webhooks seja esvaziado, ou
* um timeout rígido seja atingido

(evita perda de dados financeiros)

---

### FASE 3: A CAMADA DE INFRAESTRUTURA E ADAPTAÇÃO (Semanas 5-6)

Apenas nesta fase o Asaas é introduzido. Ele é tratado como um módulo substituível e não confiável.

#### 3.1. Cliente HTTP Customizado

* Escreva um wrapper sobre o `net/http`
* Injete automaticamente:

  * headers de autenticação
  * motor de retry desenvolvido na Fase 2

#### 3.2. Mapeamento de Contratos (DTOs)

* Crie structs específicas para o payload do gateway.
* Faça o `Unmarshal` do JSON nessas structs.
* Converta para as entidades de domínio da Fase 1.

#### 3.3. Tradução de Webhooks

* O adaptador deve:

  * receber eventos específicos (ex: `PAYMENT_RECEIVED`)
  * validar assinaturas criptográficas
  * traduzir para eventos de domínio genéricos compreendidos pela FSM

---

### FASE 4: VETORES DE INSTRUMENTAÇÃO E LANÇAMENTO (Semana 7)

Um sistema silencioso é um sistema cego. A ausência de observabilidade anula a percepção de profundidade técnica.

#### 4.1. Logging Estruturado

* Utilize `log/slog` (Go 1.21+)
* Todo erro crítico deve conter:

  * stack trace
  * contexto da transação

#### 4.2. Documentação Defensiva

O `README.md` deve ir além de "como instalar".

Deve conter:

* diagramas de arquitetura
* explicações formais sobre consistência do sistema

Inclua:

* explicação do problema de idempotência em sistemas distribuídos
* como o sistema resolve esse problema
