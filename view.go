package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func timeNow() time.Time { return time.Now() }

// folders are the stored folder names, in the app's order.
var folders = []string{"工作", "个人", "财务", "社交", "购物", "其他"}

var tags = []string{"重要", "紧急", "待更新"}

func (a *App) root(c *ui.Context) {
	a.locale = c.Locale()
	a.applyTheme(c)
	a.drainToasts(c)
	a.watchActivity(c)
	// The locale MyGo chooses at startup may differ from the one it had
	// before Run; the title and the tray follow the first frame.
	if name := a.t("app.name"); name != a.title {
		a.title = name
		if a.win != nil {
			a.win.SetTitle(name)
		}
		if appTray != nil {
			appTray.SetToolTip(name + " - " + a.t("vault.title"))
			appTray.SetMenu(buildTrayMenu(a))
		}
	}

	ui.Column(c).Fill().Children(func() {
		a.windowTitlebar(c)
		ui.Box(c).Grow(1).Children(func() {
			switch a.phase {
			case "booting":
				a.viewBooting(c)
			case "setup", "locked":
				a.viewLock(c)
			default:
				a.viewShell(c)
			}
		})
	})

	a.viewModals(c)
	a.handleShortcuts(c)
}

// windowTitlebar draws the frameless window's title bar, as the web
// design had it: the brand at the start, the local mode note, and the
// window buttons at the end.
func (a *App) windowTitlebar(c *ui.Context) {
	t := c.Theme()
	bar := ui.Row(c)
	bar.Height(40)
	bar.FillWidth()
	bar.Background(a.surface2())
	bar.BorderWidth(0, 0, 1, 0)
	bar.BorderColor(t.Border)
	bar.DragWindow()
	if bar.WindowDoubleClicked() && a.win != nil {
		a.win.ToggleMaximize()
	}
	bar.Children(func() {
		ui.Row(c).Padding(0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			logo := ui.Box(c).Size(18, 18).Radius(4).Gradient(ui.Hex("#6366f1"), ui.Hex("#a855f7"), 135).Center()
			logo.Children(func() {
				ui.Icon(c, icon("shield")).FontSize(11).TextColor(ui.Hex("#ffffff"))
			})
			ui.Text(c, a.t("app.name")).
				FontSize(t.FontSize * 0.9286).
				FontWeight(600).
				LetterSpacing(0.2)
		})
		ui.Spacer(c)
		if a.phase == "unlocked" {
			ui.Row(c).Padding(0, 12).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Success)
				ui.Icon(c, icon("hard-drive")).FontSize(13).TextColor(t.TextMuted)
				ui.Text(c, a.t("titlebar.localMode")).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
			})
		}
		a.windowButton(c, "minus", false, func() {
			if a.win != nil {
				a.win.Minimize()
			}
		})
		a.windowButton(c, "square", false, func() {
			if a.win != nil {
				a.win.ToggleMaximize()
			}
		})
		a.windowButton(c, "x", true, func() {
			if a.win != nil {
				a.win.Close()
			}
		})
	})
}

func (a *App) windowButton(c *ui.Context, iconName string, close bool, onClick func()) {
	t := c.Theme()
	b := ui.Box(c).Width(46).Height(40).Center().Cursor(ui.CursorPointer)
	if b.Hovered() {
		if close {
			b.Background(ui.Hex("#ef4444"))
		} else {
			b.Background(a.surface3())
		}
	}
	b.Label(a.t("titlebar.minimize"))
	if close {
		b.Label(a.t("titlebar.close"))
	} else if iconName == "square" {
		b.Label(a.t("titlebar.maximize"))
	}
	b.OnClick(onClick)
	b.Children(func() {
		col := t.TextMuted
		if b.Hovered() {
			col = t.Text
			if close {
				col = ui.Hex("#ffffff")
			}
		}
		ui.Icon(c, icon(iconName)).FontSize(13).TextColor(col)
	})
}

// handleShortcuts runs the app's keyboard commands after the frame is
// built: Ctrl+L locks, Ctrl+N adds an entry and Ctrl+F searches.
func (a *App) handleShortcuts(c *ui.Context) {
	if a.phase != "unlocked" {
		return
	}
	if a.entryOpen || a.deleteOpen || a.notifOpen || a.restoreOpen || a.importOpen || a.backupPwOpen || a.backupDeleteOpen {
		return
	}
	if c.Shortcut(ui.Ctrl, ui.KeyL) {
		a.lockVault()
		return
	}
	if c.Shortcut(ui.Ctrl, ui.KeyN) {
		a.openAddEntry("")
		return
	}
	if c.Shortcut(ui.Ctrl, ui.KeyF) {
		a.page = "vault"
		a.focusSearch = true
	}
}

