# Documentação de Estrutura: Repositório e SQL
**Caminho:** `internal/infra/repository`

A pasta `repository` é onde se reside todo o poder duro de alvenaria do Framework de fato: Manipulação SQL extrema para alta resiliência de cluster. O arquivo principal `postgres_repository.go` abraça inteiramente essas filosofias mecânicas em alto desempenho, adaptabilizando as interfaces cegas contruídas globalmente pelos _Ports_.

---

## 1. Concorrência e Blind Ingestion

### A) Ingestão Assíncrona e Prevenção de Duplicatas (Idempotência Nativa)
`SaveInboxEvent(ctx, event)` 
Possui um código base focado altamente no _sqlc_ engine. Aqui vemos o comando extremo: `INSERT INTO inbox (...) VALUES (...) ON CONFLICT DO NOTHING`.

*( **Conceito Técnico - ON CONFLICT DO NOTHING (UPSERT):** Nos banco de dados de alta perfomance moderna (como Postgres), essa flag serve para gerenciar concorrência bruta "Upserting". Imagine que o banco Asaas enlouqueceu e atirou 3 Webhooks simultâneos referenciando o mesmo pagamento pra nós. Quando eles engarrafarem tentando injetar o mesmo "Event ID" ao mesmo momento no BD, o banco vai permitir apenas que o 1º crie fisicamente o espaço, avisando aos outros dois processos silenciosamente para "Fazer(DO)" absolutamente "Nada(NOTHING)", ao inves de cuspir exceções e matar e fechar nossa aplicação. É resiliência no lado do dado e não no código. )*

---

## 2. A Coreografia Paralela (Padrão de Fases / Claim)

### B) Seleção Avançada em Lotes 
`ClaimInboxEvents(ctx, limit)` | `ClaimOutboxEvents(ctx, limit)`

Trabalham na "Phase A" documentada nos workers, provendo concorrência baseada em banco puro com o comando: `SELECT FOR UPDATE SKIP LOCKED`.

*( **Conceito Técnico - SELECT FOR UPDATE SKIP LOCKED:** Isso é o Segredo de Escalabilidade Horizontal Infinita do projeto. Num cenário normal do passado (como em MySQL antigo), se o nosso App possuisse simultaneamente 5 Workers rodando juntos na Amazon, e um deles fosse ler no BD a lista "Quais mensagens pendentes eu tenho para processar hoje?". Ele travaria aquela tabela pro resto do mundo esperar, os demais 4 works ficariam sentados de braços cruzados consumindo seu dinheiro da fatura da nuvem sem render na fila. Como injetamos o `SKIP LOCKED`, caso isso ocorra, o banco dezenha nativamente: "Filho o Lote XYZ a outra maquina ja bloqueou, 'PULE ESTE LOTE (Skip Locked)' e leia adiante para as da próxima rua!". Assim as "Vans (Seus Workers/Containers)" recolhem cada uma instantaneamente as parcelas distintas do Banco operando em 100% de CPU sem ninguem aguardar por ninguem. )*

---

## 3. Gestão e Transações 

### C) Orquestração atômica 
`ExecuteInTransaction(ctx, fn func(ctx context.Context) error) error`

É responsável absoluto por agraciar a malha base do PaymentService orquestrada. 

*( **Conceito Técnico - Transações ACID:** ACID é um sigla que no português descreve coisas Indivisíveis (Atômicas), Consistentes (Constantes e sem lixo), Isoladas do resto ambiente caótico e Duráveis. Na vida natural, quando processamos uma venda de cartão, nos devemos salvar os novos centavos numa Tabela "A" e atualizar a pendencia dela na Tabela "B" via um Evento. Se voce salvou na Tabela "A", e exatos 500 milisegundos a energia elétrica cai do data-center da Amazon, voce fodeu as contas, pois na "B" numca existiu o alerta que mandava a caixa pro sedex do cliente. Padrões transacionais criam um bolha atômica onde as ações sao enfileiradas num Limbo antes da gravação de fato, onde o HD do banco as escreve simultaneamente. Se ao gravar falha 1 dos passos (seja `SaveOutboxEvent`), é ativado o conceito de "**Rollback**", que é literalmente mandar o SQL esquecer tudo oque fez nas linhas do Banco naquele fechamento ali e voltando ao exato microestágio do espaço num segundo em tempo limpo pretérito anterior da requisição. )*
