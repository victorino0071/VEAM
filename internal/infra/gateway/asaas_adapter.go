package gateway

import (
	"context"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
	"fmt"
	"net/http"
)

type AsaasAdapter struct {
	apiKey     string
	baseUrl    string
	httpClient *http.Client
}

func NewAsaasAdapter(apiKey string, baseUrl string) port.GatewayAdapter {
	return &AsaasAdapter{
		apiKey:     apiKey,
		baseUrl:    baseUrl,
		httpClient: &http.Client{},
	}
}

func (a *AsaasAdapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	// Mock implementation for Phase 3
	fmt.Printf("[Asaas] Creating Customer: %s\n", customer.Name)
	return "cus_mock_123", nil
}

func (a *AsaasAdapter) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	fmt.Printf("[Asaas] Creating Transaction: %f for %s\n", tx.Amount, tx.CustomerID)
	return "pay_mock_456", nil
}

func (a *AsaasAdapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	fmt.Printf("[Asaas] Getting state for: %s\n", externalID)
	return entity.StatusPending, nil
}

func (a *AsaasAdapter) RefundTransaction(ctx context.Context, transactionID string) error {
	fmt.Printf("[Asaas] Refunding: %s\n", transactionID)
	return nil
}
