package worker

import (
	"context"
	"asaas_framework/internal/app/acl"
	"asaas_framework/internal/app/service"
	"asaas_framework/internal/domain/port"
	"fmt"
	"time"
	"encoding/json"
)

type InboxConsumer struct {
	repo    port.Repository
	service *service.PaymentService
	baseT   time.Duration // T_base
	maxT    time.Duration // T_max
	k       int           // Tentativas vazias consecutivas
	quit    chan struct{}
}

func NewInboxConsumer(repo port.Repository, svc *service.PaymentService) *InboxConsumer {
	return &InboxConsumer{
		repo:    repo,
		service: svc,
		baseT:   500 * time.Millisecond,
		maxT:    30 * time.Second,
		k:       0,
		quit:    make(chan struct{}),
	}
}

func (c *InboxConsumer) Start(ctx context.Context) {
	for {
		// 1. Processa o lote
		processed := c.consume(ctx)

		// 2. Calcula T_poll (Backoff Exponencial)
		if processed == 0 {
			c.k++
		} else {
			c.k = 0
		}

		backoff := time.Duration(int64(c.baseT) << c.k)
		if backoff > c.maxT || backoff <= 0 { // Prevenção de overflow
			backoff = c.maxT
		}

		select {
		case <-time.After(backoff):
			continue
		case <-c.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *InboxConsumer) consume(ctx context.Context) int {
	// 2. Fetch de Inbox com FOR UPDATE SKIP LOCKED
	events, err := c.repo.FetchNextPendingInbox(ctx, 5)
	if err != nil || len(events) == 0 {
		return 0
	}

	for _, event := range events {
		fmt.Printf("[InboxConsumer] Consumindo evento externo: %s (ID: %s)\n", event.ExternalID, event.ID)
		
		// 3. ACL Deslocada: Unmarshal e Tradução no background
		var dto acl.AsaasPaymentDTO
		if err := json.Unmarshal(event.Payload, &dto); err != nil {
			fmt.Printf("[InboxConsumer] FAILED_TRANSLATION: %v\n", err)
			event.Status = "FAILED_TRANSLATION"
			continue
		}

		// 4. Converte para Domínio
		tx, err := dto.ToDomain()
		if err != nil {
			fmt.Printf("[InboxConsumer] FAILED_DOMAIN_MAPPING: %v\n", err)
			event.Status = "FAILED_TRANSLATION"
			continue
		}

		// 5. Orquestra o processamento no Domínio (FSM + ACID)
		err = c.service.ProcessPayment(ctx, tx.ID, tx.Status)
		
		if err != nil {
			event.RetryCount++
			continue
		}
		
		now := time.Now()
		event.Status = "PROCESSED"
		event.ProcessedAt = &now
	}

	return len(events)
}

func (c *InboxConsumer) Stop() {
	close(c.quit)
}