// watchActivity resets the idle timer on any pointer or key input.
func (a *App) watchActivity(c *ui.Context) {
	c.Root().ObserveInput(func(ev ui.InputEvent) {
		a.touchActivity()
	})
}

func (a *App) viewBooting(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Spinner(c).Size(24, 24)
		a.mutedText(c, a.t("status.loading"))
	})
}

func (a *App) viewShell(c *ui.Context) {
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.sidebar(c)
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			a.topbar(c)
			ui.Box(c).Grow(1).Children(func() {
				a.pageContent(c)
			})
			a.statusbar(c)
		})
	})
}

func (a *App) sidebarWidth() float32 {
	if a.settings.SidebarCollapsed {
		return 72
	}
	return 260
}

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	col := ui.Column(c)
	col.Width(a.sidebarWidth())
	col.FillHeight()
	col.Background(a.sidebarBg())
	col.BorderWidth(0, 1, 0, 0)
	col.BorderColor(t.Border)
	col.Children(func() {
		ui.Scroll(c).Grow(1).Scrollbars(false).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				topPad := float32(16)
				sidePad := float32(8)
				if a.settings.SidebarCollapsed {
					topPad = 8
					sidePad = 6
				}
				ui.Column(c).Padding(topPad, sidePad, 8).Gap(2).Children(func() {
					a.sidebarItem(c, "lock", a.t("nav.vault"), a.page == "vault" && a.view == "all" && a.folder == "all", 0, false, func() {
						a.view = "all"
						a.folder = "all"
						a.tag = ""
						a.securityFilter = "all"
						a.resetSelection()
						a.setPage("vault")
					})
					a.sidebarItem(c, "key-round", a.t("nav.generator"), a.page == "generator", 0, false, func() { a.setPage("generator") })
					a.sidebarItem(c, "shield-check", a.t("nav.security"), a.page == "security", 0, false, func() { a.setPage("security") })
				})

				ui.Column(c).Padding(0, sidePad, 12).Gap(2).Children(func() {
					a.sidebarSection(c, a.t("nav.categories"))
					a.sidebarItem(c, "star", a.t("nav.favorites"), a.page == "vault" && a.view == "favorites", 0, false, func() {
						a.view = "favorites"
						a.folder = "all"
						a.securityFilter = "all"
						a.tag = ""
						a.resetSelection()
						a.setPage("vault")
					})
					a.sidebarItem(c, "clock", a.t("nav.recent"), a.page == "vault" && a.view == "recent", 0, false, func() {
						a.view = "recent"
						a.folder = "all"
						a.securityFilter = "all"
						a.tag = ""
						a.resetSelection()
						a.setPage("vault")
					})
				})

				cnt := a.computeCounts()
				ui.Column(c).Padding(0, sidePad, 12).Gap(2).Children(func() {
					a.sidebarSection(c, a.t("vault.folders"))
					for _, f := range folders {
						folder := f
						a.sidebarItem(c, a.folderIcon(folder), a.folderLabel(folder), a.page == "vault" && a.folder == folder, cnt.Folders[folder], true, func() {
							a.folder = folder
							a.view = "all"
							a.securityFilter = "all"
							a.tag = ""
							a.resetSelection()
							a.setPage("vault")
						})
					}
				})

				ui.Column(c).Padding(0, sidePad, 16).Gap(2).Children(func() {
					a.sidebarSection(c, a.t("nav.dataManagement"))
					a.sidebarItem(c, "hard-drive", a.t("nav.backup"), a.page == "backup", 0, false, func() { a.setPage("backup") })
					a.sidebarItem(c, "arrow-down-up", a.t("nav.transfer"), a.page == "transfer", 0, false, func() { a.setPage("transfer") })
					a.sidebarItem(c, "settings", a.t("nav.settings"), a.page == "settings", 0, false, func() { a.setPage("settings") })
				})
			})
		})
		a.sidebarFooter(c)
	})
}

func (a *App) sidebarSection(c *ui.Context, title string) {
	if a.settings.SidebarCollapsed {
		return
	}
	ui.Text(c, strings.ToUpper(title)).
		FontSize(11).
		FontWeight(600).
		LetterSpacing(0.5).
		TextColor(c.Theme().TextMuted).
		Padding(8, 8).
		Margin(8, 6, 0, 0)
}

