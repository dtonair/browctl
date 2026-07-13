package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/dt/browctl/profile"
	"github.com/dt/browctl/protocol"
)

func TestProfileHandlers(t *testing.T) {
	mgr := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	d := NewWithProfileManager(filepath.Join(t.TempDir(), "daemon.sock"), mgr)

	args, _ := json.Marshal(map[string]string{"chrome_path": "/chrome"})
	created := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.create", Profile: "work", Args: args})
	if !created.OK {
		t.Fatalf("create.OK = false: %#v", created.Error)
	}

	listed := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.list"})
	if !listed.OK {
		t.Fatalf("list.OK = false: %#v", listed.Error)
	}

	inspected := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.inspect", Profile: "work"})
	if !inspected.OK {
		t.Fatalf("inspect.OK = false: %#v", inspected.Error)
	}

	deleted := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.delete", Profile: "work"})
	if !deleted.OK {
		t.Fatalf("delete.OK = false: %#v", deleted.Error)
	}

	missing := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.inspect", Profile: "work"})
	if missing.OK || missing.Error == nil || missing.Error.Code != protocol.ProfileNotFound {
		t.Fatalf("missing = %#v, want PROFILE_NOT_FOUND", missing)
	}
}

func TestProfileCreateRequiresName(t *testing.T) {
	mgr := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	d := NewWithProfileManager(filepath.Join(t.TempDir(), "daemon.sock"), mgr)
	resp := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "profile.create"})
	if resp.OK || resp.Error == nil || resp.Error.Code != protocol.InvalidRequest {
		t.Fatalf("resp = %#v, want INVALID_REQUEST", resp)
	}
}
