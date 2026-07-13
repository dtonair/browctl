package ipc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dt/browctl/protocol"
)

type Client struct {
	SocketPath string
	Timeout    time.Duration
}

func NewClient(socketPath string) Client {
	return Client{SocketPath: socketPath, Timeout: 5 * time.Second}
}

func (c Client) Do(ctx context.Context, req protocol.Request) (protocol.Response, error) {
	if req.APIVersion == 0 {
		req.APIVersion = protocol.APIVersion
	}

	dialer := net.Dialer{}
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	conn, err := dialer.DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return protocol.Response{}, fmt.Errorf("dial daemon: %w", err)
	}
	defer conn.Close()

	encoded, err := protocol.Encode(req)
	if err != nil {
		return protocol.Response{}, fmt.Errorf("encode request: %w", err)
	}
	if _, err := conn.Write(append(encoded, '\n')); err != nil {
		return protocol.Response{}, fmt.Errorf("write request: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return protocol.Response{}, fmt.Errorf("read response: %w", err)
	}
	resp, err := protocol.DecodeStrict[protocol.Response](bytes.NewReader(bytes.TrimSpace(line)))
	if err != nil {
		return protocol.Response{}, fmt.Errorf("decode response: %w", err)
	}
	return resp, nil
}
