package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dt/browctl/action"
	"github.com/dt/browctl/browser"
	chromedriver "github.com/dt/browctl/driver/chromedp"
	"github.com/dt/browctl/ipc"
	"github.com/dt/browctl/paths"
	"github.com/dt/browctl/profile"
	"github.com/dt/browctl/protocol"
	"github.com/dt/browctl/tab"
	"github.com/dt/browctl/version"
)

type Daemon struct {
	socketPath string

	mu       sync.Mutex
	cancel   context.CancelFunc
	profiles *profile.Manager
	browsers *browser.Manager
	tabs     *tab.Registry
	actions  *action.Engine
}

func New(socketPath string) *Daemon {
	return &Daemon{socketPath: socketPath}
}

func NewWithProfileManager(socketPath string, profiles *profile.Manager) *Daemon {
	return &Daemon{socketPath: socketPath, profiles: profiles}
}

func NewWithManagers(socketPath string, profiles *profile.Manager, browsers *browser.Manager) *Daemon {
	return &Daemon{socketPath: socketPath, profiles: profiles, browsers: browsers}
}

func NewWithTabRegistry(socketPath string, profiles *profile.Manager, browsers *browser.Manager, tabs *tab.Registry) *Daemon {
	return &Daemon{socketPath: socketPath, profiles: profiles, browsers: browsers, tabs: tabs}
}

func (d *Daemon) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.mu.Unlock()
	defer cancel()

	server := ipc.NewServer(d.socketPath, d.Handle)
	return server.Serve(ctx)
}

func (d *Daemon) Handle(ctx context.Context, req protocol.Request) protocol.Response {
	if req.APIVersion != 0 && req.APIVersion != protocol.APIVersion {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, fmt.Sprintf("unsupported api_version %d", req.APIVersion), nil), nil, protocol.Meta{})
	}

	switch req.Cmd {
	case "ping":
		return protocol.OK(map[string]any{"pong": true, "version": version.Get().Version}, protocol.Meta{})
	case "daemon.status":
		return protocol.OK(map[string]any{"status": "running", "version": version.Get().Version}, protocol.Meta{})
	case "daemon.stop":
		go d.stop()
		return protocol.OK(map[string]any{"status": "stopping"}, protocol.Meta{})
	case "profile.create":
		return d.handleProfileCreate(req)
	case "profile.list":
		return d.handleProfileList()
	case "profile.inspect":
		return d.handleProfileInspect(req)
	case "profile.delete":
		return d.handleProfileDelete(req)
	case "profile.start":
		return d.handleProfileStart(ctx, req)
	case "profile.stop":
		return d.handleProfileStop(ctx, req)
	case "tab.open":
		return d.handleTabOpen(ctx, req)
	case "tab.list":
		return d.handleTabList(ctx, req)
	case "tab.focus":
		return d.handleTabFocus(ctx, req)
	case "tab.close":
		return d.handleTabClose(ctx, req)
	case "page.goto":
		return d.handleActionGoto(ctx, req)
	case "element.click":
		return d.handleActionClick(ctx, req)
	case "element.fill":
		return d.handleActionFill(ctx, req)
	case "element.text":
		return d.handleActionText(ctx, req)
	case "wait.selector":
		return d.handleActionWaitSelector(ctx, req)
	case "wait.url":
		return d.handleActionWaitURL(ctx, req)
	default:
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "unknown command: "+req.Cmd, map[string]any{"cmd": req.Cmd}), nil, protocol.Meta{})
	}
}

func (d *Daemon) profileManager() (*profile.Manager, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.profiles != nil {
		return d.profiles, nil
	}
	layout, err := paths.Ensure()
	if err != nil {
		return nil, err
	}
	d.profiles = profile.NewManager(layout.ProfilesDir)
	return d.profiles, nil
}

func (d *Daemon) browserManager() (*browser.Manager, error) {
	d.mu.Lock()
	if d.browsers != nil {
		browsers := d.browsers
		d.mu.Unlock()
		return browsers, nil
	}
	d.mu.Unlock()

	mgr, err := d.profileManager()
	if err != nil {
		return nil, err
	}
	layout, err := paths.Ensure()
	if err != nil {
		return nil, err
	}
	browsers := browser.NewManager(mgr, layout.RuntimeDir)
	d.mu.Lock()
	if d.browsers == nil {
		d.browsers = browsers
	}
	out := d.browsers
	d.mu.Unlock()
	return out, nil
}

func (d *Daemon) tabRegistry() (*tab.Registry, error) {
	d.mu.Lock()
	if d.tabs != nil {
		tabs := d.tabs
		d.mu.Unlock()
		return tabs, nil
	}
	d.mu.Unlock()

	browsers, err := d.browserManager()
	if err != nil {
		return nil, err
	}
	tabs := tab.NewRegistry(browsers)
	d.mu.Lock()
	if d.tabs == nil {
		d.tabs = tabs
	}
	out := d.tabs
	d.mu.Unlock()
	return out, nil
}

