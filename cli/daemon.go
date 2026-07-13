package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dt/browctl/ipc"
	"github.com/dt/browctl/paths"
	"github.com/dt/browctl/protocol"
	"github.com/spf13/cobra"
)

func newPingCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Ping the local browctld daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "ping"}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			if err := writeResponseOrText(out, *jsonOutput, resp, "pong"); err != nil {
				return err
			}
			if !resp.OK {
				return resp.Error
			}
			return nil
		},
	}
}

func newDaemonCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "daemon", Short: "Manage the local browctld daemon"}
	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start browctld if it is not already running",
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, err := paths.Ensure()
			if err != nil {
				return err
			}
			if resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "daemon.status"}, false); err == nil && resp.OK {
				return writeResponseOrText(out, *jsonOutput, protocol.OK(map[string]any{"status": "running"}, protocol.Meta{}), "running")
			}
			_ = os.Remove(layout.DaemonSocket)
			if err := spawnDaemon(layout.DaemonSocket); err != nil {
				return err
			}
			if err := waitForDaemon(cmd.Context(), layout.DaemonSocket, 3*time.Second); err != nil {
				return err
			}
			return writeResponseOrText(out, *jsonOutput, protocol.OK(map[string]any{"status": "started"}, protocol.Meta{}), "started")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Report browctld status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "daemon.status"}, false)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeResponseOrText(out, *jsonOutput, resp, "running")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop browctld",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "daemon.stop"}, false)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeResponseOrText(out, *jsonOutput, resp, "stopping")
		},
	})
	return cmd
}

func requestDaemon(ctx context.Context, req protocol.Request, autoStart bool) (protocol.Response, error) {
	layout, err := paths.Ensure()
	if err != nil {
		return protocol.Response{}, err
	}
	client := ipc.NewClient(layout.DaemonSocket)
	resp, err := client.Do(ctx, req)
	if err == nil {
		return resp, nil
	}
	if !autoStart {
		return protocol.Response{}, err
	}

	_ = os.Remove(layout.DaemonSocket)
	if err := spawnDaemon(layout.DaemonSocket); err != nil {
		return protocol.Response{}, err
	}
	if err := waitForDaemon(ctx, layout.DaemonSocket, 3*time.Second); err != nil {
		return protocol.Response{}, err
	}
	return client.Do(ctx, req)
}

func writeResponseOrText(out io.Writer, jsonOutput bool, resp protocol.Response, text string) error {
	if jsonOutput {
		if err := writeJSON(out, resp); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(out, text); err != nil {
			return err
		}
	}
	if !resp.OK {
		return resp.Error
	}
	return nil
}

func writeProtocolError(out io.Writer, jsonOutput bool, code protocol.ErrorCode, message string) error {
	perr := protocol.NewError(code, message, nil)
	if jsonOutput {
		_ = writeJSON(out, protocol.Fail(perr, nil, protocol.Meta{}))
	}
	return perr
}

func spawnDaemon(socketPath string) error {
	daemonPath, err := findDaemonBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(daemonPath, "-socket", socketPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browctld: %w", err)
	}
	return nil
}

func findDaemonBinary() (string, error) {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "browctld")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("browctld"); err == nil {
		return path, nil
	}
	return "", errors.New("browctld binary not found next to browctl or in PATH")
}

func waitForDaemon(ctx context.Context, socketPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := ipc.NewClient(socketPath)
	client.Timeout = 250 * time.Millisecond
	for {
		resp, err := client.Do(ctx, protocol.Request{Cmd: "daemon.status"})
		if err == nil && resp.OK {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for daemon: %w", ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}
