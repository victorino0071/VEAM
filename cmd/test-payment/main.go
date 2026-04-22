package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/victorino0071/VEAM/adapters/mercadopago"
	"github.com/victorino0071/VEAM/domain/entity"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Carrega as variáveis do .env
	_ = godotenv.Load()
	token := os.Getenv("MP_ACCESS_TOKEN")
	secret := os.Getenv("MP_WEBHOOK_SECRET")

	if token == "" {
		fmt.Println("❌ ERRO: MP_ACCESS_TOKEN não definido no arquivo .env")
		fmt.Println("Pegue o token do seu Usuário Vendedor de Teste no painel do Mercado Pago.")
		os.Exit(1)
	}

	// 2. Inicializa o Adaptador
	adapter, err := mercadopago.NewAdapter(token, secret)
	if err != nil {
		log.Fatalf("❌ Falha ao iniciar adaptador: %v", err)
	}

	// 3. Cria uma transação de domínio
	// Em um sistema real, isso viria da sua App/Service
	txID := fmt.Sprintf("order-%d", time.Now().Unix())
	tx := entity.NewTransaction(
		txID,
		"user-customer-999",
		"mercadopago",
		42.50, // Valor do teste
		"Pagamento de Teste - Motor de Checkout",
		time.Now().AddDate(0, 0, 7), // Due em 7 dias
	)
	
	// Metadados necessários para o adaptador do Mercado Pago
	// Use o e-mail do seu COMPRADOR de teste criado no painel
	tx.SetMetadata("payer_email", "test_user_3351841308@testuser.com") 
	tx.SetMetadata("payment_method_id", "pix")

	fmt.Printf("🚀 Iniciando disparo para o Mercado Pago (ID Interno: %s)...\n", txID)

	// 4. Executa a chamada real para a API do Mercado Pago
	externalID, err := adapter.CreateTransaction(context.Background(), tx)
	if err != nil {
		fmt.Printf("❌ Falha crítica na API do Mercado Pago: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n====================================================")
	fmt.Println("✅ PAGAMENTO CRIADO COM SUCESSO NO MERCADO PAGO!")
	fmt.Printf("ID Externo (MP): %s\n", externalID)
	fmt.Printf("ID Interno (FSM): %s\n", txID)
	fmt.Println("====================================================")
	fmt.Println("\nCOMO TESTAR O WEBHOOK AGORA:")
	fmt.Println("1. Certifique-se que o 'ngrok' e o 'basic_server' estão rodando.")
	fmt.Println("2. Copie o ID Externo acima.")
	fmt.Println("3. Vá ao Simulador: https://www.mercadopago.com.br/developers/pt/gui/test-integration/simulator")
	fmt.Println("4. Selecione 'Pix', cole o ID e clique em 'Simular Pagamento'.")
	fmt.Println("5. Olhe o terminal do seu 'basic_server' para ver a transição de estado!")
}
