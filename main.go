package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
)

var (
	mainWindowMu  sync.Mutex
	mainWindow    *mygo.Window
	appTray       *mygo.Tray
	quitting      bool
	trayHintShown bool
	theApp        *App
)

func userDataDir() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return dir
}

// copyDirMerge copies what the target does not have yet, so newer data wins.
func copyDirMerge(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
			if err := copyDirMerge(s, d); err != nil {
				return err
			}
			continue
		}
		if _, err := os.Stat(d); err == nil {
			continue
		}
		if err := copyFile(s, d); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// migrateLegacyData brings the Electron app's data directory over, so an
// existing vault opens in this build.
func migrateLegacyData() {
	appData, err := mygo.App.Path(mygo.PathAppData)
	if err != nil {
		return
	}
	target := userDataDir()
	if target == "" {
		return
	}
	for _, name := range []string{"Password Vault", "localpwd"} {
		legacy := filepath.Join(appData, name)
		if legacy == target {
			continue
		}
		if _, err := os.Stat(legacy); err != nil {
			continue
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			log.Printf("[migrate] %v", err)
			continue
		}
		if err := copyDirMerge(legacy, target); err != nil {
			log.Printf("[migrate] %s: %v", legacy, err)
		}
	}
}

func trayIcon() []byte {
	dir, err := mygo.App.Path(mygo.PathResources)
	if err == nil {
		for _, name := range []string{"tray.png", "icon.png"} {
			if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				return data
			}
		}
	}
	return nil
}

func buildTrayMenu(a *App) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: a.t("nav.vault"), Click: func(_ *mygo.MenuItem, _ *mygo.Window) { showMainWindow() }},
		{Label: a.t("nav.lockNow"), Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			a.lockVault()
			if a.win != nil {
				a.win.Invalidate()
			}
		}},
		mygo.Separator(),
		{Label: a.quitLabel(), Click: func(_ *mygo.MenuItem, _ *mygo.Window) { mygo.App.Quit() }},
	})
}

func (a *App) quitLabel() string {
	if strings.HasPrefix(a.locale, "zh") {
		return "退出"
	}
	return "Quit"
}

func (a *App) whenHiddenTitle() string {
	if strings.HasPrefix(a.locale, "zh") {
		return "守密仍在后台运行"
	}
	return "PassGuard is still running"
}

func (a *App) whenHiddenBody() string {
	if strings.HasPrefix(a.locale, "zh") {
		return "已最小化到系统托盘，点击托盘图标可重新打开窗口。"
	}
	return "Minimized to the system tray. Click the tray icon to reopen the window."
}

func createTray(a *App) {
	t, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "main",
		Icon:    trayIcon(),
		ToolTip: a.t("app.name") + " - " + a.t("vault.title"),
		Menu:    buildTrayMenu(a),
	})
	if err != nil {
		log.Printf("[tray] %v", err)
		return
	}
	appTray = t
	t.OnClick(showMainWindow)
}

func showMainWindow() {
	mainWindowMu.Lock()
	w := mainWindow
	mainWindowMu.Unlock()
	if w == nil {
		if theApp != nil {
			theApp.createWindow()
		}
		return
	}
	if w.IsMinimized() {
		w.Restore()
	}
	w.Show()
	w.Focus()
}

func (a *App) notifyStillRunning() {
	n := mygo.NewNotification(mygo.NotificationOptions{
		Title: a.whenHiddenTitle(),
		Body:  a.whenHiddenBody(),
	})
	n.OnClick(showMainWindow)
	_ = n.Show()
}

// createWindow opens the main window, whose content MyGo draws itself.
func (a *App) createWindow() {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:           a.t("app.name"),
		Width:           1440,
		Height:          900,
		MinWidth:        1000,
		MinHeight:       650,
		Frameless:       true,
		BackgroundColor: "light-dark(#f8fafc, #0b0b12)",
		StateKey:        "main",
		Content:         ui.View(a.root),
	})
	a.win = win
	a.vault.SetParent(win)
	mainWindowMu.Lock()
	mainWindow = win
	mainWindowMu.Unlock()

	win.OnMinimize(func() {
		if a.settings.LockOnMinimize && a.phase == "unlocked" {
			a.lockVault()
			win.Invalidate()
		}
	})
	win.OnClose(func(e *mygo.CloseEvent) {
		if quitting {
			return
		}
		e.PreventDefault()
		win.Hide()
		if !trayHintShown {
			trayHintShown = true
			a.notifyStillRunning()
		}
	})
	win.OnClosed(func() {
		mainWindowMu.Lock()
		if mainWindow == win {
			mainWindow = nil
		}
		mainWindowMu.Unlock()
	})
}

func main() {
	mygo.App.SetName("PassGuard")
	mygo.App.SetVersion(appVersion)
	mygo.App.SetLanguages("en", "zh-Hans")

	// A data directory of its own, for trying the app or keeping its data
	// elsewhere.
	if dir := os.Getenv("PASSGUARD_DATA"); dir != "" {
		mygo.App.SetPath(mygo.PathUserData, dir)
	}

	if os.Getenv("MYGO_GENERATE") == "" && !mygo.App.RequestSingleInstanceLock() {
		return
	}

	a := NewApp()
	theApp = a
	mygo.Theme.SetSource(a.themeSource())

	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		showMainWindow()
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		quitting = true
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			showMainWindow()
		}
	})
	mygo.App.OnLocaleChanged(func(tag string) {
		a.locale = tag
		if a.win != nil {
			a.win.SetTitle(a.t("app.name"))
		}
		if appTray != nil {
			appTray.SetMenu(buildTrayMenu(a))
		}
	})

	mygo.App.WhenReady(func() {
		migrateLegacyData()
		a.refreshShortcutIcons()
		a.createWindow()
		createTray(a)
		go a.boot()
		if a.settings.AutoCheckUpdates {
			a.checkForUpdates(false)
		}
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
