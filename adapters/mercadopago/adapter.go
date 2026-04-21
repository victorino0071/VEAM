package mercadopago

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/internal/core/acl"
	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"github.com/mercadopago/sdk-go/pkg/refund"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type idempTransport struct {
	rt  http.RoundTripper
	key string
}

func (t *idempTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.key != "" {
		req.Header.Set("X-Idempotency-Key", t.key)
	}
	return t.rt.RoundTrip(req)
}

type Adapter struct {
	client  payment.Client
	cfg     *config.Config
	keyring atomic.Value // holds *entity.GracefulKeyring
}

func NewAdapter(accessToken string, webhookSecret string) (port.GatewayAdapter, error) {
	c, err := config.New(accessToken)
	if err != nil {
		return nil, fmt.Errorf("mp_config_error: %w", err)
	}
	a := &Adapter{
		client: payment.NewClient(c),
		cfg:    c,
	}
	a.keyring.Store(entity.NewKeyring(webhookSecret))
	return a, nil
}

func (a *Adapter) CreateCustomer(ctx context.Context, customer *entity.Customer) (string, error) {
	// O Mercado Pago trata clientes separadamente através da API de Customers.
	// Por simplicidade na demonstração (e como muitos pagamentos Pix/Cartão aceitam Payer direto),
	// retornaremos um mock ou você pode usar o client de customer do SDK.
	// Para uso real, deve-se usar github.com/mercadopago/sdk-go/pkg/customer
	return "mock-customer-id", nil
}

