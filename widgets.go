package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

type segment struct {
	icon  string
	label string
}

func (a *App) segmented(c *ui.Context, items []segment, selected int, onSelect func(int)) ui.Element {
	t := c.Theme()
	box := ui.Row(c).Padding(3).Radius(10).Background(a.surface3())
	box.Children(func() {
		for i, item := range items {
			idx, it := i, item
			active := idx == selected
			tab := ui.Row(c).Grow(1).Justify(ui.Center).Padding(8, 12).Gap(6).Radius(6).
				AlignItems(ui.Center).Cursor(ui.CursorPointer)
			tab.Label(it.label)
			switch {
			case active:
				tab.Background(t.Surface)
				tab.Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.10))
			case tab.Hovered():
				tab.Background(t.Surface)
			}
			tab.OnClick(func() { onSelect(idx) })
			tab.Children(func() {
				col := t.TextMuted
				if active {
					col = t.Accent
				} else if tab.Hovered() {
					col = t.Text
				}
				if it.icon != "" {
					ui.Icon(c, icon(it.icon)).FontSize(14).TextColor(col)
				}
				ui.Text(c, it.label).FontSize(t.FontSize * 0.9286).FontWeight(500).TextColor(col)
			})
		}
	})
	return box
}

func (a *App) toggleCard(c *ui.Context, glyph, label, sub string, value *bool, disabled bool) {
	t := c.Theme()
	card := ui.Row(c).Padding(10, 12).Gap(10).Radius(10).Border(1, t.Border).
		Background(t.Surface).AlignItems(ui.Center)
	if *value && !disabled {
		card.Border(1, t.Accent)
		card.Background(ui.RGBA(99, 102, 241, 0.06))
	}
	if disabled {
		card.Opacity(0.5)
	}
	card.Children(func() {
		box := ui.Box(c).Size(32, 32).Radius(8).Background(a.surface3()).Center()
		box.Children(func() {
			col := t.TextMuted
			if *value && !disabled {
				col = t.Accent
			}
			ui.Text(c, glyph).FontSize(12).FontWeight(700).TextColor(col)
		})
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, label).FontSize(t.FontSize * 0.9286).FontWeight(500)
			ui.Text(c, sub).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
		})
		sw := ui.Switch(c, value)
		sw.Label(label)
		sw.Disabled(disabled)
	})
}

func (a *App) switchRow(c *ui.Context, iconName, label string, value *bool) {
	t := c.Theme()
	row := ui.Row(c).Padding(8, 0).Gap(8).AlignItems(ui.Center)
	row.Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(14).TextColor(t.TextMuted)
		ui.Text(c, label).FontSize(t.FontSize * 0.9286).Grow(1)
		sw := ui.Switch(c, value)
		sw.Label(label)
	})
}

// gauge draws a circular progress ring with content inside it.
func (a *App) gauge(c *ui.Context, size, stroke, progress float32, center func()) ui.Element {
	t := c.Theme()
	box := ui.Box(c).Size(size, size)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		radius := r.W/2 - stroke/2
		var track ui.Path
		track.Circle(cx, cy, radius)
		p.StrokePath(&track, stroke, t.Border)
		if progress <= 0 {
			return
		}
		if progress > 1 {
			progress = 1
		}
		var arc ui.Path
		start := float32(-math.Pi / 2)
		end := start + 2*math.Pi*progress
		const steps = 96
		for i := 0; i <= steps; i++ {
			angle := start + (end-start)*float32(i)/steps
			x := cx + radius*float32(math.Cos(float64(angle)))
			y := cy + radius*float32(math.Sin(float64(angle)))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		arc.Cap(ui.RoundCap)
		p.StrokePathGradient(&arc, stroke, ui.LinearGradient{
			From: ui.Hex("#6366f1"), To: ui.Hex("#a855f7"), Angle: 90,
		})
	})
	box.Children(func() {
		ui.Column(c).Fill().Center().Children(center)
	})
	return box
}

// ---------- small components shared by the views ----------

func (a *App) iconButton(c *ui.Context, name, label string) ui.Element {
	t := c.Theme()
	b := ui.Button(c, "")
	b.Label(label)
	b.Tooltip(label)
	b.Width(32)
	b.Height(30)
	b.Padding(0)
	b.Radius(8)
	b.Background(ui.Transparent)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Icon(c, icon(name)).FontSize(17).TextColor(t.TextMuted)
	})
	return b
}

func (a *App) iconTextButton(c *ui.Context, name, label string, primary bool) ui.Element {
	var b ui.Element
	if primary {
		b = ui.PrimaryButton(c, "")
	} else {
		b = ui.Button(c, "")
	}
	b.Label(label)
	b.Padding(9, 18)
	b.Radius(10)
	b.FontSize(c.Theme().FontSize * 0.9643)
	b.Children(func() {
		ui.Icon(c, icon(name)).FontSize(15)
		ui.Text(c, label).SingleLine()
	})
	return b
}

func (a *App) card(c *ui.Context) ui.Element {
	t := c.Theme()
	e := ui.Column(c)
	e.Padding(20)
	e.Gap(12)
	e.Radius(14)
	e.Background(t.Surface)
	e.Border(1, t.Border)
	e.Shadow(0, 1, 3, 0, a.themeShadow())
	return e
}

func (a *App) sectionTitle(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(c.Theme().FontSize * 1.05).Bold()
}

func (a *App) divider(c *ui.Context) {
	ui.Divider(c).Background(c.Theme().Border)
}

