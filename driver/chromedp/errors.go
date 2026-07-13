package chromedp

import (
	"fmt"

	"github.com/dtonair/browctl/driver"
)

type ErrNoSuchElement struct {
	Selector driver.Selector
}

func (e ErrNoSuchElement) Error() string {
	return fmt.Sprintf("no element matched %s=%q", e.Selector.Kind, e.Selector.Value)
}

type ErrAmbiguous struct {
	Selector driver.Selector
	Count    int
}

func (e ErrAmbiguous) Error() string {
	return fmt.Sprintf("selector %s=%q matched %d elements", e.Selector.Kind, e.Selector.Value, e.Count)
}
