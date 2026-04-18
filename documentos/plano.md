### FASE 1: DEFINIÇÃO DE DOMÍNIO E MÁQUINA DE ESTADOS (Semanas 1-2)

O núcleo do sistema não deve saber que o Asaas existe. Esta fase isola as regras de negócio de fatores externos de entropia (falhas de rede, mudanças de API).

#### 1. Isolamento de Estado (Memento Pattern)

Refatoração em `/domain/entity/transaction.go`:

*   **[NEW]** `TransactionSnapshot`: Struct pública contendo o estado serializável da transação (Campos: ID, Status, Amount, etc.).
*   **[MODIFY]** `Transaction`: 
    *   Método `ToSnapshot() TransactionSnapshot`: Exporta o estado interno.
    *   Método `ApplySnapshot(TransactionSnapshot)`: Importa o estado de forma controlada.
*   **[DELETE]** `RebuildFromRepository`: Remoção da função que permitia mutação arbitrária direta via argumentos semânticos.

---

### 2. Flexibilidade do Domínio (Transition Policies)

*   **[NEW]** `domain/entity/policy.go`: 
    ```go
    type TransitionPolicy interface {
        ID() string
        Evaluate(ctx context.Context, tx *Transaction, targetState PaymentStatus) error
    }
    ```
*   **[MODIFY]** `Transaction`: O método `TransitionTo` agora itera sobre as políticas injetadas (Chain of Responsibility) antes de persistir a mudança de estado.

---

### 3. Desacoplamento de Execução (Intenção vs. Configuração)

Refatoração em `/engine.go` para remover a passividade mágica do `Start()` geral:

*   **[MODIFY]** `Engine`: Implementação de métodos terminais explícitos:
    *   `engine.ConsumeInbox(ctx context.Context) error`: Inicia o loop de processamento do Inbox.
    *   `engine.RelayOutbox(ctx context.Context) error`: Inicia o despachante do Outbox.
    *   `engine.NewWebhookHandler(providerID string) http.Handler`: Retorna o handler passivo.

---

### 4. CLI de Operações (Isolamento de Infraestrutura)

*   **[NEW]** `cmd/payment-cli/main.go`: Binário independente para tarefas administrativas.
*   Funcionalidade Inicial: `migrate up/down` consumindo os arquivos `//go:embed` do núcleo.
*   **[MODIFY]** `engine.go`: Remoção do `WithAutoMigrate()` em tempo de execução para prevenir contenção de locks DDL em escala.

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
