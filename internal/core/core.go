package core

import (
	"context"
	"os"
	"time"
	"tunnelctl/internal/domain"
)

type Listener struct {
	Port     uint16
	Username string
	Password string
}
type Capabilities struct {
	Protocol   string   `json:"protocol"`
	Methods    []string `json:"methods"`
	Mode       string   `json:"mode"`
	Transport  string   `json:"transport"`
	Statistics bool     `json:"statistics"`
}
type Process interface {
	Done() <-chan struct{}
	ExitError() error
	Ready(context.Context, Listener) error
	ListenerReady(Listener) bool
	Stop(time.Duration) error
}
type Adapter interface {
	Version(context.Context) (string, error)
	Capabilities() Capabilities
	Render(domain.Profile, Listener) ([]byte, error)
	Validate(context.Context, *os.File) error
	Start(*os.File) (Process, error)
	Close() error
}
