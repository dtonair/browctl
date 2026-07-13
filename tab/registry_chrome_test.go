package tab

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dt/browctl/browser"
	"github.com/dt/browctl/profile"
)

func TestRegistryOpenListFocusCloseChrome(t *testing.T) {
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
	browsers := browser.NewManager(profiles, filepath.Join(t.TempDir(), "run"))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if _, err := browsers.Start(ctx, "work"); err != nil {
		t.Fatalf("Start browser: %v", err)
	}
	defer browsers.Stop(context.Background(), "work")

	registry := NewRegistry(browsers)
	opened, err := registry.Open(ctx, "work", "about:blank")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if opened.ID == "" || opened.TargetID == "" {
		t.Fatalf("opened = %#v", opened)
	}

	tabs, err := registry.List(ctx, "work")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tabs) == 0 {
		t.Fatal("List() returned no tabs")
	}

	focused, err := registry.Focus(ctx, "work", opened.ID)
	if err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	if focused.ID != opened.ID || !focused.Active {
		t.Fatalf("focused = %#v, want active %s", focused, opened.ID)
	}

	closed, err := registry.Close(ctx, "work", opened.ID)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.ID != opened.ID {
		t.Fatalf("closed.ID = %s, want %s", closed.ID, opened.ID)
	}
}