func (a *Adapter) CreateTransaction(ctx context.Context, tx *entity.Transaction) (string, error) {
	paymentMethod := tx.GetMetadata("payment_method_id", "pix")
	payerEmail := tx.GetMetadata("payer_email", "")

	if payerEmail == "" {
		return "", errors.New("mercadopago_adapter: payer_email is strictly required")
	}

	request := payment.Request{
		TransactionAmount: tx.Amount,
		Description:       tx.Description,
		PaymentMethodID:   paymentMethod,
		Payer: &payment.PayerRequest{
			Email: payerEmail,
			Identification: &payment.IdentificationRequest{
				Type:   tx.GetMetadata("payer_identification_type", "CPF"),
				Number: tx.GetMetadata("payer_identification_number", "19119119100"),
			},
		},
		Token:        tx.GetMetadata("card_token", ""),
		Installments: 1, // Por padrão em teste
		ExternalReference: tx.ID,
	}

	// Tenta converter parcelas se fornecido
	if inst := tx.GetMetadata("installments", ""); inst != "" {
		if val, err := strconv.Atoi(inst); err == nil {
			request.Installments = val
		}
	}

	tracer := otel.Tracer("mercadopago-adapter")
	ctx, span := tracer.Start(ctx, "MercadoPago.CreateTransaction", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	result, err := a.client.Create(ctx, request)
	if err != nil {
		slog.ErrorContext(ctx, "[MercadoPago] Falha ao criar transação", "error", err)
		span.SetAttributes(attribute.String("error.message", err.Error()))
		return "", err
	}

	span.SetAttributes(
		attribute.String("mercadopago.payment_id", fmt.Sprintf("%d", result.ID)),
		attribute.String("http.status_code", "201"),
	)

	// O ID no Mercado Pago é numérico, então convertemos para string
	return fmt.Sprintf("%d", result.ID), nil
}

func (a *Adapter) GetTransactionState(ctx context.Context, externalID string) (entity.PaymentStatus, error) {
	idInt, err := strconv.Atoi(externalID)
	if err != nil {
		return "", fmt.Errorf("ID inválido para o Mercado Pago: %v", err)
	}

	result, err := a.client.Get(ctx, idInt)
	if err != nil {
		return "", err
	}

	return acl.MapMercadoPagoStatus(result.Status), nil
}

func (a *Adapter) RefundTransaction(ctx context.Context, transactionID string, idempotencyKey string) error {
	idInt, err := strconv.Atoi(transactionID)
	if err != nil {
		return fmt.Errorf("ID inválido para o Mercado Pago: %v", err)
	}

	tracer := otel.Tracer("mercadopago-adapter")
	ctx, span := tracer.Start(ctx, "MercadoPago.RefundTransaction", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// Injeta Header customizado no Request de Estorno via Transport HTTP
	reqOptions := []config.Option{}
	if idempotencyKey != "" {
		client := &http.Client{
			Transport: &idempTransport{
				rt:  http.DefaultTransport,
				key: idempotencyKey,
			},
		}
		reqOptions = append(reqOptions, config.WithHTTPClient(client))
	}

	// Criamos um client avulso com este custom config para aplicar o cabeçalho globalmente
	tmpCfg, _ := config.New(a.cfg.AccessToken, reqOptions...)
	refundClient := refund.NewClient(tmpCfg)

	// O Mercado Pago para refund simples cria um "refund" atrelado ao paymentId
	result, err := refundClient.Create(ctx, idInt)
	if err != nil {
		slog.ErrorContext(ctx, "[MercadoPago] Falha ao realizar estorno", "error", err)
		span.SetAttributes(attribute.String("error.message", err.Error()))

		// Analisa erro do MP: erros 400 são terminais.
		if strings.Contains(err.Error(), "status: 400") || strings.Contains(err.Error(), "status: 404") || strings.Contains(err.Error(), "status: 401") || strings.Contains(err.Error(), "bad_request") {
			return fmt.Errorf("%w: %v", entity.ErrTerminalGatewayRejection, err)
		}
		return err
	}

	span.SetAttributes(
		attribute.String("mercadopago.refund_id", fmt.Sprintf("%d", result.ID)),
		attribute.String("mercadopago.payment_id", transactionID),
		attribute.String("http.status_code", "201"),
	)

	return nil
}

func (a *Adapter) RotateWebhookSecret(newSecret string, gracePeriod time.Duration) {
	oldKeyring := a.keyring.Load().(*entity.GracefulKeyring)
	a.keyring.Store(oldKeyring.Rotate(newSecret, gracePeriod))
}

func (a *Adapter) ValidateWebhook(r *http.Request) (bool, error) {
	signatureHeader := r.Header.Get("x-signature")
	requestID := r.Header.Get("x-request-id")
	
	if signatureHeader == "" || requestID == "" {
		return false, errors.New("missing x-signature or x-request-id")
	}

	parts := strings.Split(signatureHeader, ",")
	var ts, v1 string
	for _, part := range parts {
		if strings.HasPrefix(part, "ts=") {
			ts = strings.TrimPrefix(part, "ts=")
		} else if strings.HasPrefix(part, "v1=") {
			v1 = strings.TrimPrefix(part, "v1=")
		}
	}

	if ts == "" || v1 == "" {
		return false, errors.New("malformed x-signature header")
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return false, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var evt struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bodyBytes, &evt)

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", evt.Data.ID, requestID, ts)
	kr := a.keyring.Load().(*entity.GracefulKeyring)

	// 1. Tenta Validar com Chave Primária
	if a.verifyHMAC(manifest, v1, kr.PrimarySecret) {
		return true, nil
	}

	// 2. Tenta Validar com Chave Secundária (Grace Period)
	if kr.IsSecondaryValid() {
		if a.verifyHMAC(manifest, v1, kr.SecondarySecret) {
			return true, nil
		}
	}

	return false, errors.New("invalid hmac signature (all keys exhausted)")
}

func (a *Adapter) verifyHMAC(manifest, signature, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(manifest))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (a *Adapter) TranslateWebhook(r *http.Request) (*port.WebhookResponse, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var evt struct {
		Action string          `json:"action"`
		Type   string          `json:"type"`
		Data   json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &evt); err != nil {
		return nil, err
	}

	eventType := evt.Action
	if eventType == "" {
		eventType = evt.Type
	}

	// Extract ID flexibly since MP might send string or int based on topic
	var dataObj map[string]interface{}
	if err := json.Unmarshal(evt.Data, &dataObj); err != nil {
		return nil, err
	}

	var externalID string
	if idVal, ok := dataObj["id"]; ok {
		switch v := idVal.(type) {
		case string:
			externalID = v
		case float64:
			externalID = fmt.Sprintf("%.0f", v)
		}
	}

	return &port.WebhookResponse{
		ExternalID: externalID,
		EventType:  eventType,
		Payload:    body,
	}, nil
}

func (a *Adapter) TranslatePayload(ctx context.Context, payload []byte) (*entity.Transaction, entity.PaymentStatus, error) {
	// MP webhook usually just signals an event ID. We must query the API.
	var dto acl.MercadoPagoWebhookDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return nil, "", fmt.Errorf("falha ao deserializar notificação do mercadopago: %w", err)
	}

	var paymentID string
	if idVal, ok := dto.Data["id"]; ok {
		switch v := idVal.(type) {
		case string:
			paymentID = v
		case float64:
			paymentID = fmt.Sprintf("%.0f", v)
		}
	}

	idInt, err := strconv.ParseInt(paymentID, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("falha ao converter ID da notificação: %w", err)
	}

	// Fazendo Fetch da API ao invés de aceitar cegamente o Webhook
	result, err := a.client.Get(ctx, int(idInt))
	if err != nil {
		return nil, "", fmt.Errorf("falha ao buscar transação real no mercado pago: %w", err)
	}

	tx, err := acl.ToDomainFromMPPayment(result, "mercadopago")
	if err != nil {
		return nil, "", err
	}

	// Sobrescreve o ID com a nossa referência externa se ela existir
	if result.ExternalReference != "" {
		tx.ID = result.ExternalReference
	}

	return tx, acl.MapMercadoPagoStatus(result.Status), nil
}
