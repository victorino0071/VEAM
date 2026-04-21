package entity

import "errors"

// ErrTerminalGatewayRejection indica que um serviço ou gateway recusou definitivamente 
// a operação devido a regras de negócio (ex: Saldo Insuficiente, Requisição Inválida, Cartão Roubado).
// Diferencia-se de falhas de rede (transientes) que induzem backoff.
var ErrTerminalGatewayRejection = errors.New("terminal gateway rejection: business error (irrecoverable)")
