package asaas

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/victorino0071/VEAM/domain/entity"
	"github.com/victorino0071/VEAM/domain/port"
	"github.com/victorino0071/VEAM/internal/core/acl"
	"github.com/cespare/xxhash/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Adapter struct {
	apiKey     string
	keyring    atomic.Value // holds *entity.GracefulKeyring
	baseUrl    string
	httpClient *http.Client
}

func NewAdapter(apiKey string, webhookSecret string, baseUrl string) port.GatewayAdapter {
	a := &Adapter{
		apiKey:     apiKey,
		baseUrl:    baseUrl,
		httpClient: &http.Client{},
	}
	a.keyring.Store(entity.NewKeyring(webhookSecret))
	return a
}

func (a *Adapter) doRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", a.baseUrl, path)
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	tracer := otel.Tracer("asaas-adapter")
	ctx, span := tracer.Start(ctx, fmt.Sprintf("Asaas.%s", method), trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("access_token", a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	span.SetAttributes(attribute.String("http.url", url), attribute.String("http.method", method))
	slog.InfoContext(ctx, "[Gateway] Efetuando requisição externa", "method", method, "url", url)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "[Gateway] Falha crítica de transporte HTTP", "error", err, "url", url)
		span.SetAttributes(attribute.String("error.message", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.String("http.status_code", fmt.Sprintf("%d", resp.StatusCode)))

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		json.Unmarshal(respBody, &errResp)
		if len(errResp.Errors) > 0 {
			slog.WarnContext(ctx, "[Gateway] API do Provedor retornou erro de negócio", "status", resp.StatusCode, "error", errResp.Errors[0].Description)
			span.SetAttributes(attribute.String("error.provider", errResp.Errors[0].Description))
			return nil, fmt.Errorf("Provider API Error [%d]: %s", resp.StatusCode, errResp.Errors[0].Description)
		}
		slog.ErrorContext(ctx, "[Gateway] Erro inesperado na API do Provedor", "status", resp.StatusCode, "body", string(respBody))
		span.SetAttributes(attribute.String("error.provider", string(respBody)))
		
		errBase := fmt.Errorf("Provider API Error [%d]: %s", resp.StatusCode, string(respBody))
		// Erros 4xx são regras de negócio/terminais. Diferente de 5xx (transiente/retry)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 && resp.StatusCode != 408 {
			return nil, fmt.Errorf("%w: %v", entity.ErrTerminalGatewayRejection, errBase)
		}
		return nil, errBase
	}

	slog.InfoContext(ctx, "[Gateway] Resposta externa recebida com sucesso", "status", resp.StatusCode)
	// Try capturing IDs if body is JSON with "id"
	var maybeID struct {
		ID string `json:"id"`
	}
	if _err := json.Unmarshal(respBody, &maybeID); _err == nil && maybeID.ID != "" {
		span.SetAttributes(attribute.String("asaas.payment_id", maybeID.ID))
	}
	
	return respBody, nil
}

func (a *Adapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	req := CustomerRequest{
		Name:    customer.Name,
		CpfCnpj: customer.Document,
		Email:   customer.Email,
	}

	respBody, err := a.doRequest(ctx, "POST", "/customers", req, nil)
	if err != nil {
		return "", err
	}

	var resp CustomerResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (a *Adapter) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	req := TransactionRequest{
		Customer:    tx.CustomerID,
		BillingType: "PIX", // Default ou extraído de metadata futuramente
		Value:       tx.Amount,
		DueDate:     tx.DueDate.Format("2006-01-02"),
		Description: tx.Description,
	}

	respBody, err := a.doRequest(ctx, "POST", "/payments", req, nil)
	if err != nil {
		return "", err
	}

	var resp TransactionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (a *Adapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	respBody, err := a.doRequest(ctx, "GET", fmt.Sprintf("/payments/%s", externalID), nil, nil)
	if err != nil {
		return "", err
	}

	var resp TransactionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	// Aqui usaríamos o mesmo mapProviderStatus do ACL, mas por simplicidade no Adapter retornamos bruto/pendente
	// O ideal é que o Worker/Service cuide da tradução via ACL.
	return entity.StatusPending, nil
}

func (a *Adapter) RefundTransaction(ctx context.Context, transactionID string, idempotencyKey string) error {
	headers := make(map[string]string)
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	_, err := a.doRequest(ctx, "POST", fmt.Sprintf("/payments/%s/refund", transactionID), nil, headers)
	return err
}

func (a *Adapter) RotateWebhookSecret(newSecret string, gracePeriod time.Duration) {
	oldKeyring := a.keyring.Load().(*entity.GracefulKeyring)
	a.keyring.Store(oldKeyring.Rotate(newSecret, gracePeriod))
}

func (a *Adapter) ValidateWebhook(r *http.Request) (bool, error) {
	// O Asaas envia o token no header 'asaas-access-token'
	token := r.Header.Get("asaas-access-token")
	if token == "" {
		token = r.Header.Get("X-Provider-Token") // Fallback para testes manuais
	}

	kr := a.keyring.Load().(*entity.GracefulKeyring)
	tokenBytes := []byte(token)

	// Tenta a Chave Primária
	if subtle.ConstantTimeCompare(tokenBytes, []byte(kr.PrimarySecret)) == 1 {
		return true, nil
	}

	// Tenta a Chave Secundária (Drift Temporal)
	if kr.IsSecondaryValid() {
		if subtle.ConstantTimeCompare(tokenBytes, []byte(kr.SecondarySecret)) == 1 {
			return true, nil
		}
	}

	return false, errors.New("invalid webhook access token (all keys exhausted)")
}

func (a *Adapter) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var evt struct {
		ID    string `json:"id"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		return nil, err
	}

	return &port.WebhookResponse{
		ExternalID: evt.ID,
		EventType:  evt.Event,
		Payload:    body,
	}, nil
}

func (a *Adapter) TranslatePayload(ctx context.Context, payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	var dto acl.WebhookDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return nil, "", fmt.Errorf("falha ao deserializar payload asaas: %w", err)
	}

	tx, err := dto.Payment.ToDomain("asaas")
	if err != nil {
		return nil, "", err
	}

	return tx, acl.MapAsaasStatus(dto.Payment.Status), nil
}

func (a *Adapter) Fingerprint(payload []byte) (string, error) {
	var dto acl.WebhookDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return "", err
	}

	// Composição estrita para evitar Identidade de Entrega vs Negócio
	// Ignoramos o ID do evento global (delivery id) e usamos apenas o core financeiro
	core := fmt.Sprintf("%s|%s|%.2f|%s", 
		dto.Payment.ID, 
		dto.Payment.Status, 
		dto.Payment.Value, 
		dto.Payment.DueDate,
	)

	h := xxhash.New()
	h.Write([]byte(core))
	return fmt.Sprintf("%016x", h.Sum64()), nil
}
