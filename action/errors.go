package action

import (
	"context"
	"errors"
	"strings"

	cdp "github.com/chromedp/chromedp"
	chromedriver "github.com/dt/browctl/driver/chromedp"
	"github.com/dt/browctl/protocol"
)

func TranslateError(err error) *protocol.Error {
	if err == nil {
		return nil
	}
	var perr *protocol.Error
	if errors.As(err, &perr) {
		return perr
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return protocol.NewError(protocol.ActionTimeout, "action timed out", nil)
	}
	var noSuch chromedriver.ErrNoSuchElement
	if errors.As(err, &noSuch) {
		return protocol.NewError(protocol.ElementNotFound, noSuch.Error(), map[string]any{"selector": noSuch.Selector})
	}
	var ambiguous chromedriver.ErrAmbiguous
	if errors.As(err, &ambiguous) {
		return protocol.NewError(protocol.ElementAmbiguous, ambiguous.Error(), map[string]any{"selector": ambiguous.Selector, "count": ambiguous.Count})
	}
	if errors.Is(err, cdp.ErrNoResults) {
		return protocol.NewError(protocol.ElementNotFound, err.Error(), nil)
	}
	if errors.Is(err, cdp.ErrNotVisible) {
		return protocol.NewError(protocol.ElementNotVisible, err.Error(), nil)
	}
	if errors.Is(err, cdp.ErrDisabled) || errors.Is(err, cdp.ErrInvalidBoxModel) {
		return protocol.NewError(protocol.ElementNotInteractable, err.Error(), nil)
	}
	if errors.Is(err, cdp.ErrPollingTimeout) {
		return protocol.NewError(protocol.ActionTimeout, err.Error(), nil)
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "context deadline exceeded"), strings.Contains(msg, "timeout"):
		return protocol.NewError(protocol.ActionTimeout, err.Error(), nil)
	case strings.Contains(msg, "no node") || strings.Contains(msg, "could not find"):
		return protocol.NewError(protocol.ElementNotFound, err.Error(), nil)
	case strings.Contains(msg, "not visible"):
		return protocol.NewError(protocol.ElementNotVisible, err.Error(), nil)
	case strings.Contains(msg, "target closed") || strings.Contains(msg, "target detached"):
		return protocol.NewError(protocol.TargetDetached, err.Error(), nil)
	default:
		return protocol.NewError(protocol.InternalError, err.Error(), nil)
	}
}
