# Documentação de Estrutura: Handlers (Recepção HTTP)
**Caminho:** `internal/app/handler`

A pasta `handler` atua como a fronteira de Inbound (Entrada) do nosso sistema para comunicação síncrona/Web. Ela é responsável por interceptar requisições da rede, validá-las criptograficamente, lidar com telemetria e entregar o dado cru para persistência cega o mais rápido possível.

---

## 1. Visão Geral do Arquivo `webhook_handler.go`
Esse handler não processa lógica de negócio nem tenta entender o pagamento. Ele implementa o padrão "**Store and Forward**" (Salva rápido, processa depois) usando nosso Inbox, para que possamos responder ao sistema externo instantaneamente, evitando _timeouts_.

---

## 2. Construtores e Setup

#### `type WebhookHandler`
*   Depende exclusivamente do `port.Repository` (para salvar os dados) e de uma `accessToken` para verificação de segurança.

#### `NewWebhookHandler(repo, accessToken) *WebhookHandler`
*   Garante a injeção adequada destas duas dependências cruciais ao expor essa infraestrutura.

---

## 3. O Fluxo HTTP Principal

### `ServeHTTP(w http.ResponseWriter, r *http.Request)`
Este método obedece à assinatura padrão do Go para servir requisições Web. Vamos quebrar suas 5 etapas críticas:

#### Passo 1: Inicia Rastreamento (OpenTelemetry)
*   **O que faz e por que:** Assim que o pacote HTTP bate no servidor, a primeira linha chama a biblioteca do OpenTelemetry (`otel.Tracer`) para abrir um `span` nomeado `"ReceiveWebhook"`. Isso garante o inicio imediato da rastreabilidade da request, ideal para debug do tempo de CPU utilizado só para segurar aquela requisição web na porta exposta do framework.

#### Passo 2: Context Injection (Metadata Carrier)
*   **O que faz e por que:** Os _Trace IDs_ e informações sensíveis e invisíveis geradas no passo interior pelo APM/OpenTelemetry precisam caminhar pelo sistema. O handler injeta esses dados gerados globalmente dentro de um map puro em Go (`metadata := make(map[string]string)`).
*   Isso nos deixa agnósticos a lib de tracing para o processamento assíncrono posterior, pois passamos apenas as _keys_.

#### Passo 3: Verificação Criptográfica
*   **O que faz e por que:** É um escudo contra invasores de API. Ele busca um header bem específico (`"asaas-access-token"`) e compara se ele é igual ao esperado do nosso provedor confiável.
*   **Fail Fast:** Se for diferente, ele imediatamente aborta a operação reusando um HTTP puro (Retornando erro de "Unauthorized") e impedindo ataques de DOS de persistência.

#### Passo 4: Leitura do Payload e Identificador Único
*   **O que faz:** Captura de fato os "bytes" não convertidos contidos no Content-Body (`ioutil.ReadAll`) e um _header_ valioso `asaas-event-id`
*   **Fail Fast:** Aborta a execução caso ocorra um erro de serialização do body (garantindo que se enviaram lixo, nós não iremos nem guardar o lixo).

#### Passo 5: Ingestão Cega "Blind Ingestion" (Mastery Pattern)
*   **O que faz e por que:** Cria a nossa entidade `InboxEvent`. Passa tudo mastigado. E manda nosso `repo.SaveInboxEvent()`.
*   **A grande sacada:** Nós NÃO tentamos chamar a anti-corruption layer aqui. NÃO validamos se a transação existe no domínio. E muito menos abrimos transações complexas de domínio no banco. Nós apenas salvamos os Bytes recebidos do Asaas no Database e encerramos a request com o HTTP status Code de sucesso (ex: StatusAccepted 202).
*   **Objetivo de negócio:** A requisição demorou apenas o tempo da verificação de autoridade + tempo do Insert na tebela `inbox`. Portanto um pico absurdo com 1 milhoes de pagamentos simultâneos vao apenas escrever rapidamente na coluna os bytes, e voltar na hora, liberando as theads web do framework de colapsar. Não existe chance de "Timeout" de endpoint, salvando dinheiro.