func (d *Daemon) actionEngine() (*action.Engine, error) {
	d.mu.Lock()
	if d.actions != nil {
		engine := d.actions
		d.mu.Unlock()
		return engine, nil
	}
	d.mu.Unlock()

	tabs, err := d.tabRegistry()
	if err != nil {
		return nil, err
	}
	engine := action.NewEngine(tabs, chromedriver.New())
	d.mu.Lock()
	if d.actions == nil {
		d.actions = engine
	}
	out := d.actions
	d.mu.Unlock()
	return out, nil
}

type profileCreateArgs struct {
	ChromePath string `json:"chrome_path,omitempty"`
}

func (d *Daemon) handleProfileCreate(req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	var args profileCreateArgs
	if len(req.Args) > 0 {
		if err := json.Unmarshal(req.Args, &args); err != nil {
			return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid profile.create args: "+err.Error(), nil), nil, protocol.Meta{})
		}
	}
	mgr, err := d.profileManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	p, err := mgr.Create(name, args.ChromePath)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(p, protocol.Meta{Profile: name})
}

func (d *Daemon) handleProfileList() protocol.Response {
	mgr, err := d.profileManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{})
	}
	profiles, err := mgr.List()
	if err != nil {
		return errorResponse(err, protocol.Meta{})
	}
	return protocol.OK(map[string]any{"profiles": profiles}, protocol.Meta{})
}

func (d *Daemon) handleProfileInspect(req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	mgr, err := d.profileManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	p, err := mgr.Inspect(name)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(p, protocol.Meta{Profile: name})
}

func (d *Daemon) handleProfileDelete(req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	mgr, err := d.profileManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	if err := mgr.Delete(name); err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(map[string]any{"deleted": name}, protocol.Meta{Profile: name})
}

func (d *Daemon) handleProfileStart(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	mgr, err := d.browserManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	rt, err := mgr.Start(ctx, name)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(rt, protocol.Meta{Profile: name})
}

func (d *Daemon) handleProfileStop(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	mgr, err := d.browserManager()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	rt, err := mgr.Stop(ctx, name)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(rt, protocol.Meta{Profile: name})
}

type tabOpenArgs struct {
	URL string `json:"url"`
}

func (d *Daemon) handleTabOpen(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	var args tabOpenArgs
	if len(req.Args) > 0 {
		if err := json.Unmarshal(req.Args, &args); err != nil {
			return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid tab.open args: "+err.Error(), nil), nil, protocol.Meta{Profile: name})
		}
	}
	tabs, err := d.tabRegistry()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	t, err := tabs.Open(ctx, name, args.URL)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(t, protocol.Meta{Profile: name, Tab: t.ID})
}

func (d *Daemon) handleTabList(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	tabs, err := d.tabRegistry()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	list, err := tabs.List(ctx, name)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name})
	}
	return protocol.OK(map[string]any{"tabs": list}, protocol.Meta{Profile: name})
}

func (d *Daemon) handleTabFocus(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	tabs, err := d.tabRegistry()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name, Tab: req.Tab})
	}
	t, err := tabs.Focus(ctx, name, req.Tab)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name, Tab: req.Tab})
	}
	return protocol.OK(t, protocol.Meta{Profile: name, Tab: t.ID})
}

func (d *Daemon) handleTabClose(ctx context.Context, req protocol.Request) protocol.Response {
	name := req.Profile
	if name == "" {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "profile is required", nil), nil, protocol.Meta{})
	}
	tabs, err := d.tabRegistry()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name, Tab: req.Tab})
	}
	t, err := tabs.Close(ctx, name, req.Tab)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: name, Tab: req.Tab})
	}
	return protocol.OK(t, protocol.Meta{Profile: name, Tab: t.ID})
}

func (d *Daemon) handleActionGoto(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.GotoArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid page.goto args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.Goto(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func (d *Daemon) handleActionClick(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.SelectorArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid element.click args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.Click(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func (d *Daemon) handleActionFill(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.FillArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid element.fill args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.Fill(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func (d *Daemon) handleActionText(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.SelectorArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid element.text args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.Text(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func (d *Daemon) handleActionWaitSelector(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.SelectorArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid wait.selector args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.WaitSelector(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func (d *Daemon) handleActionWaitURL(ctx context.Context, req protocol.Request) protocol.Response {
	var args action.WaitURLArgs
	if err := json.Unmarshal(req.Args, &args); err != nil {
		return protocol.Fail(protocol.NewError(protocol.InvalidRequest, "invalid wait.url args: "+err.Error(), nil), nil, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	engine, err := d.actionEngine()
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	data, err := engine.WaitURL(ctx, req.Profile, req.Tab, args)
	if err != nil {
		return errorResponse(err, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
	}
	return protocol.OK(data, protocol.Meta{Profile: req.Profile, Tab: req.Tab})
}

func errorResponse(err error, meta protocol.Meta) protocol.Response {
	if perr, ok := err.(*protocol.Error); ok {
		return protocol.Fail(perr, nil, meta)
	}
	return protocol.Fail(protocol.NewError(protocol.InternalError, err.Error(), nil), nil, meta)
}

func (d *Daemon) stop() {
	d.mu.Lock()
	cancel := d.cancel
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
