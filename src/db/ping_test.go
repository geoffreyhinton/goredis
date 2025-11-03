package db

import (
	"bufio"
	"net"
	"testing"
	"time"
)

func TestPingPong(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "localhost:6399", 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send PING command in RESP protocol
	_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		t.Fatalf("failed to send PING: %v", err)
	}

	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if resp != "+PONG\r\n" {
		t.Errorf("expected '+PONG\\r\\n', got '%s'", resp)
	}
}
