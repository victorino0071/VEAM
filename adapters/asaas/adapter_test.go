package asaas

import (
	"testing"
)

func TestAdapter_Fingerprint_Stability(t *testing.T) {
	adapter := NewAdapter("key", "secret", "url")
	
	// Payload 1: ID de entrega A, Timestamp T1
	payload1 := []byte(`{
		"id": "evt_AAAAA",
		"event": "PAYMENT_RECEIVED",
		"payment": {
			"id": "pay_123",
			"status": "RECEIVED",
			"value": 150.50,
			"dueDate": "2023-12-31"
		}
	}`)

	// Payload 2: ID de entrega B, Timestamp T2 (Ruído de transporte)
	payload2 := []byte(`{
		"id": "evt_BBBBB",
		"event": "PAYMENT_RECEIVED",
		"payment": {
			"id": "pay_123",
			"status": "RECEIVED",
			"value": 150.50,
			"dueDate": "2023-12-31"
		}
	}`)

	f1, err := adapter.Fingerprint(payload1)
	if err != nil {
		t.Fatalf("Erro ao gerar fingerprint 1: %v", err)
	}

	f2, err := adapter.Fingerprint(payload2)
	if err != nil {
		t.Fatalf("Erro ao gerar fingerprint 2: %v", err)
	}

	if f1 != f2 {
		t.Errorf("Fingerprints diferentes para o mesmo conteúdo de negócio! \nF1: %s \nF2: %s", f1, f2)
	}
}
