package terminal

import (
	"context"
	"io"

	"suda-forge/internal/runtime"
)

type Manager interface {
	Create(context.Context, SessionSpec) (Session, error)
	Attach(context.Context, string) (io.ReadWriteCloser, error)
	Resize(context.Context, string, uint16, uint16) error
	Close(context.Context, string) error
}

type SessionSpec struct {
	ID        string
	RuntimeID string
	Shell     string
	Workdir   string
}

type Session struct {
	ID        string `json:"id"`
	RuntimeID string `json:"runtime_id"`
	Status    string `json:"status"`
}

var _ runtime.Provider
