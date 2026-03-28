package payment

import (
	"asaas_framework/internal/domain/entity"
	"testing"
)

func TestPaymentFSM_TransitionTo(t *testing.T) {
	trans := &entity.Transaction{
		Status: entity.StatusPending,
	}
	fsm := NewPaymentFSM(trans)

	if _, err := fsm.TransitionTo(entity.StatusPaid); err != nil {
		t.Errorf("Esperado transição válida para PAID, obtido erro: %v", err)
	}

	if trans.Status != entity.StatusPaid {
		t.Errorf("Esperado status PAID, obtido: %s", trans.Status)
	}

	_, err := fsm.TransitionTo(entity.StatusPending)
	if err == nil {
		t.Error("Esperado erro para transição ilegal PAID -> PENDING, mas não ocorreu")
	}

	event, err := fsm.TransitionTo(entity.StatusRefunded)
	if err != nil {
		t.Errorf("Esperado transição válida para REFUNDED, obtido erro: %v", err)
	}

	if event == nil || event.EventType != "PAYMENT_REFUNDED" {
		t.Errorf("Esperado evento PAYMENT_REFUNDED, obtido: %v", event)
	}
}
