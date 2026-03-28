package payment

import (
	"asaas_framework/internal/domain/entity"
	"errors"
	"fmt"
)

var (
	ErrIllegalTransition = errors.New("transição de estado ilegal")
)

// PaymentState defines the behavior for each payment state.
type PaymentState interface {
	Status() entity.PaymentStatus
	TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error)
}

// State structs
type pendingState struct{}
func (s pendingState) Status() entity.PaymentStatus { return entity.StatusPending }
func (s pendingState) TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error) {
	switch next {
	case entity.StatusPaid:
		// Em caso de pagamento, geramos um evento para o Outbox
		event := entity.NewOutboxEvent("id-gerado-pelo-infra", "PAYMENT_PAID", nil)
		return paidState{}, event, nil
	case entity.StatusFailed:
		return failedState{}, nil, nil
	case entity.StatusCanceled:
		return canceledState{}, nil, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, s.Status(), next)
	}
}

type paidState struct{}
func (s paidState) Status() entity.PaymentStatus { return entity.StatusPaid }
func (s paidState) TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error) {
	if next == entity.StatusRefunded {
		event := entity.NewOutboxEvent("id-gerado", "PAYMENT_REFUNDED", nil)
		return refundedState{}, event, nil
	}
	return nil, nil, fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, s.Status(), next)
}

type failedState struct{}
func (s failedState) Status() entity.PaymentStatus { return entity.StatusFailed }
func (s failedState) TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error) {
	if next == entity.StatusPending || next == entity.StatusCanceled {
		return pendingState{}, nil, nil
	}
	return nil, nil, fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, s.Status(), next)
}

type canceledState struct{}
func (s canceledState) Status() entity.PaymentStatus { return entity.StatusCanceled }
func (s canceledState) TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error) {
	return nil, nil, fmt.Errorf("%w: %s is a final state", ErrIllegalTransition, s.Status())
}

type refundedState struct{}
func (s refundedState) Status() entity.PaymentStatus { return entity.StatusRefunded }
func (s refundedState) TransitionTo(next entity.PaymentStatus) (PaymentState, *entity.OutboxEvent, error) {
	return nil, nil, fmt.Errorf("%w: %s is a final state", ErrIllegalTransition, s.Status())
}

type PaymentFSM struct {
	transaction *entity.Transaction
	state       PaymentState
}

func NewPaymentFSM(t *entity.Transaction) *PaymentFSM {
	return &PaymentFSM{
		transaction: t,
		state:       getStateFromStatus(t.Status),
	}
}

// TransitionTo tenta mudar o estado e retorna um eventual evento de domínio para o Outbox.
func (f *PaymentFSM) TransitionTo(next entity.PaymentStatus) (*entity.OutboxEvent, error) {
	newState, event, err := f.state.TransitionTo(next)
	if err != nil {
		return nil, err
	}
	f.state = newState
	f.transaction.Status = next
	return event, nil
}

func (f *PaymentFSM) GetCurrentStatus() entity.PaymentStatus {
	return f.state.Status()
}

func getStateFromStatus(status entity.PaymentStatus) PaymentState {
	switch status {
	case entity.StatusPending:
		return pendingState{}
	case entity.StatusPaid:
		return paidState{}
	case entity.StatusFailed:
		return failedState{}
	case entity.StatusCanceled:
		return canceledState{}
	case entity.StatusRefunded:
		return refundedState{}
	default:
		return pendingState{}
	}
}
