package service

import (
	"context"
	"github.com/Victor/payment-engine/domain/entity"
	"github.com/Victor/payment-engine/domain/port"
	"github.com/Victor/payment-engine/domain/registry"
	"fmt"
	"log/slog"
)

type PaymentService struct {
	repo     port.Repository
	registry *registry.ProviderRegistry
}

func NewPaymentService(repo port.Repository, reg *registry.ProviderRegistry) *PaymentService {
	return &PaymentService{
		repo:     repo,
		registry: reg,
	}
}

// ProcessPaymentWithMetadata orquestra a transição de estado usando metadados (JSONB).
func (s *PaymentService) ProcessPaymentWithMetadata(ctx context.Context, incomingTx *entity.Transaction, metadata map[string]string, nextStatus entity.PaymentStatus) error {
	slog.InfoContext(ctx, "Iniciando ProcessPayment (Mastery)", "transaction_id", incomingTx.ID)
	
	return s.repo.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		tx, err := s.repo.GetTransactionByID(txCtx, incomingTx.ID)
		if err != nil {
			return fmt.Errorf("falha ao buscar transação: %w", err)
		}

		// Se não existe no banco (primeiro webhook, ex: PAYMENT_CREATED)
		if tx == nil {
			tx = incomingTx
		}

		event, err := tx.TransitionTo(nextStatus, metadata)
		if err != nil {
			return fmt.Errorf("transição inválida: %w", err)
		}

		if err := s.repo.SaveTransaction(txCtx, tx); err != nil {
			return fmt.Errorf("falha ao salvar transação: %w", err)
		}

		if event != nil {
			if err := s.repo.SaveOutboxEvent(txCtx, event); err != nil {
				return fmt.Errorf("falha ao salvar evento outbox: %w", err)
			}
			slog.InfoContext(txCtx, "[Service] Evento de Outbox gerado com sucesso", "event_type", event.EventType)
		}

		slog.InfoContext(txCtx, "[Service] Transição de estado persistida atómicamente", "new_status", tx.Status())
		return nil
	})
}
