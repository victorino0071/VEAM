package service

import (
	"context"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/payment"
	"asaas_framework/internal/domain/port"
	"fmt"
	"log/slog"
)

type PaymentService struct {
	repo    port.Repository
	gateway port.GatewayAdapter
}

func NewPaymentService(repo port.Repository, gateway port.GatewayAdapter) *PaymentService {
	return &PaymentService{
		repo:    repo,
		gateway: gateway,
	}
}

// ProcessPaymentWithMetadata orquestra a transição de estado usando metadados (JSONB).
func (s *PaymentService) ProcessPaymentWithMetadata(ctx context.Context, transactionID string, metadata map[string]string, nextStatus entity.PaymentStatus) error {
	slog.InfoContext(ctx, "Iniciando ProcessPayment (Mastery)", "transaction_id", transactionID)
	
	return s.repo.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		tx, err := s.repo.GetTransactionByID(txCtx, transactionID)
		if err != nil {
			return fmt.Errorf("falha ao buscar transação: %w", err)
		}

		fsm := payment.NewPaymentFSM(tx)
		// Propaga os metadados (Trace Context) para o evento gerado
		fsm.SetMetadata(metadata) 

		event, err := fsm.TransitionTo(nextStatus)
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
		}

		slog.InfoContext(txCtx, "Transição de estado processada", "new_status", tx.Status)
		return nil
	})
}