// chooseOption shows a drop-down of labels, with the item current chosen,
// and calls onChange with the index of a new choice. It keeps the settings
// and backup forms to one kind of control, as the Electron app's selects
// were.
func chooseOption(c *ui.Context, labels []string, current int, width float32, onChange func(int)) ui.Element {
	choice := ""
	if current >= 0 && current < len(labels) {
		choice = labels[current]
	}
	sel := ui.Select(c, &choice, labels).Width(width)
	if sel.Changed() {
		for i, label := range labels {
			if label == choice {
				onChange(i)
				break
			}
		}
	}
	return sel
}

func (a *App) tagPill(c *ui.Context, tag string, active bool, onClick func()) ui.Element {
	t := c.Theme()
	b := ui.Button(c, tag)
	b.Padding(3, 9)
	b.Radius(999)
	b.FontSize(t.FontSize * 0.82)
	if active {
		b.Background(t.Selection)
		b.Border(1, t.Accent)
		b.TextColor(t.Accent)
	} else {
		b.Background(ui.Transparent)
		b.Border(1, t.Border)
		b.TextColor(t.TextMuted)
	}
	b.OnClick(onClick)
	return b
}

func avatarGradient(seed string) (ui.Color, ui.Color) {
	_ = seed
	return ui.Hex("#6366f1"), ui.Hex("#a855f7")
}

func initials(site string) string {
	s := strings.TrimSpace(site)
	if s == "" {
		return "?"
	}
	r := []rune(s)
	if len(r) >= 2 {
		return strings.ToUpper(string(r[:2]))
	}
	return strings.ToUpper(string(r))
}

func (a *App) avatar(c *ui.Context, site string, size float32) ui.Element {
	c1, c2 := avatarGradient(site)
	box := ui.Box(c)
	box.Size(size, size)
	box.Radius(size * 0.28)
	box.Gradient(c1, c2, 135)
	box.Center()
	box.Children(func() {
		ui.Text(c, initials(site)).
			FontSize(size * 0.34).
			Bold().
			TextColor(ui.Hex("#ffffff"))
	})
	return box
}

func strengthColor(t *ui.Theme, s Strength) ui.Color {
	switch s.Level {
	case "breached":
		return t.Danger
	case "weak":
		return t.Danger
	case "medium":
		return t.Warning
	default:
		return t.Success
	}
}

func (a *App) strengthBars(c *ui.Context, s Strength) {
	t := c.Theme()
	col := strengthColor(t, s)
	ui.Row(c).Gap(2).Children(func() {
		for i := 1; i <= 5; i++ {
			bar := ui.Box(c).Width(13).Height(3).Radius(2)
			if i <= s.Bars {
				bar.Background(col)
			} else {
				bar.Background(t.Border)
			}
		}
	})
}

func (a *App) strengthRow(c *ui.Context, s Strength) {
	t := c.Theme()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		a.strengthBars(c, s)
		ui.Text(c, a.strengthLabel(s)).FontSize(t.FontSize * 0.86).TextColor(strengthColor(t, s))
	})
}

func (a *App) mutedText(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).TextColor(c.Theme().TextMuted)
}

func (a *App) emptyState(c *ui.Context, iconName, title, desc string) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(8).Padding(40).Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(36).TextColor(t.TextMuted)
		if title != "" {
			ui.Text(c, title).FontSize(t.FontSize * 1.07).Bold()
		}
		if desc != "" {
			ui.Text(c, desc).FontSize(t.FontSize * 0.93).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(360)
		}
	})
}

// settingsRow builds a labeled setting with its control on the right.
func (a *App) settingsRow(c *ui.Context, label, desc string, control func()) {
	t := c.Theme()
	ui.Box(c).Padding(10, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).PaddingY(10).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, label).FontSize(t.FontSize * 1.0)
				if desc != "" {
					ui.Text(c, desc).FontSize(t.FontSize * 0.86).TextColor(t.TextMuted)
				}
			})
			ui.Box(c).Shrink(0).Children(control)
		})
	})
}

func (a *App) folderIcon(folder string) string {
	switch folder {
	case "工作":
		return "briefcase"
	case "个人":
		return "user"
	case "财务":
		return "credit-card"
	case "社交":
		return "users"
	case "购物":
		return "shopping-bag"
	default:
		return "folder"
	}
}

func (a *App) sortLabel(sort string) string {
	switch sort {
	case "name":
		return a.t("vault.sortName")
	case "strength":
		return a.t("vault.sortStrength")
	default:
		return a.t("vault.sortRecent")
	}
}

func (a *App) pageTitle() string {
	switch a.page {
	case "vault":
		return a.t("vault.title")
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
	return a.t("vault.title")
}

func (a *App) pageIcon() string {
	switch a.page {
	case "generator":
		return "key-round"
	case "security":
		return "shield-check"
	case "transfer":
		return "arrow-down-up"
	case "backup":
		return "hard-drive"
	case "settings":
		return "settings"
	default:
		return "lock"
	}
}

func (a *App) fmtSize(bytes int64) string {
	switch {
	case bytes <= 0:
		return "0 B"
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.2f MB", float64(bytes)/1024/1024)
	}
}

func (a *App) autoLockText() string {
	if a.settings.AutoLockMinutes <= 0 {
		return a.t("core.autoLockOff")
	}
	return a.tf("core.autoLockIn", map[string]any{"min": a.settings.AutoLockMinutes})
}
