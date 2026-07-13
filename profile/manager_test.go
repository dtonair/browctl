package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dtonair/browctl/protocol"
)

func TestCreateInspectListDelete(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "profiles"))

	p, err := m.Create("work", "/path/to/chrome")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p.Name != "work" || p.ChromePath != "/path/to/chrome" || p.CreatedAt.IsZero() {
		t.Fatalf("profile = %#v", p)
	}

	for _, path := range []string{
		m.ProfileDir("work"),
		m.ChromeDataDir("work"),
		filepath.Join(m.ProfileDir("work"), ProfileFileName),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	inspected, err := m.Inspect("work")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if inspected.Name != "work" || inspected.ChromePath != "/path/to/chrome" {
		t.Fatalf("inspected = %#v", inspected)
	}

	profiles, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "work" {
		t.Fatalf("profiles = %#v", profiles)
	}

	if err := m.Delete("work"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := m.Inspect("work"); err == nil || err.(*protocol.Error).Code != protocol.ProfileNotFound {
		t.Fatalf("Inspect deleted err = %v, want PROFILE_NOT_FOUND", err)
	}
}

func TestCreateRejectsInvalidNameAndDuplicate(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := m.Create("../bad", ""); err == nil || err.(*protocol.Error).Code != protocol.InvalidRequest {
		t.Fatalf("Create invalid err = %v, want INVALID_REQUEST", err)
	}
	if _, err := m.Create("work", ""); err != nil {
		t.Fatalf("Create work error = %v", err)
	}
	if _, err := m.Create("work", ""); err == nil || err.(*protocol.Error).Code != protocol.InvalidRequest {
		t.Fatalf("Create duplicate err = %v, want INVALID_REQUEST", err)
	}
}

func TestAcquireRejectsSecondWriterAndReleaseClearsLock(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := m.Create("work", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	lock, err := m.Acquire("work", LockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(m.ProfileDir("work"), LockFileName)); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}

	_, err = m.Acquire("work", LockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("second Acquire() error = nil, want PROFILE_LOCKED")
	}
	perr, ok := err.(*protocol.Error)
	if !ok || perr.Code != protocol.ProfileLocked {
		t.Fatalf("second Acquire() err = %#v, want PROFILE_LOCKED", err)
	}

	if err := m.Release(lock); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.ProfileDir("work"), LockFileName)); !os.IsNotExist(err) {
		t.Fatalf("lock file still exists or unexpected err: %v", err)
	}
}

func TestAcquireRecoversStaleLock(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := m.Create("work", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stale := LockInfo{PID: 99999999, StartedAt: time.Now().Add(-time.Hour).UTC()}
	lockPath := filepath.Join(m.ProfileDir("work"), LockFileName)
	if err := os.WriteFile(lockPath, []byte(`{"pid":99999999,"started_at":"`+stale.StartedAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	lock, err := m.Acquire("work", LockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if lock.Info().PID != os.Getpid() {
		t.Fatalf("lock PID = %d, want %d", lock.Info().PID, os.Getpid())
	}
}

func TestDeleteLockedProfileFails(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := m.Create("work", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	lock, err := m.Acquire("work", LockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer m.Release(lock)

	err = m.Delete("work")
	if err == nil {
		t.Fatal("Delete locked error = nil, want PROFILE_LOCKED")
	}
	if perr := err.(*protocol.Error); perr.Code != protocol.ProfileLocked {
		t.Fatalf("Delete locked code = %s, want PROFILE_LOCKED", perr.Code)
	}
}
