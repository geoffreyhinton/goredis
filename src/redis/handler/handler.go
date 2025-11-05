package handler

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	DBImpl "github.com/geoffreyhinton/goredis/src/db"
	"github.com/geoffreyhinton/goredis/src/interface/db"
	"github.com/geoffreyhinton/goredis/src/lib/logger"
	"github.com/geoffreyhinton/goredis/src/lib/sync/atomic"
	"github.com/geoffreyhinton/goredis/src/redis/parser"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

var (
	UnknownErrReplyBytes = []byte("-ERR unknown\r\n")
)

type Handler struct {
	activeConn sync.Map // *client -> placeholder
	db         db.DB
	closing    atomic.AtomicBool // refusing new client and new request
}

func MakeHandler() *Handler {
	return &Handler{
		db: DBImpl.MakeDB(),
	}
}

func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	if h.closing.Get() {
		// closing handler refuse new connection
		conn.Close()
	}

	client := MakeClient(conn, h.db.(*DBImpl.DB))
	h.activeConn.Store(client, 1)

	reader := bufio.NewReader(conn)
	for {
		// may occurs: client EOF, client timeout, server early close
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				logger.Info("connection close")
			} else {
				logger.Warn(err)
			}
			client.Close()
			h.activeConn.Delete(client)
			return // io error, disconnect with client
		}

		if len(msg) == 0 {
			continue // ignore empty request
		}

		if !client.sending.Get() {
			// new request
			if msg[0] == '*' {
				// bulk multi msg
				expectedLine, err := strconv.ParseUint(string(msg[1:len(msg)-2]), 10, 32)
				if err != nil {
					client.conn.Write(UnknownErrReplyBytes)
					continue
				}
				expectedLine *= 2
				client.waitingReply.Add(1)
				client.sending.Set(true)
				client.expectedLineCount = uint32(expectedLine)
				client.sentLineCount = 0
				client.sentLines = make([][]byte, expectedLine)
			} else {
				// TODO: text protocol
			}
		} else {
			// receive following part of a request
			client.sentLines[client.sentLineCount] = msg[0 : len(msg)-2]
			client.sentLineCount++
			// if sending finished
			if client.sentLineCount == client.expectedLineCount {
				client.sending.Set(false) // finish sending progress
				// exec cmd
				if len(client.sentLines)%2 != 0 {
					client.conn.Write(UnknownErrReplyBytes)
					client.expectedLineCount = 0
					client.sentLineCount = 0
					client.sentLines = nil
					client.waitingReply.Done()
					continue
				}

				// send reply
				args := parser.Parse(client.sentLines)

				// Create authentication context for this client
				authCtx := &DBImpl.AuthContext{
					User:          client.GetUser(),
					Authenticated: client.IsAuthenticated(),
				}

				// Execute command with authentication
				result := h.db.(*DBImpl.DB).AuthenticatedExec(authCtx, args)

				// Update client authentication state if AUTH command was successful
				if len(args) > 0 && strings.ToLower(string(args[0])) == "auth" {
					if _, ok := result.(*reply.OkReply); ok {
						// Get the authenticated user
						var username string
						if len(args) == 2 {
							username = "default"
						} else if len(args) == 3 {
							username = string(args[1])
						}

						if user, exists := h.db.(*DBImpl.DB).Auth.GetUser(username); exists {
							client.SetUser(user)
						}
					}
				}

				if result != nil {
					conn.Write(result.ToBytes())
				} else {
					conn.Write(UnknownErrReplyBytes)
				}

				// finish reply
				client.expectedLineCount = 0
				client.sentLineCount = 0
				client.sentLines = nil
				client.waitingReply.Done()
			}
		}

	}
}

func (h *Handler) Close() error {
	logger.Info("handler shuting down...")
	h.closing.Set(true)
	// TODO: concurrent wait
	h.activeConn.Range(func(key interface{}, val interface{}) bool {
		client := key.(*Client)
		client.Close()
		return true
	})
	return nil
}
