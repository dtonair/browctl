package driver

import "context"

type SelectorKind string

const (
	SelectorCSS  SelectorKind = "css"
	SelectorText SelectorKind = "text"
)

type Selector struct {
	Kind  SelectorKind `json:"kind"`
	Value string       `json:"value"`
}

type WaitPolicy struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

type BrowserDriver interface {
	Navigate(ctx context.Context, url string, wait WaitPolicy) error
	Click(ctx context.Context, selector Selector, wait WaitPolicy) error
	Fill(ctx context.Context, selector Selector, value string) error
	Text(ctx context.Context, selector Selector) (string, error)
	WaitSelector(ctx context.Context, selector Selector) error
	WaitURL(ctx context.Context, pattern string) error
}
