package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/Victor/VEAM/domain/entity"

	veam "github.com/Victor/VEAM"
	"github.com/Victor/VEAM/adapters/asaas"
	"github.com/Victor/VEAM/adapters/mercadopago"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Driver Postgres
)

func main() {
	_ = godotenv.Load()

	asaasKey := getEnv("ASAAS_API_KEY", "")
	asaasSecret := getEnv("ASAAS_WEBHOOK_ACCESS_TOKEN", "")
	asaasBase := getEnv("ASAAS_BASE_URL", "https://sandbox.asaas.com/api/v3")
	
	mpToken := getEnv("MP_ACCESS_TOKEN", "")
	mpSecret := getEnv("MP_WEBHOOK_SECRET", "secret")

	httpPort := getEnv("HTTP_PORT", "8080")

	// 1. Conexão com Banco de Dados
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", ""), getEnv("DB_PORT", ""), getEnv("DB_USER", ""),
		getEnv("DB_PASS", ""), getEnv("DB_NAME", ""))

	db, err := sql.Open("postgres", dsn)
	if err != nil || db.Ping() != nil {
		slog.Error("Falha crítica no banco de dados", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mpRealAdapter, err := mercadopago.NewAdapter(mpToken, mpSecret)
	if err != nil {
		slog.Error("Falha ao configurar Mercado Pago", "error", err)
		os.Exit(1)
	}

	engine := veam.NewEngine(db).
		WithTelemetry("VEAM").
		RegisterProvider("asaas", asaas.NewAdapter(asaasKey, asaasSecret, asaasBase)).
		RegisterProvider("mercadopago", mpRealAdapter)

	// 3. Inicia Background Workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	// 4. Configura Rotas HTTP
	mux := http.NewServeMux()
	mux.Handle("/webhooks/asaas", engine.NewWebhookHandler("asaas"))
	mux.Handle("/webhooks/mercadopago", engine.NewWebhookHandler("mercadopago"))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	// 4.1 Endpoint de Teste: Criar Pagamento Mercado Pago
	mux.HandleFunc("/test/mp/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Amount      float64 `json:"amount"`
			Email       string  `json:"email"`
			Description string  `json:"description"`
			Token       string  `json:"token"`
			CPF         string  `json:"cpf"`
			Installments int    `json:"installments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		// 1. Criar Transação de Domínio (PENDING)
		txID := uuid.New().String()
		productDueDate := time.Now().Add(24 * time.Hour)
		tx := entity.NewTransaction(txID, "customer-test-1", "mercadopago", input.Amount, input.Description, productDueDate)
		
		// Metadados para o Mercado Pago
		tx.SetMetadata("payer_email", input.Email)
		if input.Token != "" {
			tx.SetMetadata("card_token", input.Token)
			tx.SetMetadata("payment_method_id", "master")
			if input.Installments > 0 {
				tx.SetMetadata("installments", fmt.Sprintf("%d", input.Installments))
			} else {
				tx.SetMetadata("installments", "1")
			}
			if input.CPF != "" {
				tx.SetMetadata("payer_identification_number", input.CPF)
			}
		} else {
			tx.SetMetadata("payment_method_id", "pix")
		}

		// 2. Criar no Provedor através do Adapter
		adapter, _ := engine.Registry.Get("mercadopago")
		externalID, err := adapter.CreateTransaction(context.Background(), tx)
		if err != nil {
			slog.Error("Erro ao criar no MP", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Atualizar com ID Externo e Persistir localmente
		// Isso é vital para que o Webhook consiga correlacionar depois
		tx.SetMetadata("external_id", externalID)
		if err := engine.Repo.SaveTransaction(context.Background(), tx); err != nil {
			slog.Error("Erro ao salvar localmente", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Pagamento de teste criado com sucesso", "internal_id", txID, "external_id", externalID)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "created",
			"internal_id": txID,
			"external_id": externalID,
			"message":     "Aguarde o webhook no ngrok!",
		})
	})

	// 4.1 Endpoint Administrativo de Rotação (Referência de Operações)
	mux.HandleFunc("/internal/system/rotate-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		// Segurança básica para demonstração (Referência: mTLS ou RBAC em produção)
		if r.Header.Get("X-Admin-Token") != getEnv("ADMIN_TOKEN", "secret-admin-token") {
			http.Error(w, "Não autorizado", http.StatusUnauthorized)
			return
		}

		var req struct {
			Provider     string `json:"provider"`
			NewSecret    string `json:"new_secret"`
			GraceSeconds int    `json:"grace_seconds"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Payload inválido", http.StatusBadRequest)
			return
		}

		grace := time.Duration(req.GraceSeconds) * time.Second
		if err := engine.RotateGatewaySecret(req.Provider, req.NewSecret, grace); err != nil {
			slog.Error("Falha ao rotacionar chaves", "provider", req.Provider, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Rotação de chaves concluída com sucesso via API", "provider", req.Provider, "grace_period", grace)
		w.WriteHeader(http.StatusAccepted)
	})

	server := &http.Server{Addr: ":" + httpPort, Handler: mux}

	// 5. Graceful Shutdown
	go func() {
		slog.Info("Servidor HTTP iniciado", "port", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Erro no servidor HTTP", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("Encerrando graciosamente...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	cancel()
	server.Shutdown(shutdownCtx)
	slog.Info("Aplicação encerrada com sucesso.")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		// Remove aspas simples ou duplas que podem ter sido colocadas para evitar expansão de $
		return strings.Trim(strings.Trim(value, "'"), "\"")
	}
	return fallback
}
