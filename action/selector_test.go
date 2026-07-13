package action

import (
	"testing"

	"github.com/dtonair/browctl/driver"
	"github.com/dtonair/browctl/protocol"
)

func TestParseSelector(t *testing.T) {
	tests := []struct {
		raw  string
		kind driver.SelectorKind
		val  string
	}{
		{raw: "css=button.submit", kind: driver.SelectorCSS, val: "button.submit"},
		{raw: "text=Sign in", kind: driver.SelectorText, val: "Sign in"},
		{raw: "button.submit", kind: driver.SelectorCSS, val: "button.submit"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseSelector(tt.raw)
			if err != nil {
				t.Fatalf("ParseSelector() error = %v", err)
			}
			if got.Kind != tt.kind || got.Value != tt.val {
				t.Fatalf("got = %#v, want %s/%q", got, tt.kind, tt.val)
			}
		})
	}
}

func TestParseSelectorRejectsUnsupported(t *testing.T) {
	_, err := ParseSelector("xpath=//button")
	if err == nil {
		t.Fatal("ParseSelector() error = nil, want INVALID_REQUEST")
	}
	if perr := err.(*protocol.Error); perr.Code != protocol.InvalidRequest {
		t.Fatalf("code = %s, want INVALID_REQUEST", perr.Code)
	}
}

func TestParseWaitPolicy(t *testing.T) {
	tests := []string{"", "none", "load", "url:**/done", "selector:css=.ready"}
	for _, raw := range tests {
		if _, err := ParseWaitPolicy(raw); err != nil {
			t.Fatalf("ParseWaitPolicy(%q) error = %v", raw, err)
		}
	}
}
