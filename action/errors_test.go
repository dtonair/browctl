package action

import (
	"context"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/dtonair/browctl/driver"
	chromedriver "github.com/dtonair/browctl/driver/chromedp"
	"github.com/dtonair/browctl/protocol"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want protocol.ErrorCode
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: protocol.ActionTimeout},
		{name: "no such", err: chromedriver.ErrNoSuchElement{Selector: driver.Selector{Kind: driver.SelectorCSS, Value: ".missing"}}, want: protocol.ElementNotFound},
		{name: "ambiguous", err: chromedriver.ErrAmbiguous{Selector: driver.Selector{Kind: driver.SelectorCSS, Value: "button"}, Count: 2}, want: protocol.ElementAmbiguous},
		{name: "chromedp no results", err: chromedp.ErrNoResults, want: protocol.ElementNotFound},
		{name: "chromedp not visible", err: chromedp.ErrNotVisible, want: protocol.ElementNotVisible},
		{name: "chromedp polling", err: chromedp.ErrPollingTimeout, want: protocol.ActionTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateError(tt.err)
			if got.Code != tt.want {
				t.Fatalf("code = %s, want %s", got.Code, tt.want)
			}
		})
	}
}
