package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewSettingsPage(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		nav := ui.Column(c).Width(200).FillHeight().Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Padding(10).Gap(2)
		nav.Children(func() {
			for i, item := range []struct{ icon, key string }{
				{"settings", "settings.nav.general"},
				{"shield-check", "settings.nav.security"},
				{"palette", "settings.nav.appearance"},
				{"info", "settings.nav.about"},
			} {
				idx := i
				a.settingsNavItem(c, item.icon, a.t(item.key), a.settingsTab == idx, func() { a.settingsTab = idx })
			}
		})
		ui.Scroll(c).Grow(1).Padding(24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(860).Gap(18).Children(func() {
				switch a.settingsTab {
				case 0:
					a.settingsGeneral(c)
				case 1:
					a.settingsSecurity(c)
				case 2:
					a.settingsAppearance(c)
				default:
					a.settingsAbout(c)
				}
			})
		})
	})
}

func (a *App) settingsNavItem(c *ui.Context, iconName, label string, active bool, onClick func()) {
	t := c.Theme()
	row := ui.Row(c).Gap(10).Padding(8, 10).Radius(8).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Label(label)
	if active {
		row.Background(t.Selection)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.OnClick(onClick)
	row.Children(func() {
		col := t.TextMuted
		if active {
			col = t.Accent
		}
		ui.Icon(c, icon(iconName)).FontSize(16).TextColor(col)
		ui.Text(c, label).FontSize(t.FontSize * 0.92).TextColor(t.Text)
	})
}

func (a *App) settingsGroup(c *ui.Context, iconName, title string) ui.Element {
	t := c.Theme()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(15).TextColor(t.Accent)
		ui.Text(c, title).FontSize(t.FontSize * 1.0).Bold()
	})
	return a.card(c)
}

