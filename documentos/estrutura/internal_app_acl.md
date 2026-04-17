# Documentação de Estrutura: ACL (Anti-Corruption Layer)
**Caminho:** `internal/app/acl`

A pasta `acl` implementa o padrão **Anti-Corruption Layer (Camada Anticorrupção)**. O objetivo desta camada é evitar que os modelos de dados e nomenclaturas de sistemas externos (ex: APIs de gateways de pagamento) "vazem" e poluam o modelo de domínio principal da aplicação.

---

## 1. O Conceito de ACL
Quando o Asaas envia um Webhook, ele usa seus próprios nomes de variáveis (ex: `value` em vez de `amount`, `netValue`, etc) e seus próprios status de pagamento (`RECEIVED`, `OVERDUE`). Se o nosso núcleo financeiro dependesse diretamente dessa estrutura, ficaríamos fortemente acoplados ao Asaas. A ACL atua como um "tradutor universal", recebendo a linguagem externa e convertendo-a para a nossa linguagem interna (Entidades do Domínio).

---

## 2. Detalhamento do Arquivo `asaas_translator.go`

### A. Estrutura Externa (DTO)

#### `type AsaasPaymentDTO struct`
*   **O que faz:** Representa exatamente o contrato de dados que a API ou o Webhook do Asaas nos envia (formato JSON).
*   **Campos de destaque:**
    *   `Customer`, `Value`, `Status`: Propriedades puramente externas. Note que as tags de struct (ex: `` `json:"value"` ``) garantem o parsing correto da rede direto para o DTO.

### B. Funções de Tradução

#### `ToDomain() (*entity.Transaction, error)`
*   **O que faz:** É a função central da ACL associada ao DTO. Ela mapeia o objeto que acabou de chegar pela internet (`AsaasPaymentDTO`) para a nossa entidade central de pagamentos (`entity.Transaction`).
*   **Objetivos e Lógicas Internas:**
    1.  **Parsing Seguro:** Tenta converter a `dueDate` (que vem como string no formato `"2006-01-02"`) para um objeto `time.Time` nativo da linguagem.
    2.  **Mapeamento de Status:** Invoca a função privada `mapAsaasStatus` para entender o que um status externo significa para nós.
    3.  **Padronização Dinâmica:** Exemplo clássico da ACL atuando: A ACL define à força (hardcode) que a `Currency` (moeda) será sempre `"BRL"`, pois o Asaas opera no Brasil, preenchendo uma lacuna que não vem no DTO de forma explícita, mas e que nosso domínio exige.

#### `mapAsaasStatus(asaasStatus string) entity.PaymentStatus`
*   **O que faz:** Função auxiliar (privada ao pacote) que traduz o status textual vindo do provedor externo em um de nossos enumeradores fortemente tipados (`entity.PaymentStatus`).
*   **Exemplos Reais de Tradução:**
    *   De fora recebemos `"RECEIVED"` ou `"CONFIRMED"`. A ACL diz ao nosso domínio: "Isso significa `entity.StatusPaid`".
    *   Se vem `"OVERDUE"` ou `"FAILED"`, ela traduz para o uníssono `entity.StatusFailed`.
*   **Por que isso é vital?** Amanhã, se o Asaas decidir mudar a palavra `"CONFIRMED"` para `"PAYMENT_CONFIRMED"`, nós não mexemos em **NADA** na lógica de pagamentos (na FSM), nos bancos de dados, ou nos casos de uso. Nós alteramos apenas a string neste único `switch` da ACL.

---

> [!NOTE]
> A ACL é a barreira final da fronteira de entrada. Um dado não confiavel (string json) entra e um dado altamente validado e estruturado logicamente sai pronto para consumo nas esferas profundas da aplicação.