func (a *App) sidebarItem(c *ui.Context, iconName, label string, active bool, count int, compact bool, onClick func()) {
	t := c.Theme()
	fontSize := t.FontSize * 1.0
	rowPadding := float32(9)
	gap := float32(12)
	radius := float32(10)
	if compact {
		fontSize = t.FontSize * 0.9286
		rowPadding = 7
		gap = 10
		radius = 6
	}
	row := ui.Row(c).Gap(gap).Padding(rowPadding, 12).Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Label(label)
	switch {
	case active:
		row.Background(a.sidebarActive())
	case row.Hovered():
		row.Background(a.sidebarHover())
	}
	if a.settings.SidebarCollapsed {
		row.Justify(ui.Center).Padding(rowPadding, 0)
	}
	row.OnClick(onClick)
	row.Children(func() {
		col := t.TextMuted
		if active {
			col = t.Accent
		} else if row.Hovered() {
			col = t.Text
		}
		ui.Icon(c, icon(iconName)).FontSize(16).TextColor(col)
		if !a.settings.SidebarCollapsed {
			ui.Text(c, label).FontSize(fontSize).FontWeight(500).Grow(1).SingleLine().TextColor(col)
			if count > 0 {
				a.countPill(c, count, active)
			}
		}
	})
}

func (a *App) countPill(c *ui.Context, count int, active bool) {
	t := c.Theme()
	bg := a.surface3()
	col := t.TextMuted
	if active {
		bg = ui.RGBA(129, 140, 248, 0.15)
		col = t.Accent
	}
	ui.Box(c).Padding(2, 8).Radius(999).Background(bg).Children(func() {
		ui.Text(c, fmt.Sprint(count)).FontSize(11.5).FontWeight(500).TextColor(col)
	})
}

func (a *App) sidebarFooter(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		if a.settings.SidebarCollapsed {
			ui.Row(c).Padding(12, 0).Justify(ui.Center).Children(func() {
				box := ui.Box(c).Size(38, 38).Radius(10).Background(ui.RGBA(99, 102, 241, 0.15)).Center()
				box.Children(func() { ui.Icon(c, icon("shield-check")).FontSize(20).TextColor(t.Accent) })
			})
		} else {
			card := ui.Row(c).Padding(12, 16).Gap(12).Background(a.surface2()).AlignItems(ui.Center)
			card.Children(func() {
				iconBox := ui.Box(c).Size(38, 38).Radius(10).Background(ui.RGBA(99, 102, 241, 0.15)).Center()
				iconBox.Children(func() { ui.Icon(c, icon("shield-check")).FontSize(20).TextColor(t.Accent) })
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(7, 7).Radius(3.5).Background(t.Success)
						ui.Text(c, a.t("nav.vaultUnlocked")).FontSize(13).FontWeight(600).TextColor(t.Success).SingleLine()
					})
					ui.Text(c, a.autoLockText()).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				})
				lock := ui.Box(c).Size(32, 32).Radius(16).Center().Cursor(ui.CursorPointer)
				if lock.Hovered() {
					lock.Background(t.Accent)
				}
				lock.Label(a.t("nav.lockNow"))
				lock.OnClick(a.lockVault)
				lock.Children(func() {
					col := t.TextMuted
					if lock.Hovered() {
						col = t.AccentText
					}
					ui.Icon(c, icon("lock")).FontSize(15).TextColor(col)
				})
			})
		}
		dir := "panel-left-close"
		label := a.t("nav.collapseSidebar")
		if a.settings.SidebarCollapsed {
			dir = "panel-left-open"
			label = a.t("nav.expandSidebar")
		}
		toggle := ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).Cursor(ui.CursorPointer)
		if a.settings.SidebarCollapsed {
			toggle.Justify(ui.Center).Padding(12, 0)
		}
		if toggle.Hovered() {
			toggle.Background(a.sidebarHover())
		}
		toggle.Label(label)
		toggle.OnClick(func() {
			a.settings.SidebarCollapsed = !a.settings.SidebarCollapsed
			a.saveSettings()
		})
		toggle.Children(func() {
			col := t.TextMuted
			if toggle.Hovered() {
				col = t.Text
			}
			ui.Icon(c, icon(dir)).FontSize(15).TextColor(col)
			if !a.settings.SidebarCollapsed {
				ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(col).SingleLine()
			}
		})
	})
}

func (a *App) topbar(c *ui.Context) {
	t := c.Theme()
	row := ui.Row(c)
	row.Height(64)
	row.Padding(0, 24)
	row.Gap(20)
	row.AlignItems(ui.Center)
	row.BorderWidth(0, 0, 1, 0)
	row.BorderColor(t.Border)
	row.Background(t.Background)
	row.Children(func() {
		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, a.pageTitle()).FontSize(t.FontSize * 1.1429).FontWeight(700).SingleLine()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("app.name")).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted)
				ui.Icon(c, icon("chevron-right")).FontSize(12).TextColor(t.TextMuted)
				ui.Text(c, a.pageCrumbs()).FontSize(t.FontSize * 0.9286).FontWeight(500).TextColor(t.Text).SingleLine()
			})
		})
		ui.Spacer(c)
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			a.updateButton(c)
			switch a.page {
			case "vault":
				a.searchField(c)
				a.viewToggle(c)
				a.topThemeButton(c)
				a.bellButton(c)
				if a.iconTextButton(c, "plus", a.t("vault.addPassword"), true).Clicked() {
					a.openAddEntry("")
				}
			case "generator":
				a.topThemeButton(c)
				a.bellButton(c)
				if a.iconTextButton(c, "plus", a.t("gen.addPassword"), true).Clicked() {
					a.openAddEntry("")
				}
			case "security":
				label := a.t("security.scan.start")
				if a.scanning() {
					label = a.t("security.scan.scanning")
				}
				if a.iconTextButton(c, "scan", label, true).Disabled(a.scanning()).Clicked() {
					a.startScan()
				}
				a.topThemeButton(c)
				a.bellButton(c)
			default:
				a.topThemeButton(c)
				a.bellButton(c)
			}
		})
	})
}

