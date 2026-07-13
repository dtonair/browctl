package tab

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/dt/browctl/protocol"
)

type BrowserContextProvider interface {
	BrowserContext(profile string) (context.Context, error)
}

type Tab struct {
	ID       string `json:"id"`
	Profile  string `json:"profile"`
	TargetID string `json:"target_id"`
	URL      string `json:"url,omitempty"`
	Title    string `json:"title,omitempty"`
	Active   bool   `json:"active,omitempty"`
}

type entry struct {
	tab Tab
	mu  sync.Mutex
}

type Registry struct {
	provider BrowserContextProvider

	mu      sync.Mutex
	byID    map[string]*entry
	byCDP   map[string]string // profile + "\x00" + targetID -> stable tab ID
	active  map[string]string // profile -> stable tab ID
	idMaker func() string
}

func NewRegistry(provider BrowserContextProvider) *Registry {
	return &Registry{
		provider: provider,
		byID:     map[string]*entry{},
		byCDP:    map[string]string{},
		active:   map[string]string{},
		idMaker:  newID,
	}
}

func (r *Registry) Open(ctx context.Context, profileName, url string) (Tab, error) {
	if profileName == "" {
		return Tab{}, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	if url == "" {
		return Tab{}, protocol.NewError(protocol.InvalidRequest, "url is required", nil)
	}
	browserCtx, err := r.provider.BrowserContext(profileName)
	if err != nil {
		return Tab{}, err
	}
	targetID, err := target.CreateTarget(url).Do(browserCtx)
	if err != nil {
		return Tab{}, protocol.NewError(protocol.NavigationFailed, "open tab: "+err.Error(), map[string]any{"url": url})
	}
	tab := r.register(profileName, string(targetID), url, "")
	r.setActive(profileName, tab.ID)
	return tab, nil
}

func (r *Registry) List(ctx context.Context, profileName string) ([]Tab, error) {
	if profileName == "" {
		return nil, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	browserCtx, err := r.provider.BrowserContext(profileName)
	if err != nil {
		return nil, err
	}
	infos, err := chromedp.Targets(browserCtx)
	if err != nil {
		return nil, protocol.NewError(protocol.TargetDetached, "list tabs: "+err.Error(), nil)
	}
	for _, info := range infos {
		if info == nil || info.Type != "page" {
			continue
		}
		r.register(profileName, string(info.TargetID), info.URL, info.Title)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	activeID := r.active[profileName]
	out := []Tab{}
	for _, e := range r.byID {
		if e.tab.Profile != profileName {
			continue
		}
		tab := e.tab
		tab.Active = tab.ID == activeID
		out = append(out, tab)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *Registry) Focus(ctx context.Context, profileName, tabID string) (Tab, error) {
	tab, err := r.WithTab(ctx, profileName, tabID, func(targetCtx context.Context, tab Tab) error {
		if err := page.BringToFront().Do(targetCtx); err != nil {
			return protocol.NewError(protocol.TargetDetached, "focus tab: "+err.Error(), map[string]any{"tab": tab.ID})
		}
		return nil
	})
	if err != nil {
		return Tab{}, err
	}
	r.setActive(profileName, tab.ID)
	tab.Active = true
	return tab, nil
}

func (r *Registry) Close(ctx context.Context, profileName, tabID string) (Tab, error) {
	tab, err := r.lookup(profileName, tabID)
	if err != nil {
		return Tab{}, err
	}
	browserCtx, err := r.provider.BrowserContext(profileName)
	if err != nil {
		return Tab{}, err
	}
	if err := target.CloseTarget(target.ID(tab.TargetID)).Do(browserCtx); err != nil {
		return Tab{}, protocol.NewError(protocol.TargetDetached, "close tab: "+err.Error(), map[string]any{"tab": tab.ID})
	}
	r.remove(profileName, tab.ID, tab.TargetID)
	return tab, nil
}

func (r *Registry) WithTab(ctx context.Context, profileName, tabID string, fn func(context.Context, Tab) error) (Tab, error) {
	if fn == nil {
		return Tab{}, protocol.NewError(protocol.InvalidRequest, "tab function is required", nil)
	}
	e, err := r.lookupEntry(profileName, tabID)
	if err != nil {
		return Tab{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	browserCtx, err := r.provider.BrowserContext(profileName)
	if err != nil {
		return Tab{}, err
	}
	targetCtx, cancel := chromedp.NewContext(browserCtx, chromedp.WithTargetID(target.ID(e.tab.TargetID)))
	defer cancel()
	if ctx != nil {
		var ctxCancel context.CancelFunc
		targetCtx, ctxCancel = context.WithCancel(targetCtx)
		go func() {
			select {
			case <-ctx.Done():
				ctxCancel()
			case <-targetCtx.Done():
			}
		}()
		defer ctxCancel()
	}
	if err := fn(targetCtx, e.tab); err != nil {
		return Tab{}, err
	}
	return e.tab, nil
}

func (r *Registry) Register(profileName, targetID, url, title string) Tab {
	return r.register(profileName, targetID, url, title)
}

func (r *Registry) register(profileName, targetID, url, title string) Tab {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cdpKey(profileName, targetID)
	if id := r.byCDP[key]; id != "" {
		e := r.byID[id]
		e.tab.URL = url
		e.tab.Title = title
		return e.tab
	}
	id := "tab_" + r.idMaker()
	tab := Tab{ID: id, Profile: profileName, TargetID: targetID, URL: url, Title: title}
	r.byID[id] = &entry{tab: tab}
	r.byCDP[key] = id
	if r.active[profileName] == "" {
		r.active[profileName] = id
	}
	return tab
}

func (r *Registry) lookup(profileName, tabID string) (Tab, error) {
	e, err := r.lookupEntry(profileName, tabID)
	if err != nil {
		return Tab{}, err
	}
	return e.tab, nil
}

func (r *Registry) lookupEntry(profileName, tabID string) (*entry, error) {
	if profileName == "" {
		return nil, protocol.NewError(protocol.InvalidRequest, "profile is required", nil)
	}
	if tabID == "" || tabID == "active" {
		r.mu.Lock()
		tabID = r.active[profileName]
		r.mu.Unlock()
	}
	if tabID == "" {
		return nil, protocol.NewError(protocol.TabNotFound, "no active tab", map[string]any{"profile": profileName})
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.byID[tabID]
	if e == nil || e.tab.Profile != profileName {
		return nil, protocol.NewError(protocol.TabNotFound, "tab not found", map[string]any{"profile": profileName, "tab": tabID})
	}
	return e, nil
}

func (r *Registry) setActive(profileName, tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[profileName] = tabID
}

func (r *Registry) remove(profileName, tabID, targetID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, tabID)
	delete(r.byCDP, cdpKey(profileName, targetID))
	if r.active[profileName] == tabID {
		delete(r.active, profileName)
		for _, e := range r.byID {
			if e.tab.Profile == profileName {
				r.active[profileName] = e.tab.ID
				break
			}
		}
	}
}

func cdpKey(profileName, targetID string) string {
	return profileName + "\x00" + targetID
}

func newID() string {
	var random [10]byte
	_, _ = rand.Read(random[:])
	var timestamp [6]byte
	ms := uint64(time.Now().UnixMilli())
	for i := 5; i >= 0; i-- {
		timestamp[i] = byte(ms)
		ms >>= 8
	}
	buf := append(timestamp[:], random[:]...)
	enc := base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)
	return strings.TrimRight(enc.EncodeToString(buf), "=")
}

func (t Tab) String() string {
	return fmt.Sprintf("%s %s %s", t.ID, t.TargetID, t.URL)
}
