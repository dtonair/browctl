package ipc

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/dtonair/browctl/protocol"
)

// Handler processes a single protocol request and returns exactly one response.
type Handler func(context.Context, protocol.Request) protocol.Response

type Server struct {
	socketPath string
	handler    Handler

	mu       sync.Mutex
	listener net.Listener
}

func NewServer(socketPath string, handler Handler) *Server {
	return &Server{socketPath: socketPath, handler: handler}
}

func (s *Server) Serve(ctx context.Context) error {
	if s.handler == nil {
		return errors.New("ipc server: nil handler")
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	if err := os.Chmod(filepath.Dir(s.socketPath), 0o700); err != nil {
		return fmt.Errorf("chmod socket dir: %w", err)
	}

	if err := removeStaleSocket(s.socketPath); err != nil {
		return err
	}

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen unix %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	defer func() {
		_ = ln.Close()
		_ = os.Remove(s.socketPath)
	}()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		writeResponse(conn, protocol.Fail(protocol.NewError(protocol.InvalidRequest, "read request: "+err.Error(), nil), nil, protocol.Meta{}))
		return
	}

	req, err := protocol.DecodeStrict[protocol.Request](bytes.NewReader(bytes.TrimSpace(line)))
	if err != nil {
		perr, ok := err.(*protocol.Error)
		if !ok {
			perr = protocol.NewError(protocol.InvalidRequest, err.Error(), nil)
		}
		writeResponse(conn, protocol.Fail(perr, nil, protocol.Meta{}))
		return
	}

	resp := s.handler(ctx, req)
	writeResponse(conn, resp)
}

func writeResponse(conn net.Conn, resp protocol.Response) {
	encoded, err := protocol.Encode(resp)
	if err != nil {
		encoded, _ = protocol.Encode(protocol.Fail(protocol.NewError(protocol.InternalError, "encode response: "+err.Error(), nil), nil, protocol.Meta{}))
	}
	_, _ = conn.Write(append(encoded, '\n'))
}

func removeStaleSocket(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat socket: %w", err)
	}

	conn, err := net.Dial("unix", path)
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("socket already in use: %s", path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}
