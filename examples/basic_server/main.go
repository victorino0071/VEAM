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

	paymentengine "github.com/Victor/payment-engine"
	"github.com/Victor/payment-engine/adapters/asaas"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Driver Postgres
)

func main() {
	_ = godotenv.Load()

	providerKey := getEnv("GATEWAY_API_KEY", "")
	providerBase := getEnv("GATEWAY_BASE_URL", "https://sandbox.asaas.com/api/v3")
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

	// 2. Setup do Motor via Builder Pattern (Facade)
	engine := paymentengine.NewEngine(db).
		WithTelemetry("payment-engine").
		RegisterProvider("asaas", asaas.NewAdapter(providerKey, providerBase))

	// 3. Inicia Background Workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	// 4. Configura Rotas HTTP
	mux := http.NewServeMux()
	mux.Handle("/webhooks/provider", engine.NewWebhookHandler("asaas"))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
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
		return value
	}
	return fallback
}
