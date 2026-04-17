package payment

import (
	"asaas_framework/internal/domain/entity"
	"fmt"

	"github.com/google/uuid"
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
	// Se for exatamente o mesmo, é idempotente no domínio
	if f.tx.Status == next {
		return nil, nil
	}

	switch f.tx.Status {
	case entity.StatusPending:
		if next == entity.StatusPaid || next == entity.StatusReceived || next == entity.StatusConfirmed {
			f.tx.Status = next
			return entity.NewOutboxEvent(uuid.New().String(), "PAYMENT_CONFIRMED", []byte(f.tx.ID), f.metadata), nil
		}
		if next == entity.StatusFailed {
			f.tx.Status = entity.StatusFailed
			return entity.NewOutboxEvent(uuid.New().String(), "PAYMENT_FAILED", []byte(f.tx.ID), f.metadata), nil
		}
	case entity.StatusPaid, entity.StatusReceived, entity.StatusConfirmed:
		if next == entity.StatusRefundInitiated {
			f.tx.Status = entity.StatusRefundInitiated
			return entity.NewOutboxEvent(uuid.New().String(), "REFUND_STARTED", []byte(f.tx.ID), f.metadata), nil
		}
		if next == entity.StatusConfirmed || next == entity.StatusPaid {
			f.tx.Status = next
			return nil, nil // Ignora upgrades silenciosos na FSM
		}
	}

	// Sagas / Anomaly Handling: Se uma ordem bizarra chega (ex: PAID mas estava FAILED)
	// Em vez de explodir com Exception e travar a DLQ imediatamente, convertemos
	// para um estado de ANOMALY para triagem manual/assistida, mas aceitamos a transição internamente.
	f.tx.Status = entity.StatusAnomaly
	return entity.NewOutboxEvent(uuid.New().String(), "PAYMENT_ANOMALY", []byte(fmt.Sprintf("Invalid transition attempted: %s -> %s", f.tx.Status, next)), f.metadata), nil
}
