package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewLock(c *ui.Context) {
	t := c.Theme()
	isSetup := a.phase == "setup"

	ui.Box(c).Fill().Background(t.Background).Children(func() {
		// The lock screen's soft glows and grid, from the design.
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).ZIndex(-1).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.5, Y: 0.3},
				Radius: 620,
				Stops:  []ui.GradientStop{{Color: ui.RGBA(129, 140, 248, 0.08), At: 0}, {Color: ui.Transparent, At: 1}},
			})
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).ZIndex(-1).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.2, Y: 0.8},
				Radius: 520,
				Stops:  []ui.GradientStop{{Color: ui.RGBA(168, 85, 247, 0.05), At: 0}, {Color: ui.Transparent, At: 1}},
			})
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).ZIndex(-1).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.8, Y: 0.7},
				Radius: 420,
				Stops:  []ui.GradientStop{{Color: ui.RGBA(99, 102, 241, 0.04), At: 0}, {Color: ui.Transparent, At: 1}},
			})
		grid := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).ZIndex(-1)
		grid.Draw(func(p *ui.Painter, r ui.Rect) {
			col := ui.RGBA(129, 140, 248, 0.03)
			for x := float32(0); x < r.W; x += 48 {
				p.Line(r.X+x, r.Y, r.X+x, r.Y+r.H, 1, col)
			}
			for y := float32(0); y < r.H; y += 48 {
				p.Line(r.X, r.Y+y, r.X+r.W, r.Y+y, 1, col)
			}
		})

		ui.Column(c).Fill().Center().Children(func() {
			card := ui.Column(c)
			card.Width(420)
			card.Padding(40, 36, 28)
			card.Gap(16)
			card.Radius(18)
			card.Background(t.Surface)
			card.Border(1, t.Border)
			card.Shadow(0, 24, 60, 0, a.themeShadow())
			card.Shadow(0, 8, 32, 0, ui.RGBA(129, 140, 248, 0.10))
			card.Children(func() {
				// Logo, glow and name
				ui.Column(c).Center().Children(func() {
					wrap := ui.Row(c).Width(80).Height(80).Center()
					wrap.Children(func() {
						ui.Box(c).Size(80, 80).Radius(40).ZIndex(-1).Absolute().Top(0).Left(0).
							RadialGradient(ui.RadialGradient{
								Center: ui.Point{X: 0.5, Y: 0.5},
								Radius: 40,
								Stops:  []ui.GradientStop{{Color: ui.RGBA(129, 140, 248, 0.35), At: 0}, {Color: ui.RGBA(129, 140, 248, 0.10), At: 0.4}, {Color: ui.Transparent, At: 1}},
							})
						logoIcon := ui.Box(c).Size(64, 64).Radius(16).Gradient(ui.Hex("#6366f1"), ui.Hex("#a855f7"), 135).Center()
						logoIcon.Shadow(0, 8, 24, 0, ui.RGBA(129, 140, 248, 0.35))
						logoIcon.Shadow(0, 2, 8, 0, ui.RGBA(168, 85, 247, 0.20))
						logoIcon.Children(func() {
							ui.Icon(c, icon("lock-keyhole")).FontSize(32).TextColor(ui.Hex("#ffffff"))
						})
					})
					ui.Text(c, a.t("app.name")).FontSize(t.FontSize*1.714).FontWeight(700).Margin(16, 0, 0, 0)
					subtitle := a.t("lock.lockedSubtitle")
					if isSetup {
						subtitle = a.t("lock.setupSubtitle")
					}
					ui.Text(c, subtitle).FontSize(t.FontSize*0.9286).TextColor(t.TextMuted).
						TextAlign(ui.Center).MaxWidth(340).Margin(6, 0, 0, 0)
				})

				if !isSetup && a.status.CanRemember {
					a.lockTabs(c)
				}
				if isSetup || a.lockTab == 0 {
					a.lockPasswordForm(c, isSetup)
				} else {
					a.lockHelloForm(c)
				}

				if !isSetup {
					ui.Checkbox(c, &a.remember, a.t("lock.autoUnlock")).FontSize(t.FontSize * 0.9286)
				}

				ui.Divider(c).Background(t.Border).Margin(8, 0, 0, 0)
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("database")).FontSize(12).TextColor(t.TextMuted)
						ui.Text(c, a.t("lock.aesEncrypted")).FontSize(t.FontSize * 0.8214).TextColor(t.TextMuted)
					})
					ui.Spacer(c)
					ui.Text(c, "v2.4.1").FontSize(t.FontSize * 0.8214).TextColor(t.TextMuted)
				})
			})
		})
	})
}

