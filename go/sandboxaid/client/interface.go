package client

import (
	"context"
	"errors"

	v1 "github.com/substratusai/sandboxai/go/api/v1"
)

var ErrSandboxNotFound = errors.New("sandbox not found")

type Sandbox struct {
	*v1.Sandbox
	BoxHostPort int
}

type Client interface {
	CreateSandbox(ctx context.Context, space string, req *v1.CreateSandboxRequest) (*Sandbox, error)
	GetSandbox(ctx context.Context, space, name string) (*Sandbox, error)
	DeleteSandbox(ctx context.Context, space, name string) error
}
