# Adaptadores: Mercado Pago (Conector via SDK)
**Caminho:** `adapters/mercadopago`

O adaptador Mercado Pago fornece integração com o gateway [mercadopago.com.br](https://mercadopago.com.br) utilizando o SDK oficial para Go.

## 🧩 Responsabilidades
1.  **SDK Orchestration:** Utiliza `github.com/mercadopago/sdk-go` para criação de transações e estornos.
2.  **Validação de Assinatura:** Implementa verificação rigorosa de HMAC usando o cabeçalho `x-signature` e o manifesto de campos (`id`, `request-id`, `ts`).
3.  **Graceful Secret Rotation:** Suporta o padrão de `Keyring`, permitindo trocar o segredo do webhook sem perder notificações em trânsito.
4.  **Fetching Reativo:** No método `TranslatePayload`, o adaptador não confia apenas no JSON recebido via webhook; ele realiza uma consulta (`Get`) à API oficial do Mercado Pago para confirmar o estado real da transação antes de entregá-la ao domínio.
5.  **Correlação de Identidade:** Utiliza o campo `ExternalReference` do Mercado Pago para mapear o ID interno (UUID) do motor. Isso garante a integridade referencial e evita a criação de transações duplicadas durante o processamento de webhooks.
6.  **Suporte a Cartão de Crédito:** Permite a ingestão de `card_token`, parcelas (`installments`) e dados de identificação (`CPF`) via metadados da transação, suportando fluxos de aprovação síncrona.

## ⚙️ Configuração
O adaptador exige um Access Token e um Webhook Secret:

```go
adapter, err := mercadopago.NewAdapter(accessToken, webhookSecret)
if err != nil {
    log.Fatal(err)
}
engine.RegisterProvider("mercadopago", adapter)
```

## 🔒 Idempotência no Estorno
O adaptador injeta automaticamente uma chave de idempotência (`X-Idempotency-Key`) em todas as solicitações de estorno, utilizando o `ID` do evento de outbox original. Isso previne estornos duplicados em caso de retentativas de rede no Relay.

## 📊 Mapeamento de Status
Os status do Mercado Pago são normalizados para o domínio universal:
- `approved` -> `entity.StatusPaid`
- `pending` / `in_process` -> `entity.StatusPending`
- `rejected` -> `entity.StatusFailed`
- `refunded` -> `entity.StatusRefunded`
