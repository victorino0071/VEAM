# Documentação de Estrutura: Resilience Breaker
**Caminho:** `internal/infra/resilience`

A pasta `resilience` é sem dúvidas um dos códigos mais matemáticos e de baixo nível de performance de todo seu ecossistema. O arquivo `breaker.go` não delega inteligência a bibliotecas gratuitas. Ele cria sua própria matriz defensiva super focada e desenhada contra falhas assíncronas ativando funções `sync/atomic` e cálculos probabilísticos `EWMA`. Abaixo está o coração lógico detalhado dessas defesas.

---

## 1. Do que defende o "Circuit Breaker"?

O Circuit Breaker (Disjuntor) é uma adaptação tática de um disjuntor da casa de qualquer indivíduo normal.

*( **Conceito Técnico - Padrão Circuit Breaker:** Imagine que uma TV entrou em curto circuito e está sugando energia inifinita na mesma malha num corredor cheio de geladeiras e de micro-ondas ligados que não estao errados nem pifados. Por conta da ignorancia de defeito em curto apenas da TV, as faíscas causaram flutuações e agora explodiram todo mundo na sua cozinha em sequencia. Um disjuntor elétrico salva vidas lendo passivamente até onde o sistema pode suportar algo hostil, e abruptamente a energia é CORTADA para o braço defeituoso isolando os estragos, desarmando o curto, e religarndo automaticamente. O CircuitBreaker de SW protege exatamente que uma lentidão grotesca no `PIX/BancoCentral` comece a represar em rede nossa maquinas do Backend do `asaas_framework` de pagamentos, "Abrindo/Ativando o relé (StateOpen)", falhando num nanossegundo cada vez que seu web-request esbarrar pedindo por isso pro PIX, ate que "Estatísticamente" a falha desative e tudo se recupere lentamente. )*

---

## 2. Avaliação por Média Móvel: O Cálulo EWMA
Em vez de avaliar primitivamente se "Falhou nos últimos 5 segundos fecha" a matemática por trás deste circuito aplica `pFailure uint64` como tracking para reações progressivas (Táticas do Google).

*( **Conceito Técnico - EWMA (Exponentially Weighted Moving Average):** Diferentemente de uma média comum (ex: somei os ultimos 10 tiros deram falha / 10 = X% falha) o calculo `EWMA` é fundamentalmente sensível ao avanço do tempo. As falhas ou os sucessos de UM milisegundo atrás possem valor estatistico 10x mais pesados ou determinativos a falha, doque sucessos que houveram à uma hora do dia antigo de manha ou final da tarde de sabado. Ao rodar em Background a formula `(1-Alpha)*oldP + Alpha*result`, ele descobre probabilididade. Assim se o Asaas demorar muito mais do que devia pra acordar nesse milisegundo, o ponteiro de reatividade EWMA explode rapido, ativando as blindagens "OPEN" num intervalo agressivamente de relâmpagos. )*

---

## 3. Segurança de Baixo Nível: Golang `sync/atomic` e `sync.RWMutex`

Este calculo de Média Móvel é atualizado assincronamente por mais de quinhentas `threads/goroutines` simultâneas sem perder performância usando Locks passivos.

*( **Conceito Técnico - Mutex (Mutual Exclusion) e sync.RWMutex:** Se o workerA e WorkerB acessarem o ponteiro do disjuntor pra ver no GoLog se lemos O "StateOpen" exatamente simultâneos e alterarem simultanamente sua memoria sem Mutex o processo pode gerar uma pane Panic. A lib nativa de Go protege lendo via Read/Locks o estado, com R-Wait liberando a fila e enfileirando ordens simultâneas. )*

*( **Conceito Técnico - Atomic Operations / Tipos Atômicos:** A mais fascinante barreira do mundo computacional. Mesmo tendo o Mutex, se todos os 10 milhões requisos quiserem re-alterar O ESTADO PROBABILÍTICO da conta de math float/ewma do `CB` usando o Lock do Mutex, o servidor web formaria literalmente um túnel único de acesso, travando. Usar `atomic.CompareAndSwapUint64` ou as injeções em bitcast do Go, instruí literalmente ao processador Intel Core nativo da Amazon do datacenter saltar pelo SO operacionao p/ ir na Memoria L2 do cache, pegar 64 bits de ponto flutuante via Math/bits e sobrever na veia RAM a troca dos numeros simultaneamente e matematicamente livre de Dead/Lacks do programa concorrente! Isso confere performance atroz à defesas no mundo cibernético do `framework`. )*
