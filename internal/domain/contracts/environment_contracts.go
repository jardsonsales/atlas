package contracts

import (
	"context"
)

// DomainEnvWriter writes the domain operation port into its environment file (.env).
// Atlas never manages the .env beyond the PORT key: every other entry is preserved, and
// the file is created with just PORT=<port> when it does not exist yet.
type DomainEnvWriter interface {
	WriteOperationPort(ctx context.Context, domain string, port int) error
}

// AllocateOperationPortUseCase defines the use case contract for reserving the next free
// operation port for a domain (sequential from 4000, or the port already assigned to it).
type AllocateOperationPortUseCase interface {
	Execute(ctx context.Context, domain string) (int, error)
}