func (a *App) settingsGeneral(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, a.t("settings.nav.general")).FontSize(t.FontSize * 1.4).Bold()
	ui.Text(c, a.t("settings.general.desc")).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)

	// Appearance
	a.settingsGroup(c, "monitor", a.t("settings.group.appearance")).Children(func() {
		themeKeys := []string{"light", "dark", "system"}
		themeLabels := []string{
			a.t("settings.theme.light"),
			a.t("settings.theme.dark"),
			a.t("settings.theme.system"),
		}
		themeChoice := 0
		for i, key := range themeKeys {
			if key == a.settings.ThemeMode {
				themeChoice = i
			}
		}
		a.settingsRow(c, a.t("settings.appearance.themeMode"), a.t("settings.appearance.themeModeDesc"), func() {
			chooseOption(c, themeLabels, themeChoice, 130, func(i int) { a.setThemeMode(themeKeys[i]) })
		})
		fontKeys := []string{"small", "normal", "large"}
		fontLabels := []string{
			a.t("settings.fontSize.small"),
			a.t("settings.fontSize.normal"),
			a.t("settings.fontSize.large"),
		}
		fontChoice := 1
		for i, key := range fontKeys {
			if key == a.settings.FontSize {
				fontChoice = i
			}
		}
		a.settingsRow(c, a.t("settings.appearance.fontSize"), a.t("settings.appearance.fontSizeDesc"), func() {
			chooseOption(c, fontLabels, fontChoice, 130, func(i int) { a.setFontSize(fontKeys[i]) })
		})
		localeKeys := []string{"zh-Hans", "en"}
		localeLabels := []string{"简体中文", "English"}
		localeChoice := 0
		if localeFile(a.locale) == "en" {
			localeChoice = 1
		}
		a.settingsRow(c, a.t("settings.appearance.language"), a.t("settings.appearance.languageDesc"), func() {
			chooseOption(c, localeLabels, localeChoice, 130, func(i int) { a.setLocale(localeKeys[i]) })
		})
	})

	// Lock
	a.settingsGroup(c, "lock", a.t("settings.group.lock")).Children(func() {
		type minOpt struct {
			label string
			min   int
		}
		minOpts := []minOpt{
			{a.t("settings.lock.never"), 0},
			{a.t("settings.lock.1m"), 1},
			{a.t("settings.lock.5m"), 5},
			{a.t("settings.lock.15m"), 15},
			{a.t("settings.lock.30m"), 30},
			{a.t("settings.lock.1h"), 60},
		}
		autoChoice := minOpts[2].label
		labels := make([]string, len(minOpts))
		for i, o := range minOpts {
			labels[i] = o.label
			if o.min == a.settings.AutoLockMinutes {
				autoChoice = o.label
			}
		}
		a.settingsRow(c, a.t("settings.lock.autoLockTime"), a.t("settings.lock.autoLockTimeDesc"), func() {
			sel := ui.Select(c, &autoChoice, labels)
			sel.Width(130)
			if sel.Changed() {
				for _, o := range minOpts {
					if o.label == autoChoice {
						a.settings.AutoLockMinutes = o.min
					}
				}
				a.saveSettings()
				a.startIdleTimer()
			}
		})
		a.settingsRow(c, a.t("settings.lock.clearClipboardOnLock"), a.t("settings.lock.clearClipboardOnLockDesc"), func() {
			sw := ui.Switch(c, &a.settings.ClipClear)
			if sw.Changed() {
				a.saveSettings()
			}
		})
		a.settingsRow(c, a.t("settings.lock.autoClearClipboard"), a.t("settings.lock.autoClearClipboardDesc"), func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				on := a.settings.ClipClearSecs > 0
				sw := ui.Switch(c, &on)
				if sw.Changed() {
					if on {
						a.settings.ClipClearSecs = 30
					} else {
						a.settings.ClipClearSecs = 0
					}
					a.saveSettings()
				}
				type secOpt struct {
					label string
					secs  int
				}
				secOpts := []secOpt{
					{a.t("settings.clip.15s"), 15},
					{a.t("settings.clip.30s"), 30},
					{a.t("settings.clip.60s"), 60},
					{a.t("settings.clip.2m"), 120},
				}
				secLabels := make([]string, len(secOpts))
				secChoice := secOpts[1].label
				for i, o := range secOpts {
					secLabels[i] = o.label
					if o.secs == a.settings.ClipClearSecs {
						secChoice = o.label
					}
				}
				sel := ui.Select(c, &secChoice, secLabels)
				sel.Width(110).Disabled(!on)
				if sel.Changed() {
					for _, o := range secOpts {
						if o.label == secChoice {
							a.settings.ClipClearSecs = o.secs
						}
					}
					a.saveSettings()
				}
			})
		})
		a.settingsRow(c, a.t("settings.lock.lockOnMinimize"), a.t("settings.lock.lockOnMinimizeDesc"), func() {
			sw := ui.Switch(c, &a.settings.LockOnMinimize)
			if sw.Changed() {
				a.saveSettings()
			}
		})
		a.settingsRow(c, a.t("settings.lock.lockOnStartup"), a.t("settings.lock.lockOnStartupDesc"), func() {
			lockOn := a.settings.LockOnStartup
			sw := ui.Switch(c, &lockOn)
			if sw.Changed() {
				a.settings.LockOnStartup = lockOn
				a.saveSettings()
				if lockOn {
					_ = a.vault.SetRemember(false)
					a.toast(a.t("settings.toast.autoUnlockDisabled"))
				} else {
					_ = a.vault.SetRemember(true)
					a.toast(a.t("settings.toast.autoUnlockEnabled"))
				}
			}
		})
	})

	// Data
	a.settingsGroup(c, "hard-drive", a.t("settings.group.data")).Children(func() {
		a.settingsRow(c, a.t("settings.data.dbLocation"), a.t("settings.data.dbLocationDesc"), func() {
			path := a.dbInfo.Dir
			if a.dbInfo.OK && a.dbInfo.File != nil {
				if s, ok := a.dbInfo.File.(string); ok {
					path = s
				}
			}
			if path == "" {
				path = a.vault.dataDir()
			}
			ui.Text(c, path).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted).MaxWidth(360).SingleLine()
		})
		ui.Row(c).Gap(8).PaddingY(8).Children(func() {
			if a.iconTextButton(c, "folder-open", a.t("settings.data.openFolder"), false).Clicked() {
				a.openDataFolder()
			}
			if a.iconTextButton(c, "move-horizontal", a.t("settings.data.changeLocation"), false).Clicked() {
				a.toast(a.t("settings.toast.moveLocationTodo"))
			}
			if a.iconTextButton(c, "download", a.t("settings.data.exportFile"), false).Clicked() {
				a.exportFileAsync()
			}
			if a.iconTextButton(c, "wrench", a.t("settings.data.repair"), false).Clicked() {
				if r := a.vault.Repair(); r.OK {
					a.toast(r.Message)
				} else {
					a.toast(r.Message)
				}
			}
		})
		a.settingsRow(c, a.t("settings.data.dbSize"), a.t("settings.data.dbSizeDesc"), func() {
			ui.Text(c, a.fmtSize(a.dbInfo.SizeBytes)).FontSize(t.FontSize * 0.92).Bold()
		})
		a.settingsRow(c, a.t("settings.data.encryption"), a.t("settings.data.encryptionDesc"), func() {
			ui.Text(c, "AES-256-GCM · scrypt").FontSize(t.FontSize * 0.88).Bold().TextColor(t.Success)
		})
	})

	// Notifications
	a.settingsGroup(c, "bell", a.t("settings.group.notifications")).Children(func() {
		for _, item := range []struct {
			label, desc string
			value       *bool
		}{
			{a.t("settings.notif.expiry"), a.t("settings.notif.expiryDesc"), &a.settings.NotifyExpiry},
			{a.t("settings.notif.weak"), a.t("settings.notif.weakDesc"), &a.settings.NotifyWeak},
			{a.t("settings.notif.breach"), a.t("settings.notif.breachDesc"), &a.settings.NotifyBreach},
			{a.t("settings.notif.backup"), a.t("settings.notif.backupDesc"), &a.settings.NotifyBackup},
		} {
			item := item
			a.settingsRow(c, item.label, item.desc, func() {
				sw := ui.Switch(c, item.value)
				if sw.Changed() {
					a.saveSettings()
				}
			})
		}
	})

	// Shortcuts
	a.settingsGroup(c, "keyboard", a.t("settings.group.shortcuts")).Children(func() {
		for _, item := range []struct{ label, desc, keys string }{
			{a.t("settings.shortcut.lockVault"), a.t("settings.shortcut.lockVaultDesc"), "Ctrl + L"},
			{a.t("settings.shortcut.addEntry"), a.t("settings.shortcut.addEntryDesc"), "Ctrl + N"},
			{a.t("settings.shortcut.search"), a.t("settings.shortcut.searchDesc"), "Ctrl + F"},
		} {
			a.settingsRow(c, item.label, item.desc, func() {
				ui.Text(c, item.keys).Font("monospace").FontSize(t.FontSize * 0.82).TextColor(t.TextMuted)
			})
		}
	})
}

