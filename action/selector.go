package action

import (
	"strings"

	"github.com/dt/browctl/driver"
	"github.com/dt/browctl/protocol"
)

func ParseSelector(raw string) (driver.Selector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return driver.Selector{}, protocol.NewError(protocol.InvalidRequest, "selector is required", nil)
	}
	kind, value, ok := strings.Cut(raw, "=")
	if !ok {
		return driver.Selector{Kind: driver.SelectorCSS, Value: raw}, nil
	}
	kind = strings.TrimSpace(kind)
	value = strings.TrimSpace(value)
	if value == "" {
		return driver.Selector{}, protocol.NewError(protocol.InvalidRequest, "selector value is required", map[string]any{"selector": raw})
	}
	switch kind {
	case "css":
		return driver.Selector{Kind: driver.SelectorCSS, Value: value}, nil
	case "text":
		return driver.Selector{Kind: driver.SelectorText, Value: value}, nil
	default:
		return driver.Selector{}, protocol.NewError(protocol.InvalidRequest, "unsupported selector type", map[string]any{"selector": raw, "type": kind})
	}
}