func (a *App) searchField(c *ui.Context) {
	t := c.Theme()
	box := ui.Box(c).Width(360).Height(36).Shrink(0)
	box.Children(func() {
		in := ui.TextInput(c, &a.search)
		in.FillWidth()
		in.Height(36)
		in.Radius(10)
		in.Background(t.Surface)
		in.Border(1, t.Border)
		in.Padding(9, 14, 9, 38)
		in.FontSize(t.FontSize * 0.9643)
		in.Placeholder(a.t("vault.searchPlaceholder"))
		in.Label(a.t("vault.searchPlaceholder"))
		in.Bind(&a.searchHandle)
		if a.focusSearch {
			a.searchHandle.Focus(c)
			a.focusSearch = false
		}
		if in.Changed() {
			a.resetSelection()
		}
		if in.Submitted() {
			a.resetSelection()
		}
		ui.Icon(c, icon("search")).FontSize(15).TextColor(t.TextMuted).
			Absolute().Left(12).Top(11)
	})
}

func (a *App) viewToggle(c *ui.Context) {
	t := c.Theme()
	box := ui.Row(c).Padding(2).Radius(10).Background(a.surface3()).Shrink(0)
	box.Children(func() {
		for _, item := range []struct {
			key   string
			icon  string
			label string
		}{
			{"list", "list", a.t("vault.listView")},
			{"grid", "layout-grid", a.t("vault.gridView")},
		} {
			item := item
			active := a.listView == item.key
			seg := ui.Box(c).Size(32, 32).Radius(6).Center().Cursor(ui.CursorPointer)
			seg.Label(item.label)
			switch {
			case active:
				seg.Background(t.Surface)
				seg.Shadow(0, 1, 2, 0, a.cardShadow())
			case seg.Hovered():
				seg.Background(t.Surface)
			}
			seg.OnClick(func() { a.listView = item.key })
			seg.Children(func() {
				col := t.TextMuted
				if active {
					col = t.Accent
				}
				ui.Icon(c, icon(item.icon)).FontSize(16).TextColor(col)
			})
		}
	})
}

func (a *App) updateButton(c *ui.Context) {
	if a.upd.info == nil {
		return
	}
	t := c.Theme()
	label := a.tf("settings.update.available", map[string]any{"version": a.upd.info.Version})
	b := ui.Box(c).Size(32, 32).Radius(10).Center().Cursor(ui.CursorPointer)
	b.Label(label)
	b.Tooltip(label)
	if b.Hovered() {
		b.Background(a.surface3())
	}
	b.OnClick(func() {
		a.page = "settings"
		a.settingsTab = 3
	})
	b.Children(func() {
		ui.Icon(c, icon("download")).FontSize(17).TextColor(t.Accent)
	})
}

func (a *App) topThemeButton(c *ui.Context) {
	t := c.Theme()
	iconName := "sun"
	label := a.t("vault.switchToDark")
	if a.themeDark() {
		iconName = "moon"
		label = a.t("vault.switchToLight")
	}
	b := ui.Box(c).Size(32, 32).Radius(10).Center().Cursor(ui.CursorPointer)
	b.Label(label)
	b.Tooltip(label)
	if b.Hovered() {
		b.Background(a.surface3())
	}
	b.OnClick(a.toggleTheme)
	b.Children(func() {
		col := t.TextMuted
		if b.Hovered() {
			col = t.Text
		}
		ui.Icon(c, icon(iconName)).FontSize(18).TextColor(col)
	})
}

func (a *App) bellButton(c *ui.Context) {
	t := c.Theme()
	wrap := ui.Box(c).Size(32, 32)
	wrap.Children(func() {
		b := ui.Box(c).Fill().Radius(10).Center().Cursor(ui.CursorPointer)
		b.Label(a.t("notif.title"))
		b.Tooltip(a.t("notif.title"))
		if b.Hovered() {
			b.Background(a.surface3())
		}
		b.OnClick(func() { a.notifOpen = true })
		b.Children(func() {
			col := t.TextMuted
			if b.Hovered() {
				col = t.Text
			}
			ui.Icon(c, icon("bell")).FontSize(18).TextColor(col)
		})
		if a.notificationCount() > 0 {
			ui.Box(c).Size(8, 8).Radius(4).Background(t.Danger).
				Border(2, t.Background).
				Absolute().Top(3).Right(3)
		}
	})
}

