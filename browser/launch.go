package browser

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dtonair/browctl/protocol"
)

const devToolsActivePortFile = "DevToolsActivePort"

type LaunchConfig struct {
	Profile        string
	ChromePath     string
	UserDataDir    string
	Headless       bool
	ExtraArgs      []string
	StartupTimeout time.Duration
}

type launchedChrome struct {
	cmd        *exec.Cmd
	done       <-chan error
	port       int
	endpoint   string
	wsEndpoint string
}

func launchChrome(ctx context.Context, cfg LaunchConfig) (*launchedChrome, error) {
	if cfg.StartupTimeout == 0 {
		cfg.StartupTimeout = 10 * time.Second
	}
	chromePath, err := resolveChromePath(cfg.ChromePath)
	if err != nil {
		return nil, err
	}
	if cfg.UserDataDir == "" {
		return nil, protocol.NewError(protocol.InvalidRequest, "user data dir is required", nil)
	}
	if err := os.MkdirAll(cfg.UserDataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create user data dir: %w", err)
	}
	_ = os.Remove(filepath.Join(cfg.UserDataDir, devToolsActivePortFile))

	args := []string{
		"--user-data-dir=" + cfg.UserDataDir,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
	}
	if cfg.Headless || truthyEnv("BROWCTL_HEADLESS") {
		args = append(args, "--headless=new", "--disable-gpu")
	}
	args = append(args, cfg.ExtraArgs...)

	cmd := exec.CommandContext(ctx, chromePath, args...)
	stdout := &limitedBuffer{max: 16 * 1024}
	stderr := &limitedBuffer{max: 16 * 1024}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, protocol.NewError(protocol.BrowserStartFailed, "start Chrome: "+err.Error(), map[string]any{"chrome_path": chromePath})
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	port, wsPath, err := waitDevToolsActivePort(ctx, cfg.UserDataDir, cfg.StartupTimeout, done, stderr)
	if err != nil {
		if processAlive(cmd.Process.Pid) {
			_ = cmd.Process.Kill()
			<-done
		}
		return nil, protocol.NewError(protocol.BrowserStartFailed, err.Error(), map[string]any{"chrome_path": chromePath})
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	wsEndpoint := fmt.Sprintf("ws://127.0.0.1:%d%s", port, wsPath)
	return &launchedChrome{cmd: cmd, done: done, port: port, endpoint: endpoint, wsEndpoint: wsEndpoint}, nil
}

func waitDevToolsActivePort(ctx context.Context, userDataDir string, timeout time.Duration, done <-chan error, stderr fmt.Stringer) (int, string, error) {
	path := filepath.Join(userDataDir, devToolsActivePortFile)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		port, wsPath, err := readDevToolsActivePort(path)
		if err == nil {
			return port, wsPath, nil
		}
		select {
		case <-ctx.Done():
			return 0, "", fmt.Errorf("wait for DevToolsActivePort: %w", ctx.Err())
		case err := <-done:
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				return 0, "", fmt.Errorf("Chrome exited before DevToolsActivePort: %v: %s", err, msg)
			}
			return 0, "", fmt.Errorf("Chrome exited before DevToolsActivePort: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitDevToolsActivePortFile(ctx context.Context, userDataDir string, timeout time.Duration) (int, string, error) {
	return waitDevToolsActivePort(ctx, userDataDir, timeout, nil, bytes.NewBuffer(nil))
}

func readDevToolsActivePort(path string) (int, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, "", errors.New("DevToolsActivePort missing port")
	}
	port, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || port <= 0 {
		return 0, "", fmt.Errorf("invalid DevToolsActivePort port: %q", scanner.Text())
	}
	if !scanner.Scan() {
		return 0, "", errors.New("DevToolsActivePort missing websocket path")
	}
	wsPath := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(wsPath, "/") {
		return 0, "", fmt.Errorf("invalid DevToolsActivePort websocket path: %q", wsPath)
	}
	if err := scanner.Err(); err != nil {
		return 0, "", err
	}
	return port, wsPath, nil
}

func truthyEnv(name string) bool {
	switch strings.ToLower(os.Getenv(name)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveChromePath(configured string) (string, error) {
	candidates := []string{}
	if configured != "" {
		candidates = append(candidates, configured)
	}
	if env := os.Getenv("BROWCTL_CHROME"); env != "" {
		candidates = append(candidates, env)
	}
	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium")
	case "linux":
		candidates = append(candidates, "/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium", "/usr/bin/chromium-browser")
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, path)
		}
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", protocol.NewError(protocol.BrowserNotFound, "Chrome executable not found", nil)
}
