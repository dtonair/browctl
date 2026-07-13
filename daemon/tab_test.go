package daemon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dtonair/browctl/profile"
	"github.com/dtonair/browctl/protocol"
	"github.com/dtonair/browctl/tab"
)

type daemonFakeTabProvider struct{}

func (daemonFakeTabProvider) BrowserContext(profile string) (context.Context, error) {
	return context.Background(), nil
}

func TestTabHandlersValidateProfileAndTab(t *testing.T) {
	mgr := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	tabs := tab.NewRegistry(daemonFakeTabProvider{})
	d := NewWithTabRegistry(filepath.Join(t.TempDir(), "daemon.sock"), mgr, nil, tabs)

	missingProfile := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "tab.list"})
	if missingProfile.OK || missingProfile.Error == nil || missingProfile.Error.Code != protocol.InvalidRequest {
		t.Fatalf("missing profile = %#v, want INVALID_REQUEST", missingProfile)
	}

	missingTab := d.Handle(context.Background(), protocol.Request{APIVersion: protocol.APIVersion, Cmd: "tab.focus", Profile: "work", Tab: "missing"})
	if missingTab.OK || missingTab.Error == nil || missingTab.Error.Code != protocol.TabNotFound {
		t.Fatalf("missing tab = %#v, want TAB_NOT_FOUND", missingTab)
	}
}
