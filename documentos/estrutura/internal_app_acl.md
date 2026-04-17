# Documentação de Estrutura: ACL (Anti-Corruption Layer)
**Caminho:** `internal/app/acl`

A pasta `acl` implementa o padrão **Anti-Corruption Layer (Camada Anticorrupção)**. O objetivo desta camada é evitar que os modelos de dados e nomenclaturas de sistemas externos (ex: APIs de gateways de pagamento) "vazem" e poluam o modelo de domínio principal da aplicação.

---

## 1. O Conceito de ACL
Quando o Asaas envia um Webhook, ele usa seus próprios nomes de variáveis (ex: `value` em vez de `amount`, `netValue`, etc) e aninha as estruturas dentro de objetos pais artificiais (O payload do webhook tem um cabeçalho de evento e só lá dentro é que reside as propiredades da transação de fato). Se o nosso núcleo financeiro dependesse diretamente dessa estrutura, ficaríamos fortemente acoplados à modelagem proprietária do Asaas. A ACL atua como um "tradutor universal", unmarshaling a linguagem externa e convertendo-a para nossas Entidades do Domínio.

---

## 2. Detalhamento do Arquivo `asaas_translator.go`

### A. Estruturas Externas (DTOs)

#### `type AsaasWebhookDTO struct`
*   **O que faz:** Representa a raiz primária do Payload do Webhook enviado pelo Asaas. Diferentemente da API passiva, o Webhook envelopa coisas.
*   **Campos de destaque:** Ele captura a chave `.event` raiz, e instancia um objeto hierárquico `Payment AsaasPaymentDTO` na chave `.payment` desempacotando dados blindados.

#### `type AsaasPaymentDTO struct`
*   **O que faz:** O extrato real do pagamento que o Asaas retorna (seja numa Response de criação via API, seja contido num envelopamento de Webhook).
*   **Campos de destaque:** `Customer`, `Value`, `Status`: Propriedades puramente externas. As struct tags (ex: `` `json:"value"` ``) garantem o parsing perfeito.

### B. Funções de Tradução

#### `ToDomain() (*entity.Transaction, error)`
*   **O que faz:** É a função central estagnada ao escopo do `AsaasPaymentDTO`.
*   **Lógicas Analíticas Internas:**
    1.  **Parsing Seguro:** Converte a `dueDate` (que vem como string `"2006-01-02"`) para objeto `time.Time` do Go.
    2.  **Mapeamento de Status:** Dispara uma avaliação usando o sub-método privado `mapAsaasStatus` para homogeneizar a taxonomia.
    3.  **Remoção de Vícios:** Note que no código nativo a gente opta por não chutar o designador de moeda `Currency = "BRL"` na força bruta — o domínio (Tabelas de BD e Use Cases) são os encarregados arquiteturais caso o sistema de multi-moedas seja adotado, mantendo a ACL enxuta.

#### `mapAsaasStatus(asaasStatus string) entity.PaymentStatus`
*   **O que faz:** Função auxiliar para mapeamento 1:1 rigoroso dos fluxos.
*   **Exemplos Reais de Tradução:**
    *   `"RECEIVED"` mapeia para nosso `entity.StatusReceived`.
    *   `"CONFIRMED"` evoca o designador transacional `entity.StatusConfirmed`.
    *   Mas veja a mágica da ACL em atípicos de negócio: Se o fluxo exibe `"OVERDUE"` ou `"FAILED"` por qualquer razão que seja do Asaas, não nos importa as minúcias fiscais exclusivas deles; nosso domínio capta tudo universalmente para `entity.StatusFailed`.
*   **Por que isso é vital?** Amanhã, se o Asaas decidir mudar a palavra `"CONFIRMED"` para `"PAYMENT_CONFIRMED"`, ou se incluirmos a API do *Mercado Pago* ao invés do Asaas, os casos de uso permanecerão ilesos.

---

> [!NOTE]
> A ACL é a barreira final de imunidade. Um dado Json selvagem entra na fronteira e um dado altamente validado/estruturado é escoado livre de corrupções para a Camada de Serviço.
