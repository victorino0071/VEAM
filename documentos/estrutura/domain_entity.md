# Domínio: Entidades e Soberania de Estado
**Caminho:** `documentos/estrutura/domain_entity.md`

Este documento detalha o modelo de dados e a blindagem lógica do Payment Engine. Seguimos o padrão de **Entidades Ricas** com **Soberania de Estado** absoluta.

---

## 🛡️ Opaque State & Memento Pattern

Diferente de sistemas convencionais onde campos de estado são exportados, a entidade `Transaction` utiliza o padrão **Memento** para garantir a integridade financeira e evitar vazamento de fronteiras:

-   **Estado Privado (`status`):** O campo status é minúsculo e inalcançável fora do pacote comercial.
-   **`TransactionSnapshot`:** Uma struct pública e imutável que representa o estado serializável da transação.
-   **`ToSnapshot()` & `RestoreTransaction(s)`:** O Snapshot exporta o estado imutável. A reidratação ocorre exclusivamente via a fábrica estática `RestoreTransaction`, que garante a auto-injeção de políticas de defesa no "nascimento" da entidade na memória. O antigo método `ApplySnapshot` foi removido para impossibilitar mutações em runtime.

## ⚙️ A Máquina de Estados Soberana (TransitionTo)

Toda mutação de estado transacional é centralizada no método `TransitionTo`. Este método não possui lógica "chumbada", mas delega a validação a uma corrente de políticas.

```go
func (t *Transaction) TransitionTo(ctx context.Context, newState PaymentStatus, eventID string, metadata map[string]string) (*OutboxEvent, error)
```

### 💉 Injeção de Políticas (TransitionPolicy)
A rigidez do motor foi mitigada pela interface `TransitionPolicy`. Isso permite que o domínio seja adaptável:

```go
type TransitionPolicy interface {
    ID() string
    Evaluate(ctx context.Context, tx *Transaction, targetState PaymentStatus) error
}
```

Dessa forma, regras de antifraude, auditoria ou fluxos de captura bifásica podem ser injetados na entidade sem alterar seu código base.

### Geração de Outbox Atômico
Ao realizar uma transição validada por todas as políticas, a entidade retorna um `OutboxEvent`. Isso garante a atomicidade entre a mudança de estado e a notificação aos sistemas periféricos.

---

## 🧩 Principais Entidades
1.  **Transaction**: O agregado principal com FSM injetável.
2.  **Customer**: Representação do pagador.
3.  **Outbox/InboxEvent**: Entidades de transporte para resiliência assíncrona.
