package tcp

import (
	"context"
	"net"
)

type HandlerFunc func(ctx context.Context, conn net.Conn) error

type Handler interface {
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}
