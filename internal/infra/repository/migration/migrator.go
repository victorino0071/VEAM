package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// EnsureSchema gerencia o versionamento do banco de dados de forma transacional.
func EnsureSchema(db *sql.DB) error {
	slog.Info("[Migrator] Iniciando verificação de esquema...")

	// 1. Garante que a tabela de controle exista
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS asaas_framework_migrations (
		version int PRIMARY KEY,
		applied_at timestamp DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("falha ao criar tabela de controle de migração: %w", err)
	}

	// 2. Recupera a versão atual
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM asaas_framework_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("falha ao buscar versão atual: %w", err)
	}

	// 3. Carrega e ordena os arquivos de migração
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("falha ao ler diretório de migrações: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	// 4. Aplica deltas pendentes
	for _, fileName := range files {
		versionStr := strings.Split(fileName, "_")[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			slog.Warn("[Migrator] Arquivo ignoreado por formato inválido", "file", fileName)
			continue
		}

		if version > currentVersion {
			slog.Info("[Migrator] Aplicando migração", "version", version, "file", fileName)
			
			content, err := migrationFiles.ReadFile("migrations/" + fileName)
			if err != nil {
				return fmt.Errorf("falha ao ler arquivo %s: %w", fileName, err)
			}

			// Execução Atômica
			tx, err := db.Begin()
			if err != nil {
				return err
			}

			if _, err := tx.Exec(string(content)); err != nil {
				tx.Rollback()
				return fmt.Errorf("falha ao executar migração %s: %w", fileName, err)
			}

			if _, err := tx.Exec("INSERT INTO asaas_framework_migrations (version) VALUES ($1)", version); err != nil {
				tx.Rollback()
				return fmt.Errorf("falha ao atualizar tabela de versão: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			slog.Info("[Migrator] Migração aplicada com sucesso", "version", version)
		}
	}

	slog.Info("[Migrator] Banco de dados está atualizado", "version", currentVersion)
	return nil
}
