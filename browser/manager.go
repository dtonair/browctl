package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"github.com/dtonair/browctl/profile"
	"github.com/dtonair/browctl/protocol"
)

type Manager struct {
	profiles   *profile.Manager
	runtimeDir string

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	runtime Runtime
	cmdPID  int
	lock    *profile.Lock

	allocCancel   context.CancelFunc
	browserCtx    context.Context
	browserCancel context.CancelFunc
	done          <-chan error
	stopping      bool
}

func NewManager(profiles *profile.Manager, runtimeDir string) *Manager {
	return &Manager{profiles: profiles, runtimeDir: runtimeDir, sessions: map[string]*session{}}
}

func (m *Manager) Start(ctx context.Context, name string) (Runtime, error) {
	if name == "" {
		return Runtime{}, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	if rt, ok := m.runningRuntime(name); ok {
		return rt, nil
	}
	p, err := m.profiles.Inspect(name)
	if err != nil {
		return Runtime{}, err
	}

	if rt, ok := m.readLiveRuntime(name); ok {
		if _, err := m.attach(ctx, rt, nil); err == nil {
			return rt, nil
		}
		_ = m.removeRuntime(name)
	}

	lock, err := m.profiles.Acquire(name, profile.LockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		return Runtime{}, err
	}
	cleanupLock := true
	defer func() {
		if cleanupLock {
			_ = lock.Release()
		}
	}()

	launched, err := launchChrome(ctx, LaunchConfig{Profile: name, ChromePath: p.ChromePath, UserDataDir: m.profiles.ChromeDataDir(name)})
	if err != nil {
		return Runtime{}, err
	}

	rt := Runtime{
		Profile:    name,
		PID:        launched.cmd.Process.Pid,
		Port:       launched.port,
		Endpoint:   launched.endpoint,
		WSEndpoint: launched.wsEndpoint,
		ChromePath: launched.cmd.Path,
		StartedAt:  time.Now().UTC(),
		State:      StateRunning,
	}
	if err := lock.Update(profile.LockInfo{PID: rt.PID, StartedAt: rt.StartedAt, Endpoint: rt.Endpoint}); err != nil {
		_ = launched.cmd.Process.Kill()
		_, _ = launched.cmd.Process.Wait()
		return Runtime{}, err
	}
	if err := m.writeRuntime(rt); err != nil {
		_ = launched.cmd.Process.Kill()
		_, _ = launched.cmd.Process.Wait()
		return Runtime{}, err
	}

	sess, err := m.attach(ctx, rt, lock)
	if err != nil {
		_ = m.removeRuntime(name)
		_ = launched.cmd.Process.Kill()
		_, _ = launched.cmd.Process.Wait()
		return Runtime{}, err
	}
	sess.cmdPID = rt.PID
	sess.done = launched.done
	go m.watch(name, sess)

	cleanupLock = false
	return rt, nil
}

func (m *Manager) Stop(ctx context.Context, name string) (Runtime, error) {
	if name == "" {
		return Runtime{}, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	sess := m.getSession(name)
	rt, err := m.ReadRuntime(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Runtime{Profile: name, State: StateStopped}, nil
		}
		return Runtime{}, err
	}
	if rt.State == StateCrashed || !processAlive(rt.PID) {
		_ = m.profiles.ReleaseLockForPID(name, rt.PID)
		_ = m.removeRuntime(name)
		return Runtime{}, protocol.NewError(protocol.BrowserCrashed, "browser crashed", map[string]any{"profile": name, "pid": rt.PID})
	}

	if sess != nil {
		m.mu.Lock()
		sess.stopping = true
		m.mu.Unlock()
		if sess.browserCtx != nil {
			closeCtx, cancel := context.WithTimeout(sess.browserCtx, 2*time.Second)
			_ = cdpbrowser.Close().Do(closeCtx) // best effort; SIGTERM/SIGKILL fallback below.
			cancel()
		}
	}

	_ = signalPID(rt.PID, syscall.SIGTERM)
	if !waitProcessGone(ctx, rt.PID, 5*time.Second) {
		_ = signalPID(rt.PID, syscall.SIGKILL)
		_ = waitProcessGone(ctx, rt.PID, 3*time.Second)
	}

	if sess != nil {
		if sess.browserCancel != nil {
			sess.browserCancel()
		}
		if sess.allocCancel != nil {
			sess.allocCancel()
		}
		if sess.lock != nil {
			_ = sess.lock.Release()
		}
	}
	_ = m.profiles.ReleaseLockForPID(name, rt.PID)
	_ = m.removeRuntime(name)
	m.deleteSession(name)
	rt.State = StateStopped
	return rt, nil
}

func (m *Manager) BrowserContext(name string) (context.Context, error) {
	sess := m.getSession(name)
	if sess != nil && sess.browserCtx != nil && processAlive(sess.runtime.PID) {
		return sess.browserCtx, nil
	}

	rt, ok := m.readLiveRuntime(name)
	if !ok {
		return nil, protocol.NewError(protocol.BrowserNotFound, "browser is not running", map[string]any{"profile": name})
	}
	reattached, err := m.attach(context.Background(), rt, nil)
	if err != nil {
		return nil, err
	}
	return reattached.browserCtx, nil
}

func (m *Manager) ReadRuntime(name string) (Runtime, error) {
	data, err := os.ReadFile(m.runtimePath(name))
	if err != nil {
		return Runtime{}, err
	}
	var rt Runtime
	if err := json.Unmarshal(data, &rt); err != nil {
		return Runtime{}, fmt.Errorf("decode runtime: %w", err)
	}
	if !processAlive(rt.PID) {
		rt.State = StateCrashed
	}
	return rt, nil
}

func (m *Manager) runningRuntime(name string) (Runtime, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[name]
	if !ok || sess.runtime.State != StateRunning || !processAlive(sess.runtime.PID) {
		return Runtime{}, false
	}
	return sess.runtime, true
}

func (m *Manager) readLiveRuntime(name string) (Runtime, bool) {
	rt, err := m.ReadRuntime(name)
	if err != nil || rt.State != StateRunning || !processAlive(rt.PID) {
		return Runtime{}, false
	}
	return rt, true
}

func (m *Manager) attach(ctx context.Context, rt Runtime, lock *profile.Lock) (*session, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), rt.Endpoint)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	checkCtx, cancel := context.WithTimeout(browserCtx, 5*time.Second)
	defer cancel()
	if _, err := chromedp.Targets(checkCtx); err != nil {
		browserCancel()
		allocCancel()
		return nil, protocol.NewError(protocol.BrowserStartFailed, "attach Chrome: "+err.Error(), map[string]any{"endpoint": rt.Endpoint})
	}
	sess := &session{runtime: rt, lock: lock, allocCancel: allocCancel, browserCtx: browserCtx, browserCancel: browserCancel}
	m.mu.Lock()
	m.sessions[rt.Profile] = sess
	m.mu.Unlock()
	return sess, nil
}

