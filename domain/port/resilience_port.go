package port

import (
	"context"
)

// CircuitBreaker define a interface para o padrão de disjuntor.
type CircuitBreaker interface {
	// Allow verifica se a operação é permitida.
	Allow(ctx context.Context) (bool, error)
	
	// RecordResult registra o sucesso ou falha de uma operação.
	RecordResult(ctx context.Context, err error) error
	
	// GetState retorna o estado atual ("CLOSED", "OPEN", "HALF_OPEN").
	GetState(ctx context.Context) (string, error)
	
	// GetFailureProbability expõe a métrica matemática EWMA (0.0 a 1.0).
	GetFailureProbability(ctx context.Context) (float64, error)
}
