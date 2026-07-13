package chromedp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	cdproto "github.com/chromedp/cdproto/cdp"
	cdp "github.com/chromedp/chromedp"
	"github.com/dtonair/browctl/driver"
)

type Driver struct{}

func New() Driver { return Driver{} }

func (Driver) Navigate(ctx context.Context, url string, wait driver.WaitPolicy) error {
	if url == "" {
		return fmt.Errorf("url is required")
	}
	if err := cdp.Run(ctx, cdp.Navigate(url)); err != nil {
		return err
	}
	return applyWait(ctx, wait)
}

func (Driver) Click(ctx context.Context, selector driver.Selector, wait driver.WaitPolicy) error {
	if err := ensureOne(ctx, selector); err != nil {
		return err
	}
	if err := cdp.Run(ctx, cdp.Click(selector.Value, queryOptions(selector)...)); err != nil {
		return err
	}
	return applyWait(ctx, wait)
}

func (Driver) Fill(ctx context.Context, selector driver.Selector, value string) error {
	if err := ensureOne(ctx, selector); err != nil {
		return err
	}
	if err := cdp.Run(ctx,
		cdp.Focus(selector.Value, queryOptions(selector)...),
		cdp.Clear(selector.Value, queryOptions(selector)...),
		cdp.SendKeys(selector.Value, value, queryOptions(selector)...),
	); err != nil {
		return err
	}
	return nil
}

func (Driver) Text(ctx context.Context, selector driver.Selector) (string, error) {
	if err := ensureOne(ctx, selector); err != nil {
		return "", err
	}
	var out string
	if err := cdp.Run(ctx, cdp.Text(selector.Value, &out, queryOptions(selector)...)); err != nil {
		return "", err
	}
	return out, nil
}

func (Driver) WaitSelector(ctx context.Context, selector driver.Selector) error {
	return cdp.Run(ctx, cdp.WaitVisible(selector.Value, queryOptions(selector)...))
}

func (Driver) WaitURL(ctx context.Context, pattern string) error {
	if pattern == "" {
		return fmt.Errorf("url pattern is required")
	}
	return retryUntil(ctx, 100*time.Millisecond, func(ctx context.Context) (bool, error) {
		var current string
		if err := cdp.Run(ctx, cdp.Location(&current)); err != nil {
			return false, err
		}
		return matchURL(pattern, current), nil
	})
}

func applyWait(ctx context.Context, wait driver.WaitPolicy) error {
	switch wait.Kind {
	case "", "none":
		return nil
	case "selector":
		return New().WaitSelector(ctx, driver.Selector{Kind: driver.SelectorCSS, Value: wait.Value})
	case "url":
		return New().WaitURL(ctx, wait.Value)
	case "load":
		return cdp.Run(ctx, cdp.WaitReady("body", cdp.ByQuery))
	default:
		return fmt.Errorf("unsupported wait policy: %s", wait.Kind)
	}
}

func ensureOne(ctx context.Context, selector driver.Selector) error {
	var count int
	if err := cdp.Run(ctx, countNodes(selector, &count)); err != nil {
		return err
	}
	if count == 0 {
		return ErrNoSuchElement{Selector: selector}
	}
	if count > 1 {
		return ErrAmbiguous{Selector: selector, Count: count}
	}
	return nil
}

func countNodes(selector driver.Selector, count *int) cdp.Action {
	return cdp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdproto.Node
		opts := queryOptions(selector)
		if selector.Kind == driver.SelectorCSS {
			opts = []cdp.QueryOption{cdp.ByQueryAll}
		}
		if err := cdp.Nodes(selector.Value, &nodes, opts...).Do(ctx); err != nil {
			return err
		}
		*count = len(nodes)
		return nil
	})
}

func queryOptions(selector driver.Selector) []cdp.QueryOption {
	switch selector.Kind {
	case driver.SelectorCSS:
		return []cdp.QueryOption{cdp.ByQuery}
	case driver.SelectorText:
		return []cdp.QueryOption{cdp.BySearch}
	default:
		return []cdp.QueryOption{cdp.ByQuery}
	}
}

func retryUntil(ctx context.Context, interval time.Duration, fn func(context.Context) (bool, error)) error {
	for {
		ok, err := fn(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func matchURL(pattern, current string) bool {
	if pattern == current {
		return true
	}
	if strings.Contains(pattern, "*") {
		matched, err := filepath.Match(pattern, current)
		if err == nil && matched {
			return true
		}
		if strings.HasPrefix(pattern, "**") {
			return strings.HasSuffix(current, strings.TrimPrefix(pattern, "**"))
		}
	}
	return strings.Contains(current, pattern)
}
