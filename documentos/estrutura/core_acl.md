# Core: ACL (Anti-Corruption Layer)
**Caminho:** `internal/core/acl`

A ACL é a barreira de imunidade que protege o domínio contra a "contaminação" de DTOs e formatos de dados de terceiros.

## 🛡️ Propósito
Cada provedor (Asaas, Stripe, etc.) possui seu próprio formato de Webhook. A ACL abstrai essa complexidade, fornecendo um **Contrato Universal de Tradução**.

## ⚙️ Funcionamento
O `Translator` recebe o payload bruto (reivindicado do Inbox) e o converte em uma entidade de domínio blindada.
-   **Mapeamento de Status:** Converte status proprietários (ex: `RECEIVED`) para o `entity.PaymentStatus` unificado.
-   **Validação de Schema:** Garante que campos obrigatórios (ID, Valor) estejam presentes antes de tocar o domínio.
