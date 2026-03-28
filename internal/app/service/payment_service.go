package service

import (
	"context"
	"asaas_framework/internal/domain/entity"
	"asaas_framework/internal/domain/payment"
	"asaas_framework/internal/domain/port"
	"fmt"
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

// ProcessPayment orquestra a transição de estado de forma atômica (ACID).
func (s *PaymentService) ProcessPayment(ctx context.Context, transactionID string, nextStatus entity.PaymentStatus) error {
	// 1. Padrão ACID: Executamos tudo dentro de uma transação.
	// Nota: Locks de concorrência (SKIP LOCKED) são tratados no nível do repositório/worker.
	return s.repo.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// 2. Busca a transação atual
		tx, err := s.repo.GetTransactionByID(txCtx, transactionID)
		if err != nil {
			return fmt.Errorf("falha ao buscar transação: %w", err)
		}

		// 3. Inicializa a FSM
		fsm := payment.NewPaymentFSM(tx)

		// 4. Tenta a transição e gera o Evento Outbox
		event, err := fsm.TransitionTo(nextStatus)
		if err != nil {
			return fmt.Errorf("transição inválida: %w", err)
		}

		// 5. Persiste a mudança de estado da transação
		if err := s.repo.SaveTransaction(txCtx, tx); err != nil {
			return fmt.Errorf("falha ao salvar transação: %w", err)
		}

		// 6. Persiste o evento no Outbox (mesma transação)
		if event != nil {
			if err := s.repo.SaveOutboxEvent(txCtx, event); err != nil {
				return fmt.Errorf("falha ao salvar evento outbox: %w", err)
			}
		}

		return nil
	})
}
