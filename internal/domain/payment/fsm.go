package payment

import (
	"asaas_framework/internal/domain/entity"
	"fmt"
)

// PaymentFSM define a interface para a máquina de estados (FSM).
type PaymentFSM interface {
	TransitionTo(next entity.PaymentStatus) (*entity.OutboxEvent, error)
	SetMetadata(metadata map[string]string)
}

type paymentFSM struct {
	tx       *entity.Transaction
	metadata map[string]string
}

func NewPaymentFSM(tx *entity.Transaction) *paymentFSM {
	return &paymentFSM{tx: tx}
}

func (f *paymentFSM) SetMetadata(metadata map[string]string) {
	f.metadata = metadata
}

func (f *paymentFSM) TransitionTo(next entity.PaymentStatus) (*entity.OutboxEvent, error) {
	switch f.tx.Status {
	case entity.StatusPending:
		if next == entity.StatusPaid {
			f.tx.Status = entity.StatusPaid
			return entity.NewOutboxEvent("event-id", "PAYMENT_CONFIRMED", []byte("payload"), f.metadata), nil
		}
		if next == entity.StatusFailed {
			f.tx.Status = entity.StatusFailed
			return entity.NewOutboxEvent("event-id", "PAYMENT_FAILED", []byte("payload"), f.metadata), nil
		}
	}
	return nil, fmt.Errorf("transição inválida de %s para %s", f.tx.Status, next)
}
