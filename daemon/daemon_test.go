package daemon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dtonair/browctl/protocol"
)

func TestHandlePingStatusAndUnknownCommand(t *testing.T) {
	d := New(filepath.Join(t.TempDir(), "daemon.sock"))

	ping := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "ping"})
	if !ping.OK {
		t.Fatalf("ping.OK = false: %#v", ping.Error)
	}

	status := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "daemon.status"})
	if !status.OK {
		t.Fatalf("status.OK = false: %#v", status.Error)
	}

	unknown := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "bogus"})
	if unknown.OK || unknown.Error == nil || unknown.Error.Code != protocol.InvalidRequest {
		t.Fatalf("unknown = %#v, want INVALID_REQUEST", unknown)
	}
}

func TestHandleRejectsUnsupportedAPIVersion(t *testing.T) {
	d := New(filepath.Join(t.TempDir(), "daemon.sock"))
	resp := d.Handle(context.Background(), protocol.Request{APIVersion: 999, Cmd: "ping"})
	if resp.OK || resp.Error == nil || resp.Error.Code != protocol.InvalidRequest {
		t.Fatalf("resp = %#v, want INVALID_REQUEST", resp)
	}
}
