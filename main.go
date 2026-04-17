package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"asaas_framework/internal/app/handler"
	"asaas_framework/internal/app/service"
	"asaas_framework/internal/app/worker"
	"asaas_framework/internal/infra/gateway"
	"asaas_framework/internal/infra/repository"
	"asaas_framework/internal/infra/resilience"
	"asaas_framework/internal/infra/telemetry"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Driver Postgres
)

func main() {
	// Carrega o arquivo .env se ele existir
	_ = godotenv.Load()

	asaasKey := getEnv("ASAAS_API_KEY", "")
	asaasBase := getEnv("ASAAS_BASE_URL", "https://sandbox.asaas.com/api/v3")
	webhookToken := getEnv("WEBHOOK_ACCESS_TOKEN", "SuperSecretToken")
	httpPort := getEnv("HTTP_PORT", "8080")

	// Configurações do Banco extraídas do Environment
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "")
	dbUser := getEnv("DB_USER", "")
	dbPass := getEnv("DB_PASS", "")
	dbName := getEnv("DB_NAME", "")

	// 2. Inicializa Telemetria (OpenTelemetry + slog)
	shutdownTele, err := telemetry.InitTelemetry("asaas-framework")
	if err != nil {
		fmt.Printf("Falha ao iniciar telemetria: %v\n", err)
		os.Exit(1)
	}
	defer shutdownTele(context.Background())

	// 3. Conexão com Banco de Dados
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("Falha ao abrir conexão com banco", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("Banco de dados inacessível", "error", err)
		os.Exit(1)
	}

	// 4. Inicializa Infraestrutura (Adapters)
	repo := repository.NewPostgresRepository(db)
	gatewayAdapter := gateway.NewAsaasAdapter(asaasKey, asaasBase)

	// Circuit Breaker reativo para o Outbox
	cbConfig := resilience.Config{
		FailureThreshold: 0.5,
		ResetTimeout:     10 * time.Second,
		Alpha:            0.2,
		MinRequests:      5,
	}
	breaker := resilience.NewCircuitBreaker(cbConfig)

	// 5. Inicializa Aplicação (Services)
	paymentService := service.NewPaymentService(repo, gatewayAdapter)

	// 6. Inicializa Workers (Background Processing)
	inboxConsumer := worker.NewInboxConsumer(repo, paymentService)
	outboxRelay := worker.NewOutboxRelay(repo, gatewayAdapter, breaker)

	// Contexto de execução com cancelamento para os Workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicia Workers em Goroutines
	go inboxConsumer.Start(ctx)
	go outboxRelay.Start(ctx)

	// 7. Configura Servidor HTTP e Handlers
	webhookHandler := handler.NewWebhookHandler(repo, webhookToken)

	mux := http.NewServeMux()
	mux.Handle("/webhooks/asaas", webhookHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	server := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
	}

	// 8. Graceful Shutdown
	go func() {
		slog.Info("Servidor HTTP iniciado", "port", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Erro no servidor HTTP", "error", err)
		}
	}()

	// Captura interrupção do sistema (Ctrl+C, kill)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // Espera sinal
	slog.Info("Sinal de parada recebido. Encerrando graciosamente...")

	// Timeout de 10s para encerramento
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Para os Workers primeiro
	cancel() // Cancela o ctx global

	// Encerra o servidor HTTP
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Erro ao encerrar servidor HTTP", "error", err)
	}

	slog.Info("Aplicação encerrada com sucesso.")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
