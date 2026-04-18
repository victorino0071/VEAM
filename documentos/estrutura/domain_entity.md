# Domínio: Entidades e Soberania de Estado
**Caminho:** `documentos/estrutura/domain_entity.md`

Este documento detalha o modelo de dados e a blindagem lógica do Payment Engine. Seguimos o padrão de **Entidades Ricas** em detrimento de modelos anêmicos, garantindo que as regras de negócio residam onde os dados estão.

---

## 🛡️ Opaque State (Encapsulamento Soberano)

Diferente de sistemas convencionais onde campos de estado são exportados, a entidade `Transaction` utiliza um **Opaque State** para garantir a integridade financeira:

-   **Campo `status` (Privado):** Impossibilita que camadas de infraestrutura (como adaptadores ou repositórios) alterem o estado arbitrariamente.
-   **Getter `Status()`:** Apenas leitura é permitida externamente.

## ⚙️ A Máquina de Estados Soberana (TransitionTo)

Toda mutação de estado transacional é centralizada no método `TransitionTo(newState, metadata)`. Este método age como o **Único Ponto de Verdade**:

```go
func (t *Transaction) TransitionTo(newState PaymentStatus, metadata map[string]string) (*OutboxEvent, error)
```

### Regras de Transição (Exemplo FSM)
-   `PENDING` -> `PAID`: Válido (Gera evento `PAYMENT_CONFIRMED`).
-   `PAID` -> `FAILED`: **Inválido** (Bloqueado pela FSM).
-   `PAID` -> `REFUND_INITIATED`: Válido (Inicia trilha de auditoria).

### Geração de Outbox Atômico
Ao realizar uma transição válida, a entidade retorna automaticamente um `OutboxEvent`. Isso garante que:
1.  A mudança de estado no banco ocorra.
2.  O evento de integração seja disparado (Ex: notificar o ERP do cliente).
3.  O rastro de metadados (W3C Trace Context) seja preservado.

---

## 🏗️ Reconstrução Segura (Hydration)

Para permitir que o repositório carregue dados do banco sem violar a FSM, utilizamos o padrão de **Rebuild**:

```go
func RebuildFromRepository(...) *Transaction
```
Este método é de uso exclusivo da camada de infraestrutura para "reidratar" objetos salvos, não devendo ser usado para criar novas transações de negócio.

---

## 🧩 Principais Entidades
1.  **Transaction**: O coração do motor.
2.  **Customer**: Representação do pagador.
3.  **Outbox/InboxEvent**: Entidades de transporte para resiliência assíncrona.
