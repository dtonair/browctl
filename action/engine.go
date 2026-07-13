package action

import (
	"context"
	"time"

	"github.com/dt/browctl/driver"
	"github.com/dt/browctl/protocol"
	"github.com/dt/browctl/tab"
)

type TabRunner interface {
	WithTab(ctx context.Context, profileName, tabID string, fn func(context.Context, tab.Tab) error) (tab.Tab, error)
}

type Engine struct {
	tabs    TabRunner
	driver  driver.BrowserDriver
	timeout time.Duration
}

func NewEngine(tabs TabRunner, driver driver.BrowserDriver) *Engine {
	return &Engine{tabs: tabs, driver: driver, timeout: 15 * time.Second}
}

func (e *Engine) WithTimeout(timeout time.Duration) *Engine {
	if timeout > 0 {
		e.timeout = timeout
	}
	return e
}

type GotoArgs struct {
	URL  string `json:"url"`
	Wait string `json:"wait,omitempty"`
}

type SelectorArgs struct {
	Selector string `json:"selector"`
	Wait     string `json:"wait,omitempty"`
}

type FillArgs struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

type WaitURLArgs struct {
	Pattern string `json:"pattern"`
}

func (e *Engine) Goto(ctx context.Context, profileName, tabID string, args GotoArgs) (map[string]any, error) {
	wait, err := ParseWaitPolicy(args.Wait)
	if err != nil {
		return nil, err
	}
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		if err := e.driver.Navigate(ctx, args.URL, wait); err != nil {
			return nil, err
		}
		return map[string]any{"url": args.URL}, nil
	})
}

func (e *Engine) Click(ctx context.Context, profileName, tabID string, args SelectorArgs) (map[string]any, error) {
	sel, err := ParseSelector(args.Selector)
	if err != nil {
		return nil, err
	}
	wait, err := ParseWaitPolicy(args.Wait)
	if err != nil {
		return nil, err
	}
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		if err := e.driver.Click(ctx, sel, wait); err != nil {
			return nil, err
		}
		return map[string]any{"selector": args.Selector}, nil
	})
}

func (e *Engine) Fill(ctx context.Context, profileName, tabID string, args FillArgs) (map[string]any, error) {
	sel, err := ParseSelector(args.Selector)
	if err != nil {
		return nil, err
	}
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		if err := e.driver.Fill(ctx, sel, args.Value); err != nil {
			return nil, err
		}
		return map[string]any{"selector": args.Selector}, nil
	})
}

func (e *Engine) Text(ctx context.Context, profileName, tabID string, args SelectorArgs) (map[string]any, error) {
	sel, err := ParseSelector(args.Selector)
	if err != nil {
		return nil, err
	}
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		text, err := e.driver.Text(ctx, sel)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": text, "selector": args.Selector}, nil
	})
}

func (e *Engine) WaitSelector(ctx context.Context, profileName, tabID string, args SelectorArgs) (map[string]any, error) {
	sel, err := ParseSelector(args.Selector)
	if err != nil {
		return nil, err
	}
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		if err := e.driver.WaitSelector(ctx, sel); err != nil {
			return nil, err
		}
		return map[string]any{"selector": args.Selector}, nil
	})
}

func (e *Engine) WaitURL(ctx context.Context, profileName, tabID string, args WaitURLArgs) (map[string]any, error) {
	return e.run(ctx, profileName, tabID, func(ctx context.Context) (map[string]any, error) {
		if err := e.driver.WaitURL(ctx, args.Pattern); err != nil {
			return nil, err
		}
		return map[string]any{"pattern": args.Pattern}, nil
	})
}

func (e *Engine) run(ctx context.Context, profileName, tabID string, fn func(context.Context) (map[string]any, error)) (map[string]any, error) {
	if profileName == "" {
		return nil, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	if tabID == "" {
		tabID = "active"
	}
	runCtx := ctx
	cancel := func() {}
	if e.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, e.timeout)
	}
	defer cancel()

	var out map[string]any
	_, err := e.tabs.WithTab(runCtx, profileName, tabID, func(targetCtx context.Context, _ tab.Tab) error {
		var actionErr error
		out, actionErr = fn(targetCtx)
		return actionErr
	})
	if err != nil {
		return nil, TranslateError(err)
	}
	return out, nil
}
