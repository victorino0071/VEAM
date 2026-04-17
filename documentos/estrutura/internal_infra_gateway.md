# Documentação de Estrutura: Gateway & Adaptadores (Outbound APIs)
**Caminho:** `internal/infra/gateway`

A pasta `gateway` contém os Adaptadores que permitem ao motor se comunicar com o mundo exterior. Através de um contrato unificado, o sistema pode alternar entre provedores de pagamento mantendo a mesma lógica central.

---

## 1. O Conceito de Multi-Gateway (Agnosticismo)
Diferente de sistemas rígidos, o Payment Engine utiliza o padrão **Adapter Pattern** em conjunto com um **Provider Registry**.
-   **Adaptador:** Implementa a interface `port.GatewayAdapter`, traduzindo requisições do Domínio para o JSON específico da API (ex: Asaas, Stripe).
-   **Registry:** Funciona como um catálogo de conexões ativas. Durante o boot, registramos os adaptadores desejados e o motor resolve dinamicamente qual deles usar baseado no `ProviderID` da transação.

---

## 2. Visão Geral do `adapter.go`

O adaptador é o componente de execução de rede. Ele é responsável por:

*   **Autenticação:** Gerenciar chaves de API e tokens de acesso.
*   **Tráfego Seguro:** Realizar chamadas HTTP com timeouts e políticas de retry monitoradas pelo OpenTelemetry.
*   **Universal Webhook ACL:** Validar assinaturas digitais de webhooks recebidos e converter o payload bruto para o formato que a ACL entende.

---

## 3. Resiliência e Isolamento

#### Integração com o Circuit Breaker
O adaptador não trabalha sozinho. Toda chamada de rede passa pelo **Circuit Breaker** (Disjuntor). Se o gateway externo (ex: Asaas Sandbox) começar a falhar ou apresentar latência alta, o adaptador é "cortado" temporariamente pelo disjuntor para evitar o empilhamento de requisições no nosso servidor.

#### Tratamento de Erros de Domínio
O adaptador mapeia erros HTTP genéricos (400, 401, 429) em constantes que o nosso sistema entende. Isso permite que os Workers tratem de forma diferente um "Erro de Cartão Recusado" (Erro de Domínio - definitivo) de um "Erro de Timeout" (Erro de Rede - reentrável).

---

> [!TIP]
> **Adicionando Novo Provedor:** Para adicionar suporte ao Stripe ou Mercado Pago, basta criar uma nova pasta em `gateway/`, implementar os métodos da interface `GatewayAdapter` e registrá-lo no builder principal no `main.go`.
