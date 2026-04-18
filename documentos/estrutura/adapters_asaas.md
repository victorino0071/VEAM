# Adaptores: Asaas (Conector de Pagamento)
**Caminho:** `adapters/asaas`

O adaptador Asaas fornece a implementação concreta dos contratos de domínio para o gateway [asaas.com](https://asaas.com).

## 🧩 Responsabilidades
1.  **Transporte HTTP:** Implementa o `doRequest` com tratamento de autenticação via `access_token` e BaseURL.
2.  **Mapeamento Outbound:** Converte entidades de domínio (`entity.Transaction`) para os DTOs proprietários do Asaas.
3.  **Validação de Webhook:** Verifica assinaturas de requisições de entrada para garantir que proveem de fontes confiáveis.
4.  **Normalização Inbound:** Implementa o `TranslateWebhook` para transformar payloads proprietários no contrato universal `port.WebhookResponse`.

## ⚙️ Configuração
O adaptador é injetado via `RegisterProvider` na `Engine`:

```go
engine.RegisterProvider("asaas", asaas.NewAdapter(apiKey, baseUrl))
```
