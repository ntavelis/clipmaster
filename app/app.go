package app

import (
	"context"

	"clipmaster/business/clipboard"
	"clipmaster/business/theme"
	"clipmaster/foundation/config"
	osclip "clipmaster/foundation/clipboard"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails bind target. It owns startup/shutdown and delegates to business packages.
type App struct {
	ctx     context.Context
	cfg     config.AppConfig
	monitor *clipboard.Monitor
	colors  theme.ThemeColors
}

// NewApp creates an App with default configuration.
func NewApp() *App {
	cfg := config.Default()
	backend := osclip.WaylandClipboard{}
	return &App{
		cfg:     cfg,
		monitor: clipboard.NewMonitor(backend, cfg.MaxHistory, cfg.PollInterval),
	}
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	colors, err := theme.Load(a.cfg.ThemeColorPath)
	if err != nil {
		runtime.LogWarningf(ctx, "could not load theme: %v", err)
	} else {
		a.colors = colors
		runtime.EventsEmit(ctx, "theme:loaded", colors)
	}

	a.monitor.OnNewEntry = func(entry clipboard.ClipboardEntry) {
		runtime.EventsEmit(ctx, "clipboard:new", entry)
	}
	a.monitor.Start(ctx)
}

// Shutdown is called by Wails when the application is closing.
func (a *App) Shutdown(ctx context.Context) {
	a.monitor.Stop()
}

// GetHistory returns all clipboard entries in reverse-chronological order.
func (a *App) GetHistory() []clipboard.ClipboardEntry {
	return a.monitor.GetHistory()
}

// CopyItem writes the entry with the given ID back to the system clipboard.
func (a *App) CopyItem(id string) error {
	return a.monitor.CopyItem(id)
}

// GetTheme returns the currently loaded theme colors.
func (a *App) GetTheme() theme.ThemeColors {
	return a.colors
}
