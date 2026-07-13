package action

import (
	"context"
	"errors"
	"testing"

	"github.com/dt/browctl/driver"
	chromedriver "github.com/dt/browctl/driver/chromedp"
	"github.com/dt/browctl/protocol"
	"github.com/dt/browctl/tab"
)

type fakeRunner struct {
	calledProfile string
	calledTab     string
}

func (r *fakeRunner) WithTab(ctx context.Context, profileName, tabID string, fn func(context.Context, tab.Tab) error) (tab.Tab, error) {
	r.calledProfile = profileName
	r.calledTab = tabID
	t := tab.Tab{ID: tabID, Profile: profileName, TargetID: "target-1"}
	return t, fn(ctx, t)
}

type fakeDriver struct {
	err       error
	text      string
	lastSel   driver.Selector
	lastWait  driver.WaitPolicy
	lastURL   string
	lastValue string
}

func (d *fakeDriver) Navigate(ctx context.Context, url string, wait driver.WaitPolicy) error {
	d.lastURL, d.lastWait = url, wait
	return d.err
}
func (d *fakeDriver) Click(ctx context.Context, selector driver.Selector, wait driver.WaitPolicy) error {
	d.lastSel, d.lastWait = selector, wait
	return d.err
}
func (d *fakeDriver) Fill(ctx context.Context, selector driver.Selector, value string) error {
	d.lastSel, d.lastValue = selector, value
	return d.err
}
func (d *fakeDriver) Text(ctx context.Context, selector driver.Selector) (string, error) {
	d.lastSel = selector
	return d.text, d.err
}
func (d *fakeDriver) WaitSelector(ctx context.Context, selector driver.Selector) error {
	d.lastSel = selector
	return d.err
}
func (d *fakeDriver) WaitURL(ctx context.Context, pattern string) error {
	d.lastURL = pattern
	return d.err
}

func TestEngineGotoDefaultsActiveTab(t *testing.T) {
	runner := &fakeRunner{}
	drv := &fakeDriver{}
	engine := NewEngine(runner, drv)

	data, err := engine.Goto(context.Background(), "work", "", GotoArgs{URL: "https://example.com", Wait: "url:**/done"})
	if err != nil {
		t.Fatalf("Goto() error = %v", err)
	}
	if runner.calledTab != "active" || runner.calledProfile != "work" {
		t.Fatalf("runner called profile/tab = %s/%s", runner.calledProfile, runner.calledTab)
	}
	if drv.lastURL != "https://example.com" || drv.lastWait.Kind != "url" || drv.lastWait.Value != "**/done" {
		t.Fatalf("driver state = url %q wait %#v", drv.lastURL, drv.lastWait)
	}
	if data["url"] != "https://example.com" {
		t.Fatalf("data = %#v", data)
	}
}

func TestEngineFillParsesSelectorAndValue(t *testing.T) {
	runner := &fakeRunner{}
	drv := &fakeDriver{}
	engine := NewEngine(runner, drv)

	_, err := engine.Fill(context.Background(), "work", "tab_1", FillArgs{Selector: "css=input[name=email]", Value: "a@example.com"})
	if err != nil {
		t.Fatalf("Fill() error = %v", err)
	}
	if drv.lastSel.Kind != driver.SelectorCSS || drv.lastSel.Value != "input[name=email]" || drv.lastValue != "a@example.com" {
		t.Fatalf("driver state = selector %#v value %q", drv.lastSel, drv.lastValue)
	}
}

func TestEngineTextReturnsText(t *testing.T) {
	runner := &fakeRunner{}
	drv := &fakeDriver{text: "Account"}
	engine := NewEngine(runner, drv)
	data, err := engine.Text(context.Background(), "work", "tab_1", SelectorArgs{Selector: "text=Account"})
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if data["text"] != "Account" || drv.lastSel.Kind != driver.SelectorText {
		t.Fatalf("data=%#v selector=%#v", data, drv.lastSel)
	}
}

func TestEngineTranslatesDriverErrors(t *testing.T) {
	runner := &fakeRunner{}
	drv := &fakeDriver{err: chromedriver.ErrAmbiguous{Selector: driver.Selector{Kind: driver.SelectorCSS, Value: "button"}, Count: 2}}
	engine := NewEngine(runner, drv)

	_, err := engine.Click(context.Background(), "work", "tab_1", SelectorArgs{Selector: "css=button"})
	if err == nil {
		t.Fatal("Click() error = nil, want ELEMENT_AMBIGUOUS")
	}
	var perr *protocol.Error
	if !errors.As(err, &perr) || perr.Code != protocol.ElementAmbiguous {
		t.Fatalf("error = %#v, want ELEMENT_AMBIGUOUS", err)
	}
}