func (a *App) statusbar(c *ui.Context) {
	t := c.Theme()
	row := ui.Row(c).Height(40).Padding(0, 20).Gap(20).AlignItems(ui.Center)
	row.BorderWidth(1, 0, 0, 0)
	row.BorderColor(t.Border)
	row.Background(t.Background)
	row.Children(func() {
		ui.Row(c).Gap(20).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon("shield-check")).FontSize(13).TextColor(t.Success)
				ui.Text(c, a.t("status.unlocked")).FontSize(12).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon("lock")).FontSize(13).TextColor(t.TextMuted)
				ui.Text(c, a.tf("status.total", map[string]any{"n": len(a.entries)})).FontSize(12).TextColor(t.TextMuted)
			})
		})
		ui.Spacer(c)
		ui.Row(c).Gap(20).AlignItems(ui.Center).Children(func() {
			dir := a.dbInfo.Dir
			if dir == "" {
				dir = a.vault.dataDir()
			}
			loc := ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(4, 10).Radius(6).Cursor(ui.CursorPointer)
			loc.Label(dir)
			loc.Tooltip(dir)
			if loc.Hovered() {
				loc.Background(t.SurfaceHover)
			}
			loc.OnClick(a.openDataFolder)
			loc.Children(func() {
				ui.Icon(c, icon("folder-open")).FontSize(13).TextColor(t.TextMuted)
				ui.Text(c, a.t("status.dataFile")+dir).FontSize(12).TextColor(t.TextMuted).SingleLine().MaxWidth(420)
			})
			lock := ui.Box(c).Size(28, 28).Radius(7).Center().Cursor(ui.CursorPointer)
			lock.Label(a.t("status.lockVaultShortcut"))
			lock.Tooltip(a.t("status.lockVaultShortcut"))
			if lock.Hovered() {
				lock.Background(t.SurfaceHover)
			}
			lock.OnClick(a.lockVault)
			lock.Children(func() {
				col := t.TextMuted
				if lock.Hovered() {
					col = t.Text
				}
				ui.Icon(c, icon("lock")).FontSize(15).TextColor(col)
			})
		})
	})
}

func (a *App) pageCrumbs() string {
	switch a.page {
	case "vault":
		if a.folder != "all" {
			return a.folderLabel(a.folder)
		}
		if a.securityFilter != "all" {
			return a.securityLabel(a.securityFilter)
		}
		if a.view == "favorites" {
			return a.t("vault.favorites")
		}
		if a.view == "recent" {
			return a.t("vault.recent")
		}
		return a.t("vault.allPasswords")
	case "generator":
		return a.t("gen.title")
	case "security":
		return a.t("security.title")
	case "transfer":
		return a.t("transfer.title")
	case "backup":
		return a.t("backup.title")
	case "settings":
		return a.t("settings.title")
	}
	return ""
}

func (a *App) securityLabel(v string) string {
	switch v {
	case "strong":
		return a.t("vault.securityStrong")
	case "medium":
		return a.t("vault.securityMedium")
	case "weak":
		return a.t("vault.securityWeak")
	case "breached":
		return a.t("issue.breached")
	default:
		return a.t("vault.securityStatus")
	}
}

func (a *App) pageContent(c *ui.Context) {
	switch a.page {
	case "vault":
		a.viewVaultPage(c)
	case "generator":
		a.viewGeneratorPage(c)
	case "security":
		a.viewSecurityPage(c)
	case "transfer":
		a.viewTransferPage(c)
	case "backup":
		a.viewBackupPage(c)
	case "settings":
		a.viewSettingsPage(c)
	default:
		a.viewVaultPage(c)
	}
}

func (a *App) scanning() bool {
	return !a.scanningUntil.IsZero() && a.scanningUntil.After(timeNow())
}

func (a *App) startScan() {
	a.scanningUntil = timeNow().Add(1200 * time.Millisecond)
	if a.win != nil {
		win := a.win
		go func() {
			time.Sleep(1200 * time.Millisecond)
			win.Update(func() {
				a.scanningUntil = time.Time{}
				issues := a.notificationIssues()
				if len(a.entries) == 0 {
					a.toast(a.t("security.scan.doneEmpty"))
				} else {
					a.toast(a.tf("security.scan.doneFound", map[string]any{"n": len(issues)}))
				}
			})
		}()
	}
}

// ---------- modals ----------

