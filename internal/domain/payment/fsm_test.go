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
	metadata := map[string]string{"traceparent": "00-test-trace-01"}
	fsm.SetMetadata(metadata)

	// 1. Pending -> Paid (Válida)
	event, err := fsm.TransitionTo(entity.StatusPaid)
	if err != nil {
		t.Errorf("Esperado transição válida para PAID, obtido erro: %v", err)
	}

	if trans.Status != entity.StatusPaid {
		t.Errorf("Esperado status PAID, obtido: %s", trans.Status)
	}

	if event == nil || event.Metadata["traceparent"] != "00-test-trace-01" {
		t.Errorf("Esperado Metadata no evento, obtido: %v", event)
	}

	// 2. Paid -> Pending (Inválida)
	_, err = fsm.TransitionTo(entity.StatusPending)
	if err == nil {
		t.Error("Esperado erro para transição ilegal PAID -> PENDING, mas não ocorreu")
	}
}
