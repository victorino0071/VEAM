# Documentação de Estrutura: Application Service
**Caminho:** `internal/app/service`

A pasta `service` acomoda a camada de "Aplicação". Em Arquiteturas Limpas (Clean Architecture), o Service é responsável orquestrar as execuções "macro". Ele é um _coordinator_ (regente de orquestra), que dita em que ordem nossos "Ports" com os BDs externos ("Database"/Repository) e regras internas (FSM) entrarão em ação de maneira atômica e segura.

---

## 1. O Arquivo `payment_service.go`

Este serviço contém as lógicas de mais alto grau de complexidade que de fato modificam estado persistente. Ele utiliza de todos os nossos conceitos definidos em `internal/domain`.

---

## 2. Detalhamento Estrutural

#### `type PaymentService struct`
*   Requer injeção de `port.Repository` e do coirmão direto para acesso à nuvem extrerna `port.GatewayAdapter`.

#### `ProcessPaymentWithMetadata(ctx context.Context, incomingTx *entity.Transaction, metadata map[string]string, nextStatus entity.PaymentStatus)`
É a operação primária de alteração de estados da aplicação toda. É ela quem rege e orquestra e delega responsabilidades sobre eventos.

**A orquestração perfeita se comporta desse jeito minucioso:**
1.  **Entrada transacional atômica:** Como essa modificação criará _side effects_ e mexerá fisicamente nos bytes gravados, a premissa base dele é abrir primeiramente `ExecuteInTransaction`. Uma promessa ACID (do BD) é repassada pelo Service via _closures_.
2.  **Adoção de Resiliência (`GetTransactionByID`)**: Estando na transação estrita, busca fisicamente a transação correspondente no banco. **Autodefesa:** Se a consulta retornar nu/vazio (ex: um Webhook `"PAYMENT_CREATED"` que chega pela primeira vez no sistema), o Service não explode; ele assume o objeto recém-traduzido `incomingTx` como entidade âncora, inicializando seu estado nativamente como `"PENDING"` antes das lógicas FSM atuarem.
3.  **Boot da Máquina de Estados**: Orquestra a injeção inicial usando as lógicas do pacote profundo (`payment.NewPaymentFSM`).
4.  **Vinculo de Metada/Context:** Diz expressamente à maquina de estados: "Guarde essa string de Trace ID atachada nesse movimento porque irei precisar deles no pattern da infra!" 
5.  **O Gatilho `TransitionTo`**:  Manda o comando rígido do negocio `TransitionTo(nextStatus)` que fará todas os "If" de garantias financeiras.
6.  **Persistir o núcleo original**: Confirma que as coisas funcionaram e manda "salvar" as novas coisas de volta ao HD  (`SaveTransaction`), na mesma malha.
7.  **Persistência Outbox**: Finalmente a orquestração captura o objeto abstrato abstrato final (`*OutboxEvent`) nascido por pura indução em Step(5) e solicita que o pacote do Repository o insira paralelamente _na mesma transação bancária do step(1)_. O evento Outbox (ex: `"PAYMENT_CONFIRMED"`) será enfileirado para os `Workers`.

**Em caso de erro na orquesta:** Como a orquestração opera via uma clojure injetada pelo `ExecuteInTransaction`, os "returns de Error" explodem silenciosamente no _closure_, obrigando a infraestrutura de Database adaptada que chame internamente o "Rollback" das querys no Postgres final.

---

> [!TIP]
> A grande essência e inteligência por trás dos `services` nesta arquitetura é que não há quase NADA de logica. Não fazemos contas de dinheiro aqui ou tomamos descisões do que um webhook pode alterar do Pix, toda essa parte lógica da dor de cabeça mora no `Domain/FSM`. Nós nos preocupamos quase que em montar com segurança um "Bloco Arquitetural" tipo Lego no código e empacotá-lo seguro para salvar nas nossas conexões, deixando o código maravilhosamente legível ao ser humano.