func (a *App) viewModals(c *ui.Context) {
	ui.Modal(c, &a.entryOpen, func() { a.entryModal(c) })
	ui.Modal(c, &a.deleteOpen, func() { a.deleteModal(c) })
	ui.Modal(c, &a.notifOpen, func() { a.notificationModal(c) })
	ui.Modal(c, &a.restoreOpen, func() { a.restoreModal(c) })
	ui.Modal(c, &a.importOpen, func() { a.importModal(c) })
	ui.Modal(c, &a.backupPwOpen, func() { a.backupPasswordModal(c) })
	ui.Modal(c, &a.backupDeleteOpen, func() { a.backupDeleteModal(c) })
}

func (a *App) entryModal(c *ui.Context) {
	t := c.Theme()
	title := a.t("modal.addTitle")
	if a.entryEditingID != "" {
		title = a.t("modal.editTitle")
	}
	ui.Column(c).Width(500).Gap(12).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize * 1.25).Bold()

		ui.Field(c, a.t("modal.siteName"), func() {
			ui.TextInput(c, &a.entryForm.Site).Placeholder(a.t("modal.sitePlaceholder"))
		})
		ui.Row(c).Gap(12).Children(func() {
			ui.Box(c).Grow(1).Children(func() {
				ui.Field(c, a.t("modal.usernameEmail"), func() {
					ui.TextInput(c, &a.entryForm.Username).Placeholder(a.t("modal.usernamePlaceholder"))
				})
			})
			ui.Box(c).Width(190).Children(func() {
				ui.Text(c, a.t("modal.folder")).FontSize(t.FontSize * 0.86).TextColor(t.TextMuted)
				// The folder is a drop-down, as the Electron form had: its
				// six choices are clearer in one field than as pills, which
				// wrap and shift the fields beside them.
				labels := make([]string, len(folders))
				chosen := 0
				for i, f := range folders {
					labels[i] = a.folderLabel(f)
					if f == a.entryForm.Folder {
						chosen = i
					}
				}
				sel := ui.Select(c, &labels[chosen], labels).Grow(1).Padding(8, 12)
				if sel.Changed() {
					for i, label := range labels {
						if label == labels[chosen] {
							a.entryForm.Folder = folders[i]
							break
						}
					}
				}
			})
		})
		ui.Field(c, a.t("modal.password"), func() {
			ui.Row(c).Gap(6).Children(func() {
				in := ui.TextInput(c, &a.entryForm.Password)
				in.Grow(1)
				in.Placeholder(a.t("modal.passwordPlaceholder"))
				if !a.showFormPassword {
					in.Password()
				}
				show := ui.Button(c, "").Label(a.t("modal.show")).Tooltip(a.t("modal.show"))
				show.Width(32).Height(30).Padding(0)
				if show.Clicked() {
					a.showFormPassword = !a.showFormPassword
				}
				show.Children(func() {
					name := "eye"
					if a.showFormPassword {
						name = "eye-off"
					}
					ui.Icon(c, icon(name)).FontSize(15).TextColor(t.TextMuted)
				})
				genBtn := ui.Button(c, "").Label(a.t("modal.generatePassword")).Tooltip(a.t("modal.generatePassword"))
				genBtn.Width(32).Height(30).Padding(0)
				if genBtn.Clicked() {
					a.entryForm.Password = generateRandom(16, true, true, true, true, false, false, false)
					a.toast(a.t("modal.generated"))
				}
				genBtn.Children(func() {
					ui.Icon(c, icon("wand-2")).FontSize(15).TextColor(t.TextMuted)
				})
			})
		})
		if a.entryForm.Password != "" {
			a.strengthRow(c, strengthFor(a.entryForm.Password, false))
		}
		ui.Field(c, a.t("modal.url"), func() {
			ui.TextInput(c, &a.entryForm.URL).Placeholder(a.t("modal.urlPlaceholder"))
		})
		ui.Field(c, a.t("modal.notes"), func() {
			ui.TextArea(c, &a.entryForm.Notes).Height(72).Placeholder(a.t("modal.notesPlaceholder"))
		})
		if a.formError != "" {
			ui.Text(c, a.formError).FontSize(t.FontSize * 0.86).TextColor(t.Danger)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("modal.cancel")).Clicked() {
				a.entryOpen = false
			}
			save := a.t("modal.save")
			if a.entryEditingID != "" {
				save = a.t("modal.saveChanges")
			}
			if ui.PrimaryButton(c, save).Clicked() {
				a.saveEntryForm()
			}
		})
	})
}

func (a *App) deleteModal(c *ui.Context) {
	t := c.Theme()
	name := ""
	if e := a.entryByID(a.deleteID); e != nil {
		name = e.Site
	}
	ui.Column(c).Width(380).Gap(14).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("alert-triangle")).FontSize(22).TextColor(t.Danger)
			ui.Text(c, a.tf("vault.deleteConfirmTitle", map[string]any{"name": name})).FontSize(t.FontSize * 1.1).Bold()
		})
		ui.Text(c, a.t("vault.deleteConfirmDesc")).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("vault.cancel")).Clicked() {
				a.deleteOpen = false
			}
			del := ui.Button(c, a.t("vault.confirmDelete"))
			del.Background(t.Danger).TextColor(t.AccentText)
			if del.Clicked() {
				a.deleteOpen = false
				a.removeEntry(a.deleteID)
			}
		})
	})
}