func (m *Manager) getSession(name string) *session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[name]
}

func (m *Manager) deleteSession(name string) {
	m.mu.Lock()
	delete(m.sessions, name)
	m.mu.Unlock()
}

func (m *Manager) watch(name string, sess *session) {
	if sess.done == nil {
		return
	}
	<-sess.done
	m.mu.Lock()
	stopping := sess.stopping
	m.mu.Unlock()
	if !stopping {
		rt := sess.runtime
		rt.State = StateCrashed
		_ = m.writeRuntime(rt)
		if sess.lock != nil {
			_ = sess.lock.Release()
		} else {
			_ = m.profiles.ReleaseLockForPID(name, rt.PID)
		}
		m.deleteSession(name)
	}
}

func (m *Manager) writeRuntime(rt Runtime) error {
	if err := os.MkdirAll(m.runtimeDir, 0o700); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}
	if err := os.Chmod(m.runtimeDir, 0o700); err != nil {
		return fmt.Errorf("chmod runtime dir: %w", err)
	}
	data, err := json.MarshalIndent(rt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime: %w", err)
	}
	if err := os.WriteFile(m.runtimePath(rt.Profile), append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write runtime: %w", err)
	}
	return nil
}

func (m *Manager) removeRuntime(name string) error {
	if err := os.Remove(m.runtimePath(name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) runtimePath(name string) string {
	return filepath.Join(m.runtimeDir, name+".json")
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func signalPID(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

func waitProcessGone(ctx context.Context, pid int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if !processAlive(pid) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
}
