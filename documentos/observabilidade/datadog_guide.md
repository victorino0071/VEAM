# Guia de Observabilidade: Usando Datadog com Asaas Framework
**Caminho:** `documentos/observabilidade/datadog_guide.md`

O nosso framework já exporta logs em formato **JSON Estruturado** através da biblioteca `slog`. Isso é ideal para o **Datadog**, pois permite que a plataforma entenda os atributos sem necessidade de Parsers complexos (Grok).

---

## 1. Por que JSON?
Quando você olha no terminal, o JSON pode parecer "poluído", mas para o Datadog ele é ouro:
*   **Atributos Automáticos**: Campos como `msg`, `level`, `id_transacao` e `service` tornam-se colunas clicáveis no Log Explorer.
*   **Busca Semântica**: Você pode buscar por `status:ERROR` ou `target_status:PAID` instantaneamente.

---

## 2. Como enviar os logs para o Datadog?

Existem três formas principais de fazer o seu `main.go` conversar com o Datadog:

### A) Via Datadog Agent (Recomendado para Produção/Docker)
O seu código apenas imprime no `STDOUT` (terminal). O Datadog Agent (rodando como um container separado ou serviço no SO) captura esses bytes e envia para a nuvem.
*   **Vantagem**: Zero impacto de performance na sua aplicação Go.
*   **Configuração**: Ative o `LOGS_ENABLED: true` no seu Agent.

### B) Envio Direto via HTTPS
Você pode configurar um `slog.Handler` customizado que faz um POST para a API do Datadog.
*   **Vantagem**: Não precisa de Agent.
*   **Desvantagem**: Aumenta a latência e o consumo de rede da sua App.

### C) OpenTelemetry Collector (O "Padrão Ouro")
Já inicializamos o OpenTelemetry no projeto. Você pode apontar o exportador para um **OTel Collector**, que então despacha os logs e traces para o Datadog.

---

## 3. Como visualizar no Datadog Log Explorer

Ao abrir o painel do Datadog, siga estes passos para extrair o máximo do fluxo:

1.  **Filtro por Serviço**: Use `service:asaas-framework` para isolar os logs do projeto.
2.  **Rastreabilidade (The "Red Line")**: 
    *   Graças ao uso do `Metadata` JSONB no nosso banco de dados, o **TraceID** viaja junto com a mensagem do Inbox até o Outbox.
    *   No Datadog, clique em um log e selecione **"View Related Traces"**. Você verá uma cascata milimétrica de quanto tempo levou desde o `ReceiveWebhook` até o `Refund` no gateway externo.
3.  **Facetas (Facets)**: Crie facetas para campos como `event_type` e `status`. Isso permite criar dashboards que mostram, por exemplo: *"Quantos pagamentos deram ANOMALY nas últimas 24h?"*.

---

## 4. Dica de Debugging Rápido
Se uma transação falhar, pegue o `id` da transação no log de erro e jogue na barra de busca. O Datadog filtrará todos os logs (desde a ingestão até a falha do worker) que contenham esse ID, reconstruindo a linha do tempo do erro para você.