func (a *App) lockTabs(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Padding(3).Radius(10).Background(t.SurfaceHover).Children(func() {
		for i, item := range []struct {
			icon  string
			label string
		}{
			{"key", a.t("lock.masterPassword")},
			{"fingerprint-pattern", "Windows Hello"},
		} {
			idx := i
			active := a.lockTab == idx
			tab := ui.Row(c).Grow(1).Justify(ui.Center).Padding(8, 12).Gap(6).Radius(6).AlignItems(ui.Center).Cursor(ui.CursorPointer)
			tab.Label(item.label)
			switch {
			case active:
				tab.Background(t.Surface)
				tab.Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.10))
			case tab.Hovered():
				tab.Background(t.Surface)
			}
			tab.OnClick(func() { a.lockTab = idx })
			tab.Children(func() {
				col := t.TextMuted
				if active {
					col = t.Accent
				}
				ui.Icon(c, icon(item.icon)).FontSize(14).TextColor(col)
				ui.Text(c, item.label).FontSize(t.FontSize * 0.8929).FontWeight(500).TextColor(col)
			})
		}
	})
}

func (a *App) lockPasswordForm(c *ui.Context, isSetup bool) {
	t := c.Theme()
	ui.Column(c).Gap(16).Children(func() {
		a.lockMessages(c)

		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, a.t("lock.masterPassword")).FontSize(t.FontSize * 0.9286).FontWeight(500)
			a.lockPasswordField(c, &a.password, "lock", isSetup)
		})
		if isSetup {
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, a.t("lock.confirmPassword")).FontSize(t.FontSize * 0.9286).FontWeight(500)
				a.lockPasswordField(c, &a.confirm, "shield-check", false)
			})
		}

		label := a.t("lock.unlockVault")
		if isSetup {
			label = a.t("lock.createVault")
		}
		if a.busy {
			label = a.t("lock.verifying")
			if isSetup {
				label = a.t("lock.creating")
			}
		}
		cooldown := a.cooldownLeft() > 0
		btn := ui.Box(c).FillWidth().Height(46).Radius(10).Center().Cursor(ui.CursorPointer)
		btn.Gradient(ui.Hex("#6366f1"), ui.Hex("#a855f7"), 135)
		btn.Label(label)
		if a.busy || cooldown {
			btn.Opacity(0.55).IgnoreInput(true)
		}
		if btn.Hovered() && !a.busy && !cooldown {
			btn.Shadow(0, 6, 20, 0, ui.RGBA(129, 140, 248, 0.40))
		}
		btn.OnClick(func() { a.submitLock(isSetup) })
		btn.Children(func() {
			ui.Icon(c, icon("lock-open")).FontSize(17).TextColor(ui.Hex("#ffffff"))
			ui.Text(c, label).FontSize(t.FontSize).FontWeight(600).TextColor(ui.Hex("#ffffff"))
		})

		hint := a.t("lock.unlockHint")
		if isSetup {
			hint = a.t("lock.setupHint")
		}
		ui.Row(c).Justify(ui.Center).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(t.Success)
			ui.Text(c, hint).FontSize(t.FontSize * 0.8214).TextColor(t.TextMuted)
		})
	})
}

func (a *App) lockPasswordField(c *ui.Context, value *string, iconName string, setup bool) {
	t := c.Theme()
	bad := a.lastError != "" || a.failAttempts > 0 || a.cooldownLeft() > 0
	box := ui.Box(c).FillWidth().Height(44)
	box.Children(func() {
		in := ui.TextInput(c, value)
		in.FillWidth()
		in.Height(44)
		in.Padding(11, 44, 11, 40)
		in.Background(a.surface2())
		in.Border(1, t.Border)
		in.Radius(10)
		in.FontSize(t.FontSize)
		if bad && !setup {
			in.Border(1, t.Danger)
		}
		placeholder := a.t("lock.passwordPlaceholder")
		if setup {
			placeholder = a.t("lock.masterPlaceholder")
		}
		in.Placeholder(placeholder)
		if !a.showPassword {
			in.Password()
		}
		if in.Submitted() {
			a.submitLock(setup)
		}
		ui.Icon(c, icon(iconName)).FontSize(16).TextColor(t.TextMuted).Absolute().Left(14).Top(14)
		eye := ui.Box(c).Size(32, 32).Radius(6).Center().Cursor(ui.CursorPointer).Absolute().Right(6).Top(6)
		eye.Label(a.t("lock.showPassword"))
		if eye.Hovered() {
			eye.Background(a.surface3())
		}
		eye.OnClick(func() { a.showPassword = !a.showPassword })
		eye.Children(func() {
			name := "eye"
			if a.showPassword {
				name = "eye-off"
			}
			ui.Icon(c, icon(name)).FontSize(16).TextColor(t.TextMuted)
		})
	})
}

