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

## 3. Mockagem e Fases de Implementação
Nesta etapa de concepção inicial, o `AsaasAdapter` atua utilizando uma técnica de estandes táticos chamada de **"Mock"**.

####  O que é Mocking?
*( **Conceito Técnico - Mock:** Mockar código é criar imitações baratas de funções caras. Num ambiente isolado, é inviável e perigoso bater num cartão de crédito de verdade e invocar a API de um banco real apenas para testar se nossa lógica local no FSM funciona. O Mock finge conectar na nuvem, escreve uma linha no console dizendo "Criei o Pagamento", devolve uma resposta perfeitamente parecida com a que a API retornaria (ex: ID `pay_mock_456`) e prossegue, tudo em zero milissegundos. )*

#### Assinaturas Implementadas:
Todas as assinaturas forçadas por `port.GatewayAdapter` operam sob mocking atrelados ao terminal:
1.  **`CreateCustomer`**: Devolve o placeholder `"cus_mock_123"`.
2.  **`CreateTransaction`**: Devolve o placeholder `"pay_mock_456"`.
3.  **`GetTransactionState`**: Simula que um pacote aleatório bateu na parede e voltou dizendo `entity.StatusPending`.
4.  **`RefundTransaction`**: Imprime alerta cego sem efeitos.

Em rodadas futuras no ambiente de Code Real (`Phase 3`), todas essas funções utilizarão a interface do Client para empacotar o JSON final com POST/GET reais via internet.
