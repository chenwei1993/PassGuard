package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppSettings is what the app remembers between launches. The Electron app
// kept these in localStorage; the native build keeps them in settings.json
// in the app's data directory.
type AppSettings struct {
	Version          int    `json:"version"`
	AutoLockMinutes  int    `json:"autoLockMinutes"`
	LockOnStartup    bool   `json:"lockOnStartup"`
	LockOnMinimize   bool   `json:"lockOnMinimize"`
	ClipClear        bool   `json:"clipClear"`
	ClipClearSecs    int    `json:"clipClearSeconds"`
	ThemeMode        string `json:"themeMode"` // light | dark | system
	FontSize         string `json:"fontSize"`  // small | normal | large
	NotifyWeak       bool   `json:"notifyWeak"`
	NotifyBreach     bool   `json:"notifyBreach"`
	NotifyExpiry     bool   `json:"notifyExpiry"`
	NotifyBackup     bool   `json:"notifyBackup"`
	SidebarCollapsed bool   `json:"sidebarCollapsed"`
	// ShellIconsVersion is the version of the app's icon that Explorer was
	// last told about, so that an update refreshes the shortcuts' icons
	// once, not on every launch.
	ShellIconsVersion int `json:"shellIconsVersion"`
}

const settingsVersion = 1

func defaultSettings() AppSettings {
	return AppSettings{
		Version:         settingsVersion,
		AutoLockMinutes: 5,
		LockOnStartup:   true,
		LockOnMinimize:  false,
		ClipClear:       false,
		ClipClearSecs:   0,
		ThemeMode:       "light",
		FontSize:        "normal",
		NotifyWeak:      true,
		NotifyBreach:    true,
		NotifyExpiry:    true,
		NotifyBackup:    false,
	}
}

func (a *App) settingsFile() string {
	return filepath.Join(userDataDir(), "settings.json")
}

func (a *App) loadSettings() {
	a.settings = defaultSettings()
	raw, err := os.ReadFile(a.settingsFile())
	if err != nil {
		return
	}
	var s AppSettings
	if json.Unmarshal(raw, &s) != nil {
		return
	}
	if s.Version != settingsVersion {
		// A file from before the format was versioned, or a partial one.
		return
	}
	// Fill what an older file lacks.
	if s.ThemeMode != "light" && s.ThemeMode != "dark" && s.ThemeMode != "system" {
		s.ThemeMode = "light"
	}
	if s.FontSize != "small" && s.FontSize != "normal" && s.FontSize != "large" {
		s.FontSize = "normal"
	}
	if s.AutoLockMinutes < 0 {
		s.AutoLockMinutes = 0
	}
	a.settings = s
}

// shellIconsVersion grows when the app's icon changes. Windows caches a
// shortcut's icon by its path, and the installer makes the desktop shortcut
// after its own steps, so the first launch of a new version is the earliest
// point to have Explorer redraw them.
const shellIconsVersion = 1

// refreshShortcutIcons asks Explorer to reread the app's icons once per
// version, which clears a blank icon left by a build from before the
// executable carried one.
func (a *App) refreshShortcutIcons() {
	if a.settings.ShellIconsVersion >= shellIconsVersion {
		return
	}
	refreshShellIcons()
	a.settings.ShellIconsVersion = shellIconsVersion
	a.saveSettings()
}

func (a *App) saveSettings() {
	a.settings.Version = settingsVersion
	raw, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(userDataDir(), 0o755)
	_ = os.WriteFile(a.settingsFile(), raw, 0o600)
}

func (a *App) baseFontSize() float32 {
	switch a.settings.FontSize {
	case "small":
		return 12
	case "large":
		return 16
	default:
		return 14
	}
}
