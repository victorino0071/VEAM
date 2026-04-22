# 🛡️ Deduplicação Industrial (Fingerprinting)

O VEAM utiliza um sistema de **Deduplicação de Negócio O(1)** para garantir que instabilidades na rede de entrega dos gateways não causem reprocessamentos desnecessários ou inconsistências financeiras.

## 🧠 O Conceito: Identidade de Entrega vs. Negócio

Em sistemas distribuídos, um único evento de negócio (ex: "Pagamento #123 Confirmado") pode ser entregue múltiplas vezes por motivos variados:
1.  **Retentativa do Gateway:** O gateway não recebeu o `200 OK` a tempo e reenvia o evento.
2.  **Mutação de Transporte:** Muitos gateways geram um **novo ID de Webhook** e um novo **Timestamp** a cada tentativa.

Se confiarmos apenas no ID do Webhook (`external_id`), o sistema aceitaria o mesmo evento várias vezes. O VEAM resolve isso através do **Fingerprinting Semântico**.

## 🛠️ Implementação Técnica

### 1. Extração do "Core" (Adaptador)
Cada `GatewayAdapter` implementa o método `Fingerprint(payload []byte)`. A responsabilidade do adaptador é limpar o JSON e extrair apenas os campos que definem a transação:
- **TransactionID** (ID do recurso no gateway)
- **Status** (Novo estado)
- **Amount** (Valor monetário)
- **Currency** (Moeda)

Qualquer campo volátil (IDs de entrega, datas de envio, headers dinâmicos) é **ignorado**.

### 2. Hashing com xxHash
O core extraído é concatenado e processado pelo algoritmo **xxHash64**. Escolhemos o xxHash por:
- **Performance:** É significativamente mais rápido que SHA-256 para payloads curtos.
- **Colisão:** Probabilidade desprezível para o volume de webhooks de um motor de pagamentos.

### 3. Idempotência em Nível de Banco (Composite Index)
O fingerprint é persistido na tabela `inbox` com uma restrição de unicidade composta:
```sql
CREATE UNIQUE INDEX idx_inbox_fingerprint ON inbox ((metadata->>'provider_id'), fingerprint);
```
O uso do `provider_id` no índice isola os universos criptográficos, garantindo que um hash idêntico em dois gateways diferentes não cause colisão.

## 🚦 Fluxo de Silenciamento

1.  **Ingestão:** O `WebhookHandler` gera o fingerprint via adaptador.
2.  **Persistência:** Tenta salvar no banco usando `ON CONFLICT DO NOTHING`.
3.  **Detecção:** Se `RowsAffected` for 0, o motor identifica uma duplicata semântica.
4.  **Resposta:** O handler retorna **HTTP 202 Accepted**.
    - O gateway para de tentar (recebeu um status de sucesso).
    - O sistema interno não gasta recursos processando algo que já foi feito.

## 🧪 Verificação
Para validar a estabilidade de um novo adaptador, deve-se garantir que dois JSONs com IDs de entrega diferentes mas mesmo conteúdo de negócio gerem o mesmo hash hexadecimal.
