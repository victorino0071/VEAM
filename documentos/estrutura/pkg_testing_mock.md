# Documentação de Estrutura: Mocking Engine (Simulador Industrial)
**Caminho:** `pkg/testing/mock`

O motor de mocking do Payment Engine foi desenvolvido como uma ferramenta de primeira classe para garantir que o sistema possa ser testado de forma determinística em CI/CD e de forma resiliente contra cenários de caos.

---

## 1. O Conceito de Dual-Engine (Determinismo vs. Caos)
*( **Conceito Técnico - Chaos Engineering:** É a disciplina de experimentar em um sistema para construir confiança na sua capacidade de suportar condições turbulentas. Nosso mock permite injetar latência e falhas de rede propositais para ver se o Circuit Breaker abre corretamente. )*

---

## 2. A Árvore de Resolução Sequencial (L1, L2, L3)

Diferente de mocks tradicionais que retornam sempre o mesmo valor, o nosso `MockProvider` avalia cada chamada através de uma hierarquia de três camadas de decisão:

### L1: Magic Overrides (Injeção Direta)
*   **Mecânica:** Um mapa de memória (`map`) protegido por Trava de Leitura/Escrita (`sync.RWMutex`).
*   **Uso:** Quando você precisa que um ID específico (ex: `tx_fatal`) retorne um erro específico SEMPRE. É o nível mais alto de prioridade e ignora todas as outras regras.

### L2: Predicados Determinísticos (Regras de Negócio)
*   **Mecânica:** Uma lista de funções Go (`MockRule`) que recebem a transação de domínio e o contexto.
*   **Uso:** Permite escrever lógica Go real no teste para decidir o retorno (ex: "Se o valor for > 10.000,00, retorne Suspeita de Fraude"). Isso permite testes reprodutíveis que não dependem de probabilidade.

### L3: Chaos Engine (Motor Estocástico)
*   **Mecânica:** Aplica probabilidade randômica (`ChaosRate`) e latência configurável (`Jitter`).
*   **Uso:** É o fallback. Se nenhuma regra L1 ou L2 for satisfeita, o motor decide aleatoriamente se a chamada falha ou demora, servindo para testar os limites do sistema.

---

## 3. Segurança e Telemetria

*   **Thread-Safety:** O uso de Mutexes garante que o mock pode ser configurado dinamicamente durante suítes de teste que rodam em paralelo (`t.Parallel()`).
*   **Context Awareness:** Todas as regras e simulações respeitam o `context.Context`. Se o sistema sofrer um timeout ou cancelamento, o mock interrompe a latência e libera as threads imediatamente.

---

> [!TIP]
> **Como rodar demonstrações:** Você pode ver esse motor em ação rodando os scripts de laboratório:
> `go test -v ./scripts/...`
