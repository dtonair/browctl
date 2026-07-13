package ipc

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/dtonair/browctl/protocol"
)

func TestServerClientPingAndSocketPermissions(t *testing.T) {
	socket := tempSocket(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(socket, func(ctx context.Context, req protocol.Request) protocol.Response {
		if req.Cmd != "ping" {
			return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "bad cmd", nil), nil, protocol.Meta{})
		}
		return protocol.OK(map[string]any{"pong": true}, protocol.Meta{})
	})

	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	waitForSocket(t, socket)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(socket)
		if err != nil {
			t.Fatalf("stat socket: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("socket permissions = %o, want 600", got)
		}
	}

	resp, err := NewClient(socket).Do(context.Background(), protocol.Request{Cmd: "ping"})
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false: %#v", resp.Error)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerRejectsInvalidRequest(t *testing.T) {
	socket := tempSocket(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(socket, func(ctx context.Context, req protocol.Request) protocol.Response {
		return protocol.OK(nil, protocol.Meta{})
	})
	go func() { _ = server.Serve(ctx) }()
	waitForSocket(t, socket)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(`{"api_version":1,"cmd":"ping","unknown":true}` + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	line := make([]byte, 512)
	n, err := conn.Read(line)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	resp, err := protocol.DecodeStrict[protocol.Response](bytes.NewReader(line[:n]))
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.OK || resp.Error == nil || resp.Error.Code != protocol.InvalidRequest {
		t.Fatalf("resp = %#v, want INVALID_REQUEST", resp)
	}
}

func TestServerRemovesStaleSocket(t *testing.T) {
	socket := tempSocket(t)
	if err := os.WriteFile(socket, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale socket: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := NewServer(socket, func(ctx context.Context, req protocol.Request) protocol.Response {
		return protocol.OK(map[string]any{"pong": true}, protocol.Meta{})
	})
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	waitForSocket(t, socket)

	resp, err := NewClient(socket).Do(context.Background(), protocol.Request{Cmd: "ping"})
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false: %#v", resp.Error)
	}
	cancel()
	<-done
}

func waitForSocket(t *testing.T, socket string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s was not ready", socket)
}

func tempSocket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "browctl-ipc-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "d.sock")
}