func (a *App) lockHelloForm(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Gap(14).Center().Children(func() {
		ui.Icon(c, icon("fingerprint-pattern")).FontSize(36).TextColor(t.Accent)
		ui.Text(c, "Windows Hello").FontSize(t.FontSize * 1.1).FontWeight(700)
		desc := a.t("lock.helloDesc")
		if a.settings.LockOnStartup {
			desc = a.t("lock.helloDisabledDesc")
		}
		ui.Text(c, desc).FontSize(t.FontSize * 0.8929).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(330)
		label := a.t("lock.startVerify")
		if a.busy {
			label = a.t("lock.verifying")
		}
		btn := ui.Box(c).FillWidth().Height(42).Radius(10).Center().Cursor(ui.CursorPointer)
		btn.Gradient(ui.Hex("#6366f1"), ui.Hex("#a855f7"), 135)
		btn.Label(label)
		if a.busy {
			btn.Opacity(0.55).IgnoreInput(true)
		}
		btn.OnClick(func() {
			if a.settings.LockOnStartup {
				a.toast(a.t("lock.helloDisabledToast"))
				a.lockTab = 0
				return
			}
			a.attemptHello()
		})
		btn.Children(func() {
			ui.Icon(c, icon("scan-face")).FontSize(16).TextColor(ui.Hex("#ffffff"))
			ui.Text(c, label).FontSize(t.FontSize).FontWeight(600).TextColor(ui.Hex("#ffffff"))
		})
		if a.lastError != "" {
			a.lockError(c, a.lastError, t.Danger)
		}
	})
}

func (a *App) lockMessages(c *ui.Context) {
	t := c.Theme()
	if secs := a.cooldownLeft(); secs > 0 {
		a.lockError(c, a.t("lock.tooManyAttempts")+" "+a.t("lock.retryBefore")+fmt.Sprint(secs)+a.t("lock.retryAfter"), t.Danger)
		c.After(250 * time.Millisecond)
		return
	}
	if a.failAttempts > 0 {
		a.lockError(c, a.t("lock.wrongPassword")+" "+a.t("lock.attemptsLeftBefore")+fmt.Sprint(a.failAttempts)+a.t("lock.attemptsLeftAfter"), t.Danger)
		return
	}
	if a.lastError != "" {
		a.lockError(c, a.lastError, t.Danger)
	}
}

func (a *App) lockError(c *ui.Context, msg string, color ui.Color) {
	t := c.Theme()
	row := ui.Row(c).Padding(10, 12).Gap(8).Radius(10).
		Background(ui.RGBA(248, 113, 113, 0.08)).
		Border(1, ui.RGBA(248, 113, 113, 0.20))
	row.Children(func() {
		ui.Icon(c, icon("circle-alert")).FontSize(16).TextColor(color)
		ui.Text(c, msg).FontSize(t.FontSize * 0.8929).TextColor(color).Grow(1)
	})
}

func (a *App) cooldownLeft() int {
	if a.cooldownUntil.IsZero() {
		return 0
	}
	d := time.Until(a.cooldownUntil)
	if d <= 0 {
		return 0
	}
	return int(d.Seconds()) + 1
}

func (a *App) submitLock(isSetup bool) {
	if a.busy {
		return
	}
	if isSetup {
		pw := a.password
		if len([]rune(pw)) < 8 {
			a.lastError = a.t("settings.toast.newPasswordTooShort")
			return
		}
		if pw != a.confirm {
			a.lastError = a.t("settings.toast.passwordMismatch")
			return
		}
		a.attemptCreate(pw)
		return
	}
	if a.cooldownLeft() > 0 || a.password == "" {
		return
	}
	a.attemptUnlock(a.password)
}
