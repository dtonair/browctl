package action

import (
	"strings"

	"github.com/dtonair/browctl/driver"
	"github.com/dtonair/browctl/protocol"
)

func ParseWaitPolicy(raw string) (driver.WaitPolicy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return driver.WaitPolicy{Kind: "none"}, nil
	}
	if raw == "load" {
		return driver.WaitPolicy{Kind: "load"}, nil
	}
	kind, value, ok := strings.Cut(raw, ":")
	if !ok || strings.TrimSpace(value) == "" {
		return driver.WaitPolicy{}, protocol.NewError(protocol.InvalidRequest, "invalid wait policy", map[string]any{"wait": raw})
	}
	switch kind {
	case "url", "selector":
		return driver.WaitPolicy{Kind: kind, Value: strings.TrimSpace(value)}, nil
	default:
		return driver.WaitPolicy{}, protocol.NewError(protocol.InvalidRequest, "unsupported wait policy", map[string]any{"wait": raw})
	}
}