func (a *App) notificationModal(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(430).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.t("notif.title")).FontSize(t.FontSize * 1.2).Bold()
			if n := a.notificationCount(); n > 0 {
				ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize * 0.78).Bold().TextColor(t.Danger)
			}
		})

		shown := false
		if a.settings.NotifyBackup {
			if due, days := a.backupDue(); due {
				shown = true
				site := a.t("notif.backupNone")
				if days > 0 {
					site = a.tf("notif.backupDays", map[string]any{"d": days})
				}
				row := ui.Row(c).Padding(10).Gap(10).Radius(8).Background(t.Surface).Cursor(ui.CursorPointer)
				if row.Hovered() {
					row.Background(t.SurfaceHover)
				}
				row.OnClick(func() {
					a.notifOpen = false
					a.setPage("backup")
				})
				row.Children(func() {
					ui.Icon(c, icon("hard-drive")).FontSize(16).TextColor(t.Warning)
					ui.Column(c).Grow(1).Gap(1).Children(func() {
						ui.Text(c, site).FontSize(t.FontSize * 0.9).Bold()
						ui.Text(c, a.t("notif.backupSoon")).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
					})
				})
			}
		}
		for _, issue := range a.notificationIssues() {
			shown = true
			it := issue
			row := ui.Row(c).Padding(10).Gap(10).Radius(8).Background(t.Surface).Cursor(ui.CursorPointer)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.OnClick(func() {
				a.notifOpen = false
				a.focusIssue(it)
			})
			row.Children(func() {
				tagColor := t.Warning
				switch it.Risk {
				case "high":
					tagColor = t.Danger
				case "low":
					tagColor = t.TextMuted
				}
				ui.Box(c).Padding(2, 8).Radius(999).Background(t.Background).Children(func() {
					ui.Text(c, a.issueTypeLabel(it.Type)).FontSize(t.FontSize * 0.74).TextColor(tagColor)
				})
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					site := a.t("common.unnamed")
					sub := ""
					if it.Entry != nil {
						site = it.Entry.Site
						sub = it.Entry.Username
						if sub == "" {
							sub = it.Entry.URL
						}
					}
					ui.Text(c, site).FontSize(t.FontSize * 0.9).Bold().SingleLine()
					ui.Text(c, sub+" · "+a.riskLabel(it.Risk)).FontSize(t.FontSize * 0.78).TextColor(t.TextMuted).SingleLine()
				})
			})
		}
		if !shown {
			ui.Column(c).FillWidth().Center().Padding(30).Gap(6).Children(func() {
				ui.Icon(c, icon("circle-check")).FontSize(20).TextColor(t.Success)
				ui.Text(c, a.t("notif.empty")).TextColor(t.TextMuted)
			})
		}
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("notif.viewAll")).Clicked() {
				a.notifOpen = false
				a.setPage("security")
			}
		})
	})
}

func (a *App) issueTypeLabel(t string) string {
	switch t {
	case "backup":
		return a.t("issue.backup")
	case "breached":
		return a.t("issue.breached")
	case "duplicate":
		return a.t("issue.duplicate")
	case "expired":
		return a.t("issue.expired")
	default:
		return a.t("issue.weak")
	}
}

func (a *App) focusIssue(issue Issue) {
	a.view = "all"
	a.folder = "all"
	a.tag = ""
	a.search = ""
	switch issue.Type {
	case "breached":
		a.securityFilter = "breached"
	case "weak":
		a.securityFilter = "weak"
	default:
		a.securityFilter = "all"
	}
	if issue.Entry != nil {
		a.search = issue.Entry.Site
	}
	a.resetSelection()
	a.setPage("vault")
}

func (a *App) restoreModal(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(430).Gap(12).Children(func() {
		ui.Text(c, a.t("backup.restoreTitle")).FontSize(t.FontSize * 1.2).Bold()
		ui.Text(c, a.t("backup.restoreLine1")+a.restoreName).TextColor(t.TextMuted)
		ui.Text(c, a.t("backup.restoreWarn1")+a.t("backup.restoreOverwrite")+a.t("backup.restoreWarn2")).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)
		ui.Text(c, a.t("backup.restorePwHint")).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)
		pw := ui.TextInput(c, &a.restorePw).Password().Placeholder(a.t("backup.masterPassword"))
		if pw.Submitted() {
			a.restoreBackupAsync()
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("backup.cancel")).Clicked() {
				a.restoreOpen = false
			}
			if ui.PrimaryButton(c, a.t("backup.confirmRestore")).Clicked() {
				a.restoreBackupAsync()
			}
		})
	})
}