func (a *App) settingsSecurity(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, a.t("settings.nav.security")).FontSize(t.FontSize * 1.4).Bold()
	ui.Text(c, a.t("settings.security.desc")).TextColor(t.TextMuted).FontSize(t.FontSize * 0.9)

	a.settingsGroup(c, "key-round", a.t("settings.group.masterPassword")).Children(func() {
		ui.Text(c, a.t("settings.security.changePasswordDesc")).FontSize(t.FontSize * 0.86).TextColor(t.TextMuted)
		ui.Column(c).Gap(10).MaxWidth(420).Children(func() {
			a.passwordField(c, a.t("settings.security.currentPassword"), &a.curPw, &a.showCurPw, a.t("settings.security.currentPasswordPlaceholder"))
			a.passwordField(c, a.t("settings.security.newPassword"), &a.newPw, &a.showNewPw, a.t("settings.security.newPasswordPlaceholder"))
			if a.newPw != "" {
				a.strengthRow(c, strengthFor(a.newPw, false))
			}
			a.passwordField(c, a.t("settings.security.confirmPassword"), &a.confirmPw, &a.showConfirmPw, a.t("settings.security.confirmPasswordPlaceholder"))
			label := a.t("settings.security.updatePassword")
			if a.busy {
				label = a.t("settings.security.reencrypting")
			}
			if a.iconTextButton(c, "save", label, true).Disabled(a.busy).Clicked() {
				a.changeMasterPasswordAsync()
			}
		})
	})

	a.settingsGroup(c, "fingerprint-pattern", a.t("settings.group.twofa")).Children(func() {
		ui.Text(c, a.t("settings.security.unlockMethodsDesc")).FontSize(t.FontSize * 0.86).TextColor(t.TextMuted)
		for _, item := range []struct{ icon, title, desc string }{
			{"fingerprint-pattern", a.t("settings.security.biometric"), a.t("settings.security.biometricDesc")},
			{"key", a.t("settings.security.yubikey"), a.t("settings.security.yubikeyDesc")},
		} {
			row := ui.Row(c).Padding(10).Gap(10).Radius(9).Border(1, t.Border).Opacity(0.75)
			row.Children(func() {
				ui.Icon(c, icon(item.icon)).FontSize(18).TextColor(t.Accent)
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Text(c, item.title).FontSize(t.FontSize * 0.9).Bold()
					ui.Text(c, item.desc).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
				})
			})
		}
	})

	// Danger zone
	danger := ui.Column(c).Padding(16).Gap(10).Radius(12).Background(t.Surface).Border(1, ui.RGBA(248, 113, 113, 0.35))
	danger.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("alert-triangle")).FontSize(15).TextColor(t.Danger)
			ui.Text(c, a.t("settings.group.danger")).FontSize(t.FontSize).Bold().TextColor(t.Danger)
		})
		a.settingsRow(c, a.t("settings.danger.resetAll"), a.t("settings.danger.resetAllDesc"), func() {
			if ui.Button(c, a.t("settings.danger.resetButton")).Clicked() {
				a.settings = defaultSettings()
				a.saveSettings()
				a.setThemeMode("light")
				a.setLocale("zh-Hans")
				a.toast(a.t("settings.toast.resetDone"))
			}
		})
		a.settingsRow(c, a.t("settings.danger.deleteAll"), a.t("settings.danger.deleteAllDesc"), func() {
			armed := time.Since(a.deleteArmedAt) < 5*time.Second && !a.deleteArmedAt.IsZero()
			b := ui.Button(c, a.t("settings.danger.deleteAllButton"))
			if armed {
				b = ui.Button(c, a.t("settings.danger.confirmDelete"))
				b.Background(t.Danger).TextColor(t.AccentText)
			}
			if b.Clicked() {
				if armed {
					a.deleteArmedAt = time.Time{}
					a.deleteAllVault()
					a.settingsTab = 0
					a.page = "vault"
				} else {
					a.deleteArmedAt = time.Now()
				}
			}
		})
	})
}

