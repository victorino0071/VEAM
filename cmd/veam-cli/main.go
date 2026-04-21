package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/Victor/VEAM/internal/core/repository/migration"
)

func main() {
	// Carrega o .env explicitamente
	_ = godotenv.Load()
	
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	
	// Prioriza DATABASE_URL do ambiente se o flag não for passado
	defaultDSN := os.Getenv("DATABASE_URL")
	dsn := migrateCmd.String("dsn", defaultDSN, "Postgres DSN (ou use DATABASE_URL env)")

	if len(os.Args) < 2 {
		fmt.Println("Uso: VEAM-cli <comando> [opções]")
		fmt.Println("Comandos disponíveis: migrate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		migrateCmd.Parse(os.Args[2:])
		if *dsn == "" {
			log.Fatal("Erro: DSN do banco de dados não fornecido. Use -dsn ou DATABASE_URL.")
		}

		db, err := sql.Open("postgres", *dsn)
		if err != nil {
			log.Fatalf("Falha ao abrir conexão: %v", err)
		}
		defer db.Close()

		fmt.Println("🚀 Iniciando migração industrial...")
		if err := migration.EnsureSchema(db); err != nil {
			log.Fatalf("❌ Erro na migração: %v", err)
		}
		fmt.Println("✅ Sincronização de esquema concluída com sucesso.")

	default:
		fmt.Printf("Comando desconhecido: %s\n", os.Args[1])
		os.Exit(1)
	}
}
