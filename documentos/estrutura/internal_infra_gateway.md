# Documentação de Estrutura: Gateway HTTP
**Caminho:** `internal/infra/gateway`

A pasta `gateway` contém os "Adaptadores" reais que saem do nosso servidor e trafegam pela internet para bater em APIs de terceiros. Seu principal componente é o `AsaasAdapter`.

---

## 1. O que é um Adapter (Adaptador)?
*( **Conceito Técnico - Adapter Pattern:** Na engenharia de software, um adaptador é um pedaço de código que pega os dados da forma que o nosso sistema interno espera e "traduz" / "embala" para o formato que um sistema externo exige. Pense nele mecânicamente como um adaptador de tomada universal: ele conecta o plug do nosso Domínio no padrão da parede do Asaas. )*

---

## 2. Visão Geral do `asaas_adapter.go`

O arquivo contém a estrutura básica para a comunicação HTTP com a provedora financeira.

#### `type AsaasAdapter struct`
Armazena credenciais ativas para invocar recursos da rede:
*   `apiKey`: A chave de acesso criptográfica para autenticação na API do Asaas.
*   `baseUrl`: O endereço raiz da API (ex: `https://sandbox.asaas.com/api/v3`).
*   `httpClient`: *( **Conceito Técnico - http.Client:** Diferente de abrir uma conexão crua na unha, cliente HTTP injetável em Go possui pools nativos de reaproveitamento TCP e servem para forçar timeouts seguros para evitar prender as threads do nosso App se a internet do Asaas cair )*

---

## 3. A Estrutura de Retransmissão Real (Fase Finalizada)
A `Phase 3` de desenvolvimento foi concluída, e o sistema não usa mais "Mocks" (imitações falsas de banco/transações). O Gateway é uma estrutura de tráfego pesado operando conexões ativas na internet.

#### A Coesão dos `asaas_dtos.go`
Enquanto o App usa as `entities` de Domínio abstratas e a camada ACL valida webhooks, o *Gateway de Saída* tem seus próprios DTOs (Objetos de Transferência de Dados).
Exemplo: `TransactionRequest`. Eles não contém lógicas, são apenas *structs* rasas preenchidas com as formatações (`json:"billingType"`) exatamente iguais à documentação web oficial da provedora Asaas.

#### Assinaturas Reais do `asaas_adapter.go`:
Todas as assinaturas forçadas por `port.GatewayAdapter` operam requisições massivas seguras:
1.  **`RefundTransaction(ctx context.Context, transactionID string) error`**: Ele aloca o ID da transação em uma URL Restful nativa (`https://sandbox.asaas.com/api/v3/payments/{id}/refund`). Usa o `http.NewRequestWithContext` para criar pacotes TCP com limites rígidos de _timeout_.
2.  **Autorização Imbutível**: Sempre injeta no Header o `'access_token': apiKey`, lido nativamente via `os.Getenv` no Boot principal do `main.go`.
3.  **Tratamento Nativo de Erros 400+**: O cliente HTTP oficial do GoLang considera "Sucesso" sempre que a conexão não caiu. A nossa inteligência aqui faz a auditoria se o `resp.StatusCode` é `>= 400`, criando retornos de erros formatados com decodificação avançada do Payload de resposta do Asaas (`ErrorResponse` DTO) para que os Circuit Breakers possam trabalhar matemática sob-erros catalogados.
