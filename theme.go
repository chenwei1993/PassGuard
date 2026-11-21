package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// themeDark resolves the theme setting against the desktop's appearance.
func (a *App) themeDark() bool {
	switch a.settings.ThemeMode {
	case "dark":
		return true
	case "light":
		return false
	default:
		return mygo.Theme.IsDark()
	}
}

func (a *App) themeSource() mygo.ThemeSource {
	switch a.settings.ThemeMode {
	case "dark":
		return mygo.ThemeDark
	case "system":
		return mygo.ThemeSystem
	default:
		return mygo.ThemeLight
	}
}

// applyTheme installs the app's brand theme for the frame. Every view calls
// it first, so the whole window follows the theme and font settings.
func (a *App) applyTheme(c *ui.Context) {
	dark := a.themeDark()
	var t *ui.Theme
	if dark {
		t = ui.DarkTheme()
	} else {
		t = ui.LightTheme()
	}
	t = t.WithPreferences(c.Preferences())

	accent := "#6366f1"
	if dark {
		accent = "#818cf8"
	}
	t = t.WithAccent(ui.Hex(accent))

	tt := *t
	tt.Radius = 10
	tt.Spacing = 4
	tt.FontSize = a.baseFontSize()
	tt.ScrollbarWidth = 8
	tt.ScrollbarStyle = ui.ScrollbarStyle{ThumbWidth: 8, TrackWidth: 10, Radius: 4, MinLength: 24}
	switch runtime.GOOS {
	case "windows":
		tt.Font = "Microsoft YaHei, Segoe UI"
	case "darwin":
		tt.Font = "PingFang SC, Helvetica Neue"
	}

	if dark {
		tt.Background = ui.Hex("#0b0b12")
		tt.Surface = ui.Hex("#14141f")
		tt.SurfaceHover = ui.Hex("#1e1e2e")
		tt.SurfacePressed = ui.Hex("#252536")
		tt.Border = ui.Hex("#2a2a3a")
		tt.Text = ui.Hex("#e2e8f0")
		tt.TextMuted = ui.Hex("#94a3b8")
		tt.Danger = ui.Hex("#f87171")
		tt.Warning = ui.Hex("#fbbf24")
		tt.Success = ui.Hex("#34d399")
		tt.Accent = ui.Hex("#818cf8")
		tt.InactiveAccent = ui.Hex("#818cf8")
		tt.Selection = ui.RGBA(129, 140, 248, 0.26)
		tt.Focus = ui.RGBA(129, 140, 248, 0.22)
	} else {
		tt.Background = ui.Hex("#f8fafc")
		tt.Surface = ui.Hex("#ffffff")
		tt.SurfaceHover = ui.Hex("#f1f5f9")
		tt.SurfacePressed = ui.Hex("#e2e8f0")
		tt.Border = ui.Hex("#e2e8f0")
		tt.Text = ui.Hex("#0f172a")
		tt.TextMuted = ui.Hex("#64748b")
		tt.Danger = ui.Hex("#ef4444")
		tt.Warning = ui.Hex("#d97706")
		tt.Success = ui.Hex("#10b981")
		tt.Accent = ui.Hex("#6366f1")
		tt.InactiveAccent = ui.Hex("#6366f1")
		tt.Selection = ui.RGBA(99, 102, 241, 0.14)
		tt.Focus = ui.RGBA(99, 102, 241, 0.20)
	}
	c.SetTheme(&tt)
}

// Surfaces and colors from the design tokens.
func (a *App) surface2() ui.Color {
	if a.themeDark() {
		return ui.Hex("#1a1a28")
	}
	return ui.Hex("#f8fafc")
}

func (a *App) surface3() ui.Color {
	if a.themeDark() {
		return ui.Hex("#252536")
	}
	return ui.Hex("#f1f5f9")
}

func (a *App) sidebarBg() ui.Color {
	if a.themeDark() {
		return ui.Hex("#12121c")
	}
	return ui.Hex("#ffffff")
}

func (a *App) sidebarHover() ui.Color {
	if a.themeDark() {
		return ui.Hex("#1e1e2e")
	}
	return ui.Hex("#f1f5f9")
}

func (a *App) sidebarActive() ui.Color {
	if a.themeDark() {
		return ui.Hex("#2a2a4a")
	}
	return ui.Hex("#eef2ff")
}

func (a *App) themeSurface() ui.Color {
	if a.themeDark() {
		return ui.Hex("#14141f")
	}
	return ui.Hex("#ffffff")
}

func (a *App) themeShadow() ui.Color {
	if a.themeDark() {
		return ui.RGBA(0, 0, 0, 0.55)
	}
	return ui.RGBA(15, 23, 42, 0.10)
}

func (a *App) cardShadow() ui.Color {
	if a.themeDark() {
		return ui.RGBA(0, 0, 0, 0.35)
	}
	return ui.RGBA(15, 23, 42, 0.06)
}

func infoColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#60a5fa")
	}
	return ui.Hex("#3b82f6")
}

// folderColor is the icon color of a folder, as the design gives each one.
func folderColor(folder string) ui.Color {
	switch folder {
	case "工作":
		return ui.Hex("#60a5fa")
	case "个人":
		return ui.Hex("#a78bfa")
	case "财务":
		return ui.Hex("#34d399")
	case "社交":
		return ui.Hex("#f472b6")
	case "购物":
		return ui.Hex("#fbbf24")
	default:
		return ui.Hex("#94a3b8")
	}
}
