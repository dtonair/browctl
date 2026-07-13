package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesXDGRuntimeDirWhenPresent(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
	xdg := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("XDG_RUNTIME_DIR", xdg)

	layout, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantBase := filepath.Join(os.Getenv("HOME"), appDirName)
	if layout.BaseDir != wantBase {
		t.Fatalf("BaseDir = %q, want %q", layout.BaseDir, wantBase)
	}

	wantRuntime := filepath.Join(xdg, "browctl")
	if layout.RuntimeDir != wantRuntime {
		t.Fatalf("RuntimeDir = %q, want %q", layout.RuntimeDir, wantRuntime)
	}

	wantSocket := filepath.Join(wantRuntime, daemonSock)
	if layout.DaemonSocket != wantSocket {
		t.Fatalf("DaemonSocket = %q, want %q", layout.DaemonSocket, wantSocket)
	}
}

func TestResolveFallsBackToHomeRuntimeDir(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", "")

	layout, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantRuntime := filepath.Join(home, appDirName, runtimeSubdir)
	if layout.RuntimeDir != wantRuntime {
		t.Fatalf("RuntimeDir = %q, want %q", layout.RuntimeDir, wantRuntime)
	}
}

func TestEnsureCreatesOwnerOnlyDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", "")

	layout, err := Ensure()
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	for _, dir := range []string{layout.BaseDir, layout.RuntimeDir, layout.ProfilesDir, layout.ArtifactsDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("%s permissions = %o, want 700", dir, got)
		}
	}
}
