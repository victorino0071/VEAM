# 🚀 Get Started with VEAM

Bem-vindo ao **VEAM (Versatile Engine for Automated Management of Payments)**. Este guia irá ajudá-lo a configurar e rodar o motor de pagamentos em poucos minutos.

## 📦 Instalação

Como o VEAM é um pacote Go, você pode adicioná-lo ao seu projeto usando:

```bash
go get github.com/Victor/VEAM
```

## 🏗️ Configuração Básica

O coração do VEAM é a `Engine`. Ela orquestra os repositórios, serviços e adaptadores de gateway.

### 1. Inicialize o Banco de Dados
O VEAM utiliza PostgreSQL para garantir transações ACID. Certifique-se de rodar as migrações usando o `veam-cli`:

```bash
./veam-cli migrate -dsn "postgres://user:pass@localhost:5432/veam_db?sslmode=disable"
```

### 2. Configure a Engine
No seu arquivo `main.go`, inicialize a engine e registre os provedores (Asaas, Mercado Pago, etc.):

```go
package main

import (
	"database/sql"
	"github.com/Victor/VEAM"
	"github.com/Victor/VEAM/adapters/mercadopago"
	_ "github.com/lib/pq"
)

func main() {
	db, _ := sql.Open("postgres", "your-dsn")
	
	// Inicializa a Engine
	engine := veam.NewEngine(db).
		WithTelemetry("meu-app-pagamentos").
		WithMaxRetries(5)

	// Registra o Mercado Pago
	mpAdapter, _ := mercadopago.NewAdapter("seu-access-token", "seu-webhook-secret")
	engine.RegisterProvider("mercadopago", mpAdapter)
}
```

## 🚦 Modos de Operação

O VEAM foi desenhado para ser escalável. Você pode rodá-lo em diferentes modos:

### Modo API (Receber Webhooks)
Configure seus endpoints HTTP para usar o handler do VEAM:

```go
mux := http.NewServeMux()
mux.Handle("/webhooks/mercadopago", engine.NewWebhookHandler("mercadopago"))
http.ListenAndServe(":8080", mux)
```

### Modo Worker (Processar Pagamentos)
Inicie os consumidores de background para processar a fila de Inbox e Outbox:

```go
// Roda em background
go engine.ConsumeInbox(context.Background())
go engine.RelayOutbox(context.Background())
```

## 🔗 Configurando Webhooks

Para que o VEAM receba notificações dos provedores:
1.  Use uma ferramenta como o **ngrok** para expor seu `localhost`.
2.  No painel do Mercado Pago/Asaas, configure a URL de notificação para:
    `https://seu-dominio.com/webhooks/mercadopago`

## 🛠️ Próximos Passos
- Consulte a [Documentação de Estrutura](../estrutura/overview.md) para detalhes arquiteturais.
- Veja os [Exemplos](../../examples/basic_server/main.go) para uma implementação completa.
