package handler

import (
	"net"
	"time"

	DBImpl "github.com/geoffreyhinton/goredis/src/db"
	"github.com/geoffreyhinton/goredis/src/lib/sync/atomic"
	"github.com/geoffreyhinton/goredis/src/lib/sync/wait"
)

type Client struct {
	conn              net.Conn
	waitingReply      wait.Wait
	sending           atomic.AtomicBool
	expectedLineCount uint32
	sentLineCount     uint32
	sentLines         [][]byte

	// Authentication fields
	authenticated bool
	user          *DBImpl.User
}

func (c *Client) Close() error {
	c.waitingReply.WaitWithTimeout(10 * time.Second)
	c.conn.Close()
	return nil
}

// SetUser sets the authenticated user for this client
func (c *Client) SetUser(user *DBImpl.User) {
	c.user = user
	c.authenticated = true
}

// GetUser returns the current user for this client
func (c *Client) GetUser() *DBImpl.User {
	return c.user
}

// IsAuthenticated returns whether the client is authenticated
func (c *Client) IsAuthenticated() bool {
	return c.authenticated
}

func MakeClient(conn net.Conn, db *DBImpl.DB) *Client {
	return &Client{
		conn:          conn,
		authenticated: false,
		user:          db.Auth.GetDefaultUser(), // Start with default user
	}
}
