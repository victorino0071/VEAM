package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/port"
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

func (a *AsaasAdapter) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", a.baseUrl, path)
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("access_token", a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		json.Unmarshal(respBody, &errResp)
		if len(errResp.Errors) > 0 {
			return nil, fmt.Errorf("Asaas API Error [%d]: %s", resp.StatusCode, errResp.Errors[0].Description)
		}
		return nil, fmt.Errorf("Asaas API Error [%d]: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (a *AsaasAdapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	req := CustomerRequest{
		Name:    customer.Name,
		CpfCnpj: customer.Document,
		Email:   customer.Email,
	}

	respBody, err := a.doRequest(ctx, "POST", "/customers", req)
	if err != nil {
		return "", err
	}

	var resp CustomerResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (a *AsaasAdapter) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	req := TransactionRequest{
		Customer:    tx.CustomerID,
		BillingType: "PIX", // Default ou extraído de metadata futuramente
		Value:       tx.Amount,
		DueDate:     tx.DueDate.Format("2006-01-02"),
		Description: tx.Description,
	}

	respBody, err := a.doRequest(ctx, "POST", "/payments", req)
	if err != nil {
		return "", err
	}

	var resp TransactionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (a *AsaasAdapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	respBody, err := a.doRequest(ctx, "GET", fmt.Sprintf("/payments/%s", externalID), nil)
	if err != nil {
		return "", err
	}

	var resp TransactionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	// Aqui usaríamos o mesmo mapAsaasStatus do ACL, mas por simplicidade no Adapter retornamos bruto/pendente
	// O ideal é que o Worker/Service cuide da tradução via ACL.
	return entity.StatusPending, nil
}

func (a *AsaasAdapter) RefundTransaction(ctx context.Context, transactionID string) error {
	_, err := a.doRequest(ctx, "POST", fmt.Sprintf("/payments/%s/refund", transactionID), nil)
	return err
}
