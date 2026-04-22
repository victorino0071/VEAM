package entity

import "context"

// TransitionPolicy define o contrato para validações de mudança de estado.
// Implementações podem ser simples (regras de FSM) ou complexas (checagem de fraude).
type TransitionPolicy interface {
	// ID retorna o nome canônico da política para logs e auditoria.
	ID() string
	// Evaluate verifica se a transição é permitida.
	// Retorna um erro caso a transição viole a regra de negócio.
	Evaluate(ctx context.Context, tx *Transaction, targetState PaymentStatus) error
}

// DefaultTransitionPolicy implementa a FSM financeira padrão do motor.
type DefaultTransitionPolicy struct{}

func (p *DefaultTransitionPolicy) ID() string { return "default_fsm_policy" }

func (p *DefaultTransitionPolicy) Evaluate(ctx context.Context, tx *Transaction, targetState PaymentStatus) error {
	current := tx.Status()
	
	// Idempotência
	if current == targetState {
		return nil
	}

	switch current {
	case StatusPending:
		if targetState == StatusConfirmed || targetState == StatusReceived || targetState == StatusPaid || targetState == StatusFailed {
			return nil
		}
	case StatusPaid, StatusReceived, StatusConfirmed:
		if targetState == StatusRefundProcessing {
			return nil
		}
	case StatusRefundProcessing:
		if targetState == StatusRefunded || targetState == StatusRefundFailed {
			return nil
		}
	}

	return ErrIllegalTransition
}

var ErrIllegalTransition = &DomainError{Code: "illegal_transition", Message: "fsm_violation: transição de estado não permitida pela política"}

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }
