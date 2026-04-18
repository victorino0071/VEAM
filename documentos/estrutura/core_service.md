# Core: Service (Orquestrador de Negócio)
**Caminho:** `internal/core/service`

O `PaymentService` é o regente que coordena a interação entre o Repositório e a Entidade de Domínio.

## ⚖️ Transações ACID
Toda operação no Service ocorre dentro de uma transação de banco de dados (`ExecuteInTransaction`). Isso garante o **Atomic Commit**: ou salvamos o novo estado da transação E o evento de outbox, ou nada é salvo.

## 🔄 Fluxo de Processamento
1.  Busca a transação atual via Repositório.
2.  Invoca a **Sovereign FSM** (`tx.TransitionTo`).
3.  Se válida, persiste o novo estado e o evento de Outbox gerado.
