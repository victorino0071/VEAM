# Documentação de Estrutura: Engine (Bootstrapping Builder)
**Caminho:** `internal/app/engine`

O arquivo `engine.go` reside no coração do orquestrador e implementa o padrão **Builder Pattern**. Seu objetivo é centralizar a fiação (wire-up) complexa de todas as camadas do sistema, fornecendo uma API fluída e simples para o desenvolvedor.

---

## 1. O Conceito de Builder Pattern
*( **Conceito Técnico - Builder Pattern:** É um padrão de design que permite a construção passo-a-passo de objetos complexos. No nosso caso, o `Engine` precisa de conexões com o banco, carregamento de segredos, registro de provedores e ativação de workers. Ao invés de passar 15 argumentos para um construtor, usamos métodos encadeados que configuram o objeto gradualmente. )*

---

## 2. Visão Geral da Estrutura `Engine`

O `Engine` encapsula os componentes principais para que o `main.go` precise apenas dar comandos de alto nível:

*   **Repo:** Instância do repositório PostgreSQL.
*   **Registry:** Central de adaptadores de gateways cadastrados.
*   **Service:** O motor de lógica de pagamentos.
*   **Consumer/Relay:** Os processos de background para Inbox/Outbox.
*   **Breaker:** O sistema de resiliência global.

---

## 3. Métodos de Configuração (Fluent API)

#### `NewEngine(db *sql.DB) *Engine`
*   **O que faz:** Inicializa a fundação do motor sobre uma conexão Postgres crua. Ele monta a fiação interna inicial sem expor as entranhas para o usuário.

#### `WithAutoMigrate() *Engine`
*   **O que faz:** Aciona o motor de migrações versionadas. É o "Big Bang" que garante que as tabelas necessárias existam antes da aplicação rodar.

#### `WithTelemetry(serviceName string) *Engine`
*   **O que faz:** Atacha a observabilidade (OpenTelemetry) no motor.

#### `RegisterProvider(id string, adapter port.GatewayAdapter) *Engine`
*   **O que faz:** Acopla um novo gateway (ex: Asaas, Stripe) ao roteador lógico. Isso possibilita o sistema trabalhar com múltiplos gateways de forma agnóstica.

#### `Start(ctx context.Context)`
*   **O que faz:** Desperta os Workers (Inbox Consumer e Outbox Relay) em goroutines separadas, iniciando de fato a operação de resiliência.

---

> [!IMPORTANT]
> O `Engine` é a única peça que conhece "todo mundo". Ele liga o banco ao serviço, o serviço ao worker, e o worker ao disjuntor. Isso mantém o `main.go` como um simples arquivo de configuração.
