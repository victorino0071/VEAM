# Documentação de Estrutura: ACL (Anti-Corruption Layer) & Universal Webhook Translator
**Caminho:** `internal/app/acl`

A pasta `acl` implementa o padrão **Anti-Corruption Layer (Camada Anticorrupção)**. O objetivo desta camada é garantir o agnosticismo do motor, evitando que os modelos de dados de gateways externos (Asaas, Stripe, Mercado Pago) "vazem" e poluam o modelo de domínio principal.

---

## 1. O Conceito de ACL Universal
Ao receber um Webhook, cada provedor envia dados em formatos proprietários (campos como `billingType`, `value`, `netValue`). Se o nosso núcleo financeiro dependesse dessa estrutura, ficaríamos acoplados a cada API. A ACL atua como um "tradutor universal". 

Cada adaptador de gateway traduz o payload externo para as **Entidades do Domínio**, garantindo que o `PaymentService` sempre receba um objeto `Transaction` padronizado.

---

## 2. Detalhamento do `translator.go`

Diferente das versões anteriores, o tradutor agora recebe o contexto do provedor (`providerID`) para garantir rastreabilidade total desde o momento do parsing.

#### `ToDomain(dto gateway.WebhookDTO, providerID string) (*entity.Transaction, error)`
*   **O que faz:** Função central de conversão.
*   **Lógicas Analíticas Internas:**
    1.  **Parsing Seguro:** Converte datas e valores financeiros para tipos nativos de alta precisão do Go.
    2.  **Injeção de Metadados:** Atribui o `ProviderID` à transação. Isso é vital para que o sistema saiba qual gateway deve processar estornos ou consultas futuras para aquela cobrança específica.
    3.  **Mapeamento de Status:** Converte status externos (ex: `"RECEIVED"`, `"CONFIRMED"`, `"SUCCEEDED"`) para a taxonomia única do nosso domínio (`entity.StatusReceived`).

---

## 3. Por que isso é vital para o Pivot Open Source?

Com esta estrutura, o framework pode suportar dezenas de gateways simultâneos. Se incluirmos o Stripe amanhã, basta criarmos um mapeamento na ACL. Os casos de uso e a Máquina de Estados (FSM) permanecerão ilesos, pois eles só "conversam" com a linguagem traduzida pela ACL.

---

> [!NOTE]
> A ACL é a barreira final de imunidade. Um dado JSON selvagem entra na fronteira e um dado altamente validado/estruturado é escoado livre de corrupções para a Camada de Serviço.
