package entity

import (
	"time"
)

// GracefulKeyring representa a matriz de chaves criptográficas para um gateway.
// Projetada para imutabilidade total via Atomic Pointers.
type GracefulKeyring struct {
	PrimarySecret   string
	SecondarySecret string
	SecondaryExpiry time.Time
}

// NewKeyring inicializa um novo keyring com apenas a chave primária.
func NewKeyring(secret string) *GracefulKeyring {
	return &GracefulKeyring{
		PrimarySecret: secret,
	}
}

// Rotate gera uma nova instância de Keyring baseada na anterior,
// movendo a primária atual para secundária com um período de graça.
func (k *GracefulKeyring) Rotate(newSecret string, gracePeriod time.Duration) *GracefulKeyring {
	return &GracefulKeyring{
		PrimarySecret:   newSecret,
		SecondarySecret: k.PrimarySecret,
		SecondaryExpiry: time.Now().Add(gracePeriod),
	}
}

// IsSecondaryValid verifica se a chave secundária ainda possui validade temporal (drifting).
func (k *GracefulKeyring) IsSecondaryValid() bool {
	if k.SecondarySecret == "" {
		return false
	}
	return time.Now().Before(k.SecondaryExpiry)
}
