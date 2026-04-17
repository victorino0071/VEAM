# Documentação de Estrutura: Observabilidade e Telemetria
**Caminho:** `internal/infra/telemetry`

A pasta `telemetry` contém infraestrutura puramente cross-cutting (transversal ao projeto inteiro). O objetivo do arquivo `telemetry.go` é injetar inteligência militar de infraestrutura à logistica dos fluxos que perpassam tudo entre o App e as APIs na rede invisivel.

---

## 1. Telemetria Ativa 

O arquivo `telemetry.go` orquestra implementações maciçamente poderosas em rastreabilidade inter-planetária.

*( **Conceito Técnico - Tracing Distribuído / OpenTelemetry:** Imagine um galpão gigante no correio onde todas a milhões de embalagens (Requests/Ações) passam por lá. Antigamente a gente via eles entrando e magicamente dando erros na caixa la do final onde são despachados num "Log escrito na tela de Erros". Tracing é literalmente selar numa etiqueta (Span ID Context) do remetente la no primeirissimo momento da recepção ou botão HTTP do framework Web, e acompanhar de câmera o tempo de estadia exato e trajeto desta mesma caixinha passando por (Handler -> Service FSM -> SaveTransaction BD -> OutboxRelay BD -> Asaas Web). A tela de monitoramento/telemetria na Amazon/Datadog formará um linha única rastreável de "Cascata de tempo", escancarando milimetricamente qual classe milisegundo a milisegundo esta atrapalhando e atrasando ou falhando a Request base principal que começou aquele processamento. )*

*( **Conceito Técnico - Propagation/Carrier and MapCarrier (Citados na ACL e Workers):** Esses são os transportadores reais das mochilas com tags em cima de tudo que roda por trás. Para não precisar colocar 5 variaveis na assinatura de uma sub-funçao atômica em loop, usamos injetores OTel Contexts transparentes para acarragar Trace-IDs do HTTP lá na parede para todo App, permitindo monitoramento. )*

---

## 2. Injetores e Controladores Atômicos em Go

Toda base complexa exige boot central assincrono e seguro.

### `InitTelemetry(serviceName string)`

No contexto arquitetural e cibernético em ambiente de servidores (Onde as invocações da CLI/`main.go` levantam vários plugins antes de escutar de fato) este boot usa mecânicas inteligentes em seu processo de ligação base do _provider_:

*( **Conceito Técnico - sync.Once no Go:** Note o uso do `once.Do(func() { ...` neste arquivo. A lib padrão Go traz esse tipo poderoso para impedir duplicados em infraestrutura de alto peso. Essa instrução significa: Múltiplas Goroutines podem tentar chamar e inicializar infinitas telemetrias do zero, eu prometo (A biblioteca sync da linguagem garante!) que toda esse pedacinho de inicialização do APM exportador JSON rodará extamente uma exclusivíssima vez por rodada de memória de máquina ligada de servidor inteiro, poupando a maquina dos erros grotescos e memory-leaks repetidos da arquitetura mal programada nas threads infinitas de chamadas erradas em Background de frameworks pesados no ambiente asincrono! )*

### O Agregador de Exportadores em Lote
Finaliza invocando as primitivas cruciais da engine da linguagem `sdktrace.WithBatcher(exporter)` 

*( **Conceito Técnico - SDK Batcher Export/Streaming de Dados:** Se cada `log/trace` do seu cliente clicar no pagamento despachasse imediatameente 1 mensagem pelo Wi-Fi ou conexão TCP pra nuvem de monitoramento, voce dobraria perigosamente as portas da AWS e aumentaria lag de internet por enviar bytes pífios um atrás do outro pela tubulação fina. Um "Batcher" de Observabilidade acumula de forma contínua por N seg em buffer interno do motor, e "vomita na nuvem de log" um caixotasso zipado num mega Byte de tamanho empilhado gigante contendo 5 mil traces atrasados, salvando de vez e liberando em assimetria massiva e alívio gigantesco o trafico na rota natural das engrenagens internas de TCP do SO Hospedeiro. )*