func (a *App) importModal(c *ui.Context) {
	t := c.Theme()
	duplicates := 0
	for _, e := range a.importList {
		for _, existing := range a.entries {
			if existing.Site == strings.TrimSpace(e.Site) && existing.Username == strings.TrimSpace(e.Username) && existing.URL == strings.TrimSpace(e.URL) {
				duplicates++
				break
			}
		}
	}
	ui.Column(c).Width(460).Gap(12).Children(func() {
		ui.Text(c, a.t("transfer.confirmImport")).FontSize(t.FontSize * 1.2).Bold()
		ui.Text(c, a.importSource).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)
		ui.Row(c).Gap(12).Children(func() {
			for _, item := range []struct {
				n     int
				label string
			}{{len(a.importList), a.t("transfer.importableEntries")}, {duplicates, a.t("transfer.duplicatesExisting")}} {
				box := ui.Column(c).Grow(1).Center().Padding(12).Gap(2).Radius(10).Background(t.Background).Border(1, t.Border)
				box.Children(func() {
					ui.Text(c, fmt.Sprint(item.n)).FontSize(t.FontSize * 1.6).Bold().TextColor(t.Accent)
					ui.Text(c, item.label).FontSize(t.FontSize * 0.82).TextColor(t.TextMuted)
				})
			}
		})
		ui.Text(c, a.t("transfer.importMode")).FontSize(t.FontSize * 0.86).Bold().TextColor(t.TextMuted)
		for i, item := range []struct{ title, desc string }{
			{a.t("transfer.mergeTitle"), a.t("transfer.mergeDesc")},
			{a.t("transfer.overwriteTitle"), a.t("transfer.overwriteDesc")},
		} {
			idx := i
			row := ui.Row(c).Padding(10).Gap(10).Radius(8).Border(1, t.Border).Cursor(ui.CursorPointer)
			if a.importMode == idx {
				row.Background(t.Selection).Border(1, t.Accent)
			} else if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.OnClick(func() { a.importMode = idx })
			row.Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, item.title).FontSize(t.FontSize * 0.92).Bold()
					ui.Text(c, item.desc).FontSize(t.FontSize * 0.82).TextColor(t.TextMuted)
				})
				if a.importMode == idx {
					ui.Icon(c, icon("check")).FontSize(15).TextColor(t.Accent)
				}
			})
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("transfer.cancel")).Clicked() {
				a.importOpen = false
			}
			if ui.PrimaryButton(c, a.t("transfer.confirmImport")).Clicked() {
				added, skipped, updated := a.importEntries(a.importList, a.importMode == 1)
				parts := []string{a.tf("transfer.importAdded", map[string]any{"n": added})}
				if updated > 0 {
					parts = append(parts, a.tf("transfer.importUpdated", map[string]any{"n": updated}))
				}
				if skipped > 0 {
					parts = append(parts, a.tf("transfer.importSkipped", map[string]any{"n": skipped}))
				}
				detail := strings.Join(parts, a.t("transfer.joinComma"))
				a.toast(a.tf("transfer.importDone", map[string]any{"detail": detail}))
				a.pushActivity("import", a.importSource, detail)
				a.importOpen = false
			}
		})
	})
}

func (a *App) backupDeleteModal(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(400).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("trash-2")).FontSize(18).TextColor(t.Danger)
			ui.Text(c, a.tf("backup.confirmDelete", map[string]any{"name": a.backupDeleteName})).Bold()
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("backup.cancel")).Clicked() {
				a.backupDeleteOpen = false
			}
			del := ui.Button(c, a.t("backup.deleteBackup"))
			del.Background(t.Danger).TextColor(t.AccentText)
			if del.Clicked() {
				a.backupDeleteOpen = false
				a.deleteBackupAsync(a.backupDeleteName)
			}
		})
	})
}

func (a *App) backupPasswordModal(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(430).Gap(12).Children(func() {
		ui.Text(c, a.t("transfer.importBinTitle")).FontSize(t.FontSize * 1.2).Bold()
		ui.Text(c, a.t("transfer.fileLabel")+a.backupName).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)
		ui.Text(c, a.t("transfer.backupPwHint")).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)
		pw := ui.TextInput(c, &a.backupPw).Password().Placeholder(a.t("transfer.backupPwPlaceholder"))
		if pw.Submitted() {
			a.decryptBackupAsync()
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("transfer.cancel")).Clicked() {
				a.backupPwOpen = false
			}
			if ui.PrimaryButton(c, a.t("transfer.decryptAndImport")).Clicked() {
				a.decryptBackupAsync()
			}
		})
	})
}
