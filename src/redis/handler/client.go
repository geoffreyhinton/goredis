package handler

import (
	"net"
	"time"

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
}

func (c *Client) Close() error {
	c.waitingReply.WaitWithTimeout(10 * time.Second)
	c.conn.Close()
	return nil
}

func MakeClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}
