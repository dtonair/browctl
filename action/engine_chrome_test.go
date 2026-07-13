package action

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dt/browctl/browser"
	chromedriver "github.com/dt/browctl/driver/chromedp"
	"github.com/dt/browctl/profile"
	"github.com/dt/browctl/protocol"
	"github.com/dt/browctl/tab"
	"github.com/dt/browctl/testserver"
)

func TestEngineChromeWorkflow(t *testing.T) {
	chromePath := os.Getenv("BROWCTL_CHROME")
	if chromePath == "" {
		t.Skip("set BROWCTL_CHROME to run Chrome integration test")
	}
	if _, err := os.Stat(chromePath); err != nil {
		t.Skipf("BROWCTL_CHROME is not usable: %v", err)
	}

	srv := testserver.New()
	defer srv.Close()

	profiles := profile.NewManager(filepath.Join(t.TempDir(), "profiles"))
	if _, err := profiles.Create("work", chromePath); err != nil {
		t.Fatalf("Create profile: %v", err)
	}
	browsers := browser.NewManager(profiles, filepath.Join(t.TempDir(), "run"))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := browsers.Start(ctx, "work"); err != nil {
		t.Fatalf("Start browser: %v", err)
	}
	defer browsers.Stop(context.Background(), "work")

	registry := tab.NewRegistry(browsers)
	openedTab, err := registry.Open(ctx, "work", srv.URL)
	if err != nil {
		t.Fatalf("Open tab: %v", err)
	}
	engine := NewEngine(registry, chromedriver.New()).WithTimeout(10 * time.Second)

	if _, err := engine.Goto(ctx, "work", openedTab.ID, GotoArgs{URL: srv.URL}); err != nil {
		t.Fatalf("Goto() error = %v", err)
	}
	if _, err := engine.Fill(ctx, "work", openedTab.ID, FillArgs{Selector: `css=input[name=email]`, Value: "a@example.com"}); err != nil {
		t.Fatalf("Fill() error = %v", err)
	}
	if _, err := engine.Click(ctx, "work", openedTab.ID, SelectorArgs{Selector: "css=#submit", Wait: "url:**/dashboard"}); err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	text, err := engine.Text(ctx, "work", openedTab.ID, SelectorArgs{Selector: "css=.account"})
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if text["text"] != "Account OK" {
		t.Fatalf("text = %#v, want Account OK", text)
	}

	if _, err := engine.Goto(ctx, "work", openedTab.ID, GotoArgs{URL: srv.URL + "/dupes"}); err != nil {
		t.Fatalf("Goto dupes error = %v", err)
	}
	_, err = engine.Click(ctx, "work", openedTab.ID, SelectorArgs{Selector: "css=button"})
	if err == nil {
		t.Fatal("ambiguous click error = nil")
	}
	if perr := TranslateError(err); perr.Code != protocol.ElementAmbiguous {
		t.Fatalf("ambiguous code = %s, want ELEMENT_AMBIGUOUS", perr.Code)
	}
}
