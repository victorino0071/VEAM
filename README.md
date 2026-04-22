# ⚡ VEAM: Virtual Engine for Atomic Management

[![Go Reference](https://pkg.go.dev/badge/github.com/victorino0071/VEAM.svg)](https://pkg.go.dev/github.com/victorino0071/VEAM)
[![Go CI](https://github.com/victorino0071/VEAM/actions/workflows/go.yml/badge.svg)](https://github.com/victorino0071/VEAM/actions/workflows/go.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**VEAM** é um motor de pagamentos industrial projetado para sistemas que não podem se dar ao luxo de perder um único centavo. Construído sobre os princípios da **Arquitetura Hexagonal** e do **Transactional Outbox**, o VEAM abstrai a complexidade de gateways de pagamento enquanto garante integridade atômica e resiliência extrema.

---

## 📑 Sumário

- [Início Rápido](#início-rápido)
  - [Instalação](#instalação)
  - [Configuração do Banco](#configuração-do-banco)
  - [Primeiros Passos](#primeiros-passos)
- [Conceitos Fundamentais](#conceitos-fundamentais)
  - [Arquitetura Hexagonal](#arquitetura-hexagonal)
  - [O Padrão Inbox](#o-padrão-inbox)
  - [O Padrão Outbox](#o-padrão-outbox)
- [Resiliência Industrial](#resiliência-industrial)
  - [Exponential Backoff](#exponential-backoff)
  - [Circuit Breaker](#circuit-breaker)
  - [Dead Letter Queue (DLQ)](#dead-letter-queue-dlq)
- [Observabilidade & Tracing](#observabilidade--tracing)
- [Extensibilidade](#extensibilidade)
  - [Criando um Adaptador](#criando-um-adaptador)
- [Manutenção & CLI](#manutenção--cli)

---

## Início Rápido

### Instalação

O VEAM requer Go 1.26+ e uma instância do PostgreSQL 13+.

```bash
go get github.com/victorino0071/VEAM
```

### Configuração do Banco

O VEAM utiliza recursos específicos do PostgreSQL (`SKIP LOCKED`, `JSONB`, `PARTITIONS`). Para preparar seu banco:

```bash
# Compile a ferramenta de migração
go build -o veam-cli ./cmd/veam-cli

# Execute a migração inicial
./veam-cli migrate -dsn "postgres://user:pass@localhost:5432/db?sslmode=disable"
```

### Primeiros Passos

O coração do VEAM é o `Engine`. Ele coordena o registro de provedores e o ciclo de vida dos workers.

```go
package main

import (
    "context"
    "database/sql"
    "github.com/victorino0071/VEAM"
    "github.com/victorino0071/VEAM/adapters/asaas"
    _ "github.com/lib/pq"
)

func main() {
    db, _ := sql.Open("postgres", "YOUR_DSN")

    // 1. Instancie o motor
    engine := veam.NewEngine(db).
        WithMaxRetries(5).
        WithTelemetry("payment-service")

    // 2. Registre seus adaptadores
    engine.RegisterProvider("asaas_prod", asaas.NewAsaasAdapter("KEY", "SECRET"))

    // 3. Suba o motor (Workers assíncronos)
    ctx := context.Background()
    engine.Start(ctx)
    
    // 4. Exponha o handler de Webhook
    // Este handler salvará automaticamente na Inbox com Idempotência.
    http.Handle("/webhooks/asaas", engine.NewWebhookHandler("asaas_prod"))
    http.ListenAndServe(":8080", nil)
}
```

---

## Conceitos Fundamentais

### Arquitetura Hexagonal

O VEAM é dividido em **Core (Domínio)**, **Ports (Interfaces)** e **Adapters (Implementações)**. Isso significa que você pode trocar o gateway de pagamento ou até o banco de dados sem tocar na lógica de negócio central.

### O Padrão Inbox

Ao receber um webhook, o VEAM **não o processa imediatamente**. O evento é salvo na tabela `inbox` com status `PENDING`. 
- **Vantagem**: Se o processamento falhar ou o sistema reiniciar, o evento não é perdido.
- **Idempotência**: O VEAM verifica o `external_id` do evento para evitar processar o mesmo webhook duas vezes.

### O Padrão Outbox

Toda mudança de estado no VEAM (ex: pagamento confirmado) gera um evento de saída na tabela `outbox`. Esse evento é salvo **na mesma transação atômica** que a atualização do status da transação.
- **Garantia**: Nunca haverá um status atualizado no seu banco sem que uma notificação seja gerada para o mundo exterior (e vice-versa).

---

## Resiliência Industrial

### Exponential Backoff

Os workers do VEAM (`InboxConsumer` e `OutboxRelay`) utilizam um algoritmo de backoff inteligente. Se não houver trabalho, o worker entra em repouso progressivo:
- Inicia em **500ms**.
- Escala exponencialmente até o limite de **30 segundos**.
- Retorna imediatamente a **0ms** assim que um novo evento é detectado.

### Circuit Breaker

Para evitar o efeito de "Cascading Failure", cada provedor de pagamento é monitorado por um **Circuit Breaker**. 
- Se a taxa de erro de um gateway ultrapassa o limite, o circuito se abre.
- As chamadas futuras falham rapidamente sem sobrecarregar o provedor ou degradar seu sistema.
- Após um período de "meio-aberto", o VEAM tenta uma requisição de teste para decidir se fecha o circuito.

### Dead Letter Queue (DLQ)

Eventos que falham repetidamente após o número máximo de tentativas (`WithMaxRetries`) são movidos para a **DLQ (Dead Letter Queue)**. 
- Isso evita que "Poison Pills" (eventos malformatados ou impossíveis de processar) bloqueiem a fila principal.
- Você pode inspecionar e reprocessar eventos da DLQ manualmente via CLI.

---

## Observabilidade & Tracing

O VEAM implementa **OpenTelemetry** nativamente. Ele propaga o contexto de trace (`W3C TraceContext`) entre camadas:

1.  O Webhook chega com um `trace-id`.
2.  O `InboxConsumer` recupera esse ID do banco.
3.  Todo o processamento de domínio e a chamada de saída (Outbox) herdam esse mesmo ID.
4.  Você pode ver todo o caminho do dinheiro no **Jaeger**, **Honeycomb** ou **Cloud Trace**.

---

## Extensibilidade

### Criando um Adaptador

Adicionar um novo gateway ao VEAM é simples. Você só precisa implementar a interface `port.GatewayAdapter`:

```go
type GatewayAdapter interface {
    TranslatePayload(ctx context.Context, payload []byte) (*entity.Transaction, entity.PaymentStatus, error)
    VerifyWebhook(r *http.Request) ([]byte, error)
}
```

Basta registrar sua implementação no `Engine.RegisterProvider` e o motor cuidará do resto.

---

## Manutenção & CLI

A CLI é sua aliada na operação do motor:

- **Migração**: `veam-cli migrate` garante que seu esquema Postgres esteja sempre atualizado.
- **Observação**: (Em breve) Comandos para listar eventos pendentes e status do Circuit Breaker.

---

## 🤝 Contribuição

O VEAM é um projeto de código aberto. Se você deseja contribuir com novos adaptadores, correções de bugs ou melhorias na performance:

1.  Leia nosso [CONTRIBUTING.md](CONTRIBUTING.md).
2.  Garanta que todos os testes passem com `go test ./...`.
3.  Assine seus commits e envie seu Pull Request!

---

## 📄 Licença

VEAM é software livre distribuído sob a licença **MIT**.

---
**Criado com paixão por Victor (victorino0071)**
