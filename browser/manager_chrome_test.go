package browser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dt/browctl/profile"
)

func TestManagerStartStopChrome(t *testing.T) {
	chromePath := os.Getenv("BROWCTL_CHROME")
	if chromePath == "" {
		t.Skip("set BROWCTL_CHROME to run Chrome integration test")
	}
	if _, err := os.Stat(chromePath); err != nil {
		t.Skipf("BROWCTL_CHROME is not usable: %v", err)
	}

	profiles := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := profiles.Create("work", chromePath); err != nil {
		t.Fatalf("Create profile: %v", err)
	}
	mgr := NewManager(profiles, filepath.Join(t.TempDir(), "run"))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rt, err := mgr.Start(ctx, "work")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if rt.PID <= 0 || rt.Port <= 0 || rt.Endpoint == "" || rt.WSEndpoint == "" || rt.State != StateRunning {
		t.Fatalf("runtime = %#v", rt)
	}
	if _, err := os.Stat(mgr.runtimePath("work")); err != nil {
		t.Fatalf("runtime file missing: %v", err)
	}
	if info, err := profiles.LockInfo("work"); err != nil || info.PID != rt.PID {
		t.Fatalf("lock info = %#v, err=%v; want pid %d", info, err, rt.PID)
	}

	stopped, err := mgr.Stop(ctx, "work")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.State != StateStopped {
		t.Fatalf("stopped.State = %s, want stopped", stopped.State)
	}
	if _, err := os.Stat(mgr.runtimePath("work")); !os.IsNotExist(err) {
		t.Fatalf("runtime file still exists or unexpected err: %v", err)
	}
	if _, err := profiles.LockInfo("work"); !os.IsNotExist(err) {
		t.Fatalf("lock file still exists or unexpected err: %v", err)
	}
}
