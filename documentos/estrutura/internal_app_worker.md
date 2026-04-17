# Documentação de Estrutura: Workers (Processos em Background)
**Caminho:** `internal/app/worker`

A pasta `worker` concentra _Daemons_ (processos autônomos em background que se perpetuam em _Lopps_ de rotina infinita). O coração dessa dinâmica gira nas táticas de _Exponential Backoff_ adaptativas, aliadas na captura (`Claim`) de resiliências de infraestrutura baseadas em Inbox / Outbox, executando as tarefas pesadas de rede sem comprometer o fluxo original em que usuários reais navegam.

---

## 1. O Worker `InboxConsumer` (Entrada Assíncrona)
**Arquivo:** `inbox_consumer.go`

Pense nesse Worker como a esteira que tira devagar o que a montanha-russa do servidor HTTP joga o tempo inteiro e espremeu no BD via Handlers. 

### A Dinâmica Nuclear do Exponential Backoff
O loop `Start` é impulsionado por mecânicas elásticas. Se temos muito trabalho acumulado nas mensagens para processar, giramos rápido igual turbina, em _zero delays_. Mas quando os estoques zeram e nada mais há a fazer, seu sono duplica de período progressivamente `(500ms -> 1s -> 2s -> ... 30s)` até o limite, barrando o desespero e economizando gigantescamente o uso de CPU atoa da nossa Amazon AWS, até a base gerar fatos que mereçam o processamento real novamente.

### O Método `consume(ctx context.Context) int`
Ele define 3 "Phases" cruciais que ditam as proteções de concorrência global e blindagem de processamento cruzado no app.
1.  **PHASE A: Claim**. Utiliza a porta dos repositórios passando comandos como `ClaimInboxEvents(limit)`. Ele pede lote com _SKIP LOCKED_. É a blindagem para N contêineres rodando o app não entrarem em roubadas conjuntas num ambiente com Auto-Scaling Group.
2.  **PHASE B: O Trabalho Isolado (Background Exec)** 
    *   Faz o Hook-up da OpenTelemetry puxando aquele map _Tracer_ abandonado pelas websockets em Metadatas antigas nascendo o Carrier e reanimando nossa rastreabilidade.
    *   Diferente do Handler, o Worker desempacota o Payload real usando a poderosa `AsaasWebhookDTO` (da ACL).
    *   Sopra em `processEvent` as funções ativas do `PaymentService`, injetando diretamente o ponteiro `tx *entity.Transaction` na thread em silêncio absoluto.
3.  **PHASE C: Finalize**. Fecha a mensagem para compilar sucesso para `"COMPLETED"`, na query final paralisa de vez aquele "Event log" e o enterra com satisfações.


---

## 2. O Worker `OutboxRelay` (Saída Assíncrona e Resiliência)
**Arquivo:** `outbox_relay.go`

Este worker reage apenas às coisas geradas magicamente na conclusão das atividades dos nossos outros Services como `PaymentService`.

A diferença estrutural base deste componente, baseia-se na forte injeção da porta global defensiva: A `Resilience Port`. Enquanto o inbox lidou com o interno, o `Outbox` mexe a poeira tentando mandar notificações `push` e webhooks ou API requests pro Gateway externo que se encontram na web hostis por default.

### `Start` Aditivado + Fail-Fast Global
Ele implementa a mesma esteira adaptativa de _Exponential Backoff_ mencionada em cima, porém com super-poder reativo: Ejetores Frontais!
*   **Fail-Fast Loop:** A linha de teste `allowed, _ := r.breaker.Allow(ctx)` barra antes mesmo que 1 bit de banda seja desperdiçado do micro-serviço e pause a infra por de `baseT` e dá `continue`. Ele barra instantaneamente se as maquinas virtuais da própria Asaas colapsaram numa nuvem a distante milhas de nós, protegendo as baterias do DB e o processamento de tentar bater na cara num muro invisivel da porta TCP.

### Processamento de Expedição (`consumeBatch`)
Ele utiliza os preceitos de fases mas acopla interações vitais inteligentes:
*   Extrai a `TransactionID` originária embutida maliciosamente dentro do `Payload`.
*   Possui um roteador switch próprio:
    *   Se for um evento de estorno global (`REFUND_STARTED`), dispara `r.gateway.RefundTransaction`. Se der erro na rede, usa a retroalimentação vital `breaker.RecordResult` imediatamente para a janela base fixar "EWMA" estatístico para avaliar cortes no circuito.
    *   Se for meramente `"PAYMENT_CONFIRMED/FAILED"`, ele faz um Short-circuit (No-Op de Externalização) atuando como um barramento simples, pois tais flags dizem sobre sucesso em DB local e não chamadas extornativas.

---

> [!NOTE]
> Os workers do framework foram criados para serem infinitamente escaláveis. Isso nos possibilita subirmos `50.000` replicas dessas instâncias, e pela malha inteligente contida no _Layer DB "PHASE ONE C"_ somada à gestão reativa das Threads via _exponential fall-off_ gerada aqui, as instâncias jamais brigarão entre si ou travarão o banco principal de tabelas.
