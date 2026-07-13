package browser

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/dt/browctl/profile"
	"github.com/dt/browctl/protocol"
)

func TestStopStaleRuntimeSurfacesBrowserCrashed(t *testing.T) {
	profiles := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := profiles.Create("work", ""); err != nil {
		t.Fatalf("Create profile: %v", err)
	}
	mgr := NewManager(profiles, filepath.Join(t.TempDir(), "run"))
	stale := Runtime{Profile: "work", PID: 99999999, Port: 9222, Endpoint: "http://127.0.0.1:9222", StartedAt: time.Now().UTC(), State: StateRunning}
	if err := mgr.writeRuntime(stale); err != nil {
		t.Fatalf("writeRuntime: %v", err)
	}

	_, err := mgr.Stop(context.Background(), "work")
	if err == nil {
		t.Fatal("Stop() error = nil, want BROWSER_CRASHED")
	}
	var perr *protocol.Error
	if !errors.As(err, &perr) || perr.Code != protocol.BrowserCrashed {
		t.Fatalf("Stop() error = %#v, want BROWSER_CRASHED", err)
	}
	if _, err := mgr.ReadRuntime("work"); err == nil {
		t.Fatal("runtime still exists after crashed cleanup")
	}
}