func (a *App) passwordField(c *ui.Context, label string, value *string, shown *bool, placeholder string) {
	t := c.Theme()
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
		ui.Row(c).Gap(6).Children(func() {
			in := ui.TextInput(c, value)
			in.Grow(1)
			in.Placeholder(placeholder)
			if !*shown {
				in.Password()
			}
			if in.Submitted() && a.phase == "unlocked" && a.settingsTab == 1 {
				// Enter in one of these fields updates the password.
				a.changeMasterPasswordAsync()
			}
			b := ui.Button(c, "").Label(a.t("modal.show")).Tooltip(a.t("modal.show"))
			b.Width(32).Height(30).Padding(0)
			if b.Clicked() {
				*shown = !*shown
			}
			b.Children(func() {
				name := "eye"
				if *shown {
					name = "eye-off"
				}
				ui.Icon(c, icon(name)).FontSize(15).TextColor(t.TextMuted)
			})
		})
	})
}

func (a *App) settingsAppearance(c *ui.Context) {
	ui.Text(c, a.t("settings.nav.appearance")).FontSize(c.Theme().FontSize * 1.4).Bold()
	a.emptyState(c, "construction", a.t("settings.nav.appearance"), a.t("settings.placeholder.desc"))
}

func (a *App) settingsAbout(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, a.t("settings.nav.about")).FontSize(t.FontSize * 1.4).Bold()
	a.settingsGroup(c, "info", a.t("app.name")).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			box := ui.Box(c).Size(56, 56).Radius(15).Gradient(ui.Hex("#6366f1"), ui.Hex("#a855f7"), 135).Center()
			box.Children(func() { ui.Icon(c, icon("lock-keyhole")).FontSize(26).TextColor(ui.Hex("#ffffff")) })
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, a.t("app.name")).FontSize(t.FontSize * 1.2).Bold()
				ui.Text(c, "v2.4.1").FontSize(t.FontSize * 0.86).TextColor(t.TextMuted)
				ui.Text(c, "AES-256-GCM · scrypt · "+a.t("vault.localStorage")).FontSize(t.FontSize * 0.82).TextColor(t.TextMuted)
			})
		})
		ui.Divider(c).Background(t.Border)
		a.settingsRow(c, a.t("settings.data.dbLocation"), "", func() {
			dir := a.dbInfo.Dir
			if dir == "" {
				dir = a.vault.dataDir()
			}
			ui.Text(c, dir).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted).MaxWidth(400).SingleLine()
		})
		ui.Row(c).Gap(8).Children(func() {
			if a.iconTextButton(c, "folder-open", a.t("settings.data.openFolder"), false).Clicked() {
				a.openDataFolder()
			}
		})
	})
}
