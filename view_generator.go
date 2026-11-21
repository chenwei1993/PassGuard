package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewGeneratorPage(c *ui.Context) {
	// Any change of the options makes a new password.
	sig := fmt.Sprintf("%d|%.0f|%v|%v|%v|%v|%v|%v|%v",
		a.gen.Type, a.gen.Length,
		a.gen.Upper, a.gen.Lower, a.gen.Numbers, a.gen.Symbols,
		a.gen.ExcludeSimilar, a.gen.ExcludeAmbiguous, a.gen.ExcludeDuplicate)
	if sig != a.gen.Signature {
		a.gen.Signature = sig
		a.gen.Password = a.generatePassword()
	}

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(1180).Padding(24).Gap(18).Children(func() {
			a.generatorResult(c)
			ui.Row(c).Gap(18).AlignItems(ui.Start).Children(func() {
				ui.Box(c).Grow(1).Children(func() { a.generatorConfig(c) })
				ui.Column(c).Width(360).Gap(18).Children(func() {
					a.generatorStrength(c)
					a.generatorHistory(c)
					a.generatorTips(c)
				})
			})
		})
	})
}

func (a *App) generatorResult(c *ui.Context) {
	t := c.Theme()
	card := ui.Column(c)
	card.Padding(32)
	card.Gap(14)
	card.Radius(18)
	card.Background(t.Surface)
	card.Border(1, t.Border)
	card.Shadow(0, 1, 3, 0, a.themeShadow())
	card.Clip()
	card.Children(func() {
		ui.Box(c).Height(3).Absolute().Top(0).Left(0).Right(0).
			Gradient(ui.Hex("#6366f1"), ui.Hex("#10b981"), 0)
		// The password, each kind of character in its color.
		display := ui.RichText(c).Font("monospace").FontSize(t.FontSize*2).FontWeight(600).
			LetterSpacing(2).TextAlign(ui.Center).Padding(8, 16).Radius(14).
			Gradient(ui.RGBA(129, 140, 248, 0.08), ui.RGBA(168, 85, 247, 0.08), 135)
		display.Children(func() {
			if a.gen.Password == "" {
				ui.Text(c, a.t("gen.clickRegenerate")).TextColor(t.TextMuted)
				return
			}
			for _, r := range a.gen.Password {
				col := t.Text
				switch {
				case r >= 'A' && r <= 'Z':
					col = t.Accent
				case r >= '0' && r <= '9':
					col = t.Success
				case !(r >= 'a' && r <= 'z'):
					col = t.Warning
				}
				ui.Text(c, string(r)).TextColor(col)
			}
		})

		s := strengthFor(a.gen.Password, false)
		bar := ui.Box(c).FillWidth().Height(8).Radius(4).Background(t.Border)
		bar.Children(func() {
			fill := ui.Box(c).Absolute().Top(0).Left(0).Bottom(0).
				WidthPercent(float32(s.Score)).Radius(4).Background(strengthColor(t, s))
			_ = fill
		})
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(3, 9).Radius(999).Background(ui.RGBA(52, 211, 153, 0.12)).Children(func() {
				ui.Icon(c, icon("shield-check")).FontSize(12).TextColor(strengthColor(t, s))
				ui.Text(c, a.strengthLabel(s)).FontSize(t.FontSize * 0.8571).FontWeight(500).TextColor(strengthColor(t, s))
			})
			ui.Spacer(c)
			ui.Textf(c, "%s · %s", a.tf("gen.charsCount", map[string]any{"n": len([]rune(a.gen.Password))}),
				a.tf("gen.entropyCount", map[string]any{"n": int(s.Entropy)})).
				FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
		})
		ui.Row(c).Gap(8).Justify(ui.Start).Children(func() {
			if a.iconTextButton(c, "copy", a.t("gen.copyPassword"), true).Clicked() {
				a.copyText(a.gen.Password, a.t("common.field.password"))
				a.gen.History = pushHistory(a.gen.History, a.gen.Password)
			}
			if a.iconTextButton(c, "refresh-cw", a.t("gen.regenerate"), false).Clicked() {
				a.gen.Password = a.generatePassword()
				a.gen.History = pushHistory(a.gen.History, a.gen.Password)
			}
			if a.iconTextButton(c, "save", a.t("gen.saveToVault"), false).Clicked() {
				a.openAddEntry(a.gen.Password)
			}
		})
	})
}

func (a *App) generatorConfig(c *ui.Context) {
	t := c.Theme()
	card := a.card(c)
	card.Children(func() {
		a.segmented(c, []segment{
			{"dices", a.t("gen.typeRandom")},
			{"brain", a.t("gen.typeMemorable")},
			{"hash", a.t("gen.typePin")},
		}, a.gen.Type, func(i int) { a.gen.Type = i })

		ui.Row(c).AlignItems(ui.End).Margin(0, 0, 10, 0).Children(func() {
			ui.Text(c, a.t("gen.lengthLabel")).FontSize(t.FontSize * 0.9286).FontWeight(500)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%.0f %s", a.gen.Length, a.t("gen.lengthUnit"))).
				FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted)
		})
		ui.Slider(c, &a.gen.Length, 8, 64).Step(1)

		ui.Divider(c).Background(t.Border).Margin(6, 0, 0, 0)
		ui.Text(c, a.t("gen.charTypeTitle")).FontSize(t.FontSize * 0.9286).FontWeight(600).TextColor(t.TextMuted)
		ui.Grid(c).Columns(2).Gap(10).Children(func() {
			disabled := a.gen.Type == 2
			a.toggleCard(c, "Aa", a.t("gen.uppercase"), "A-Z", &a.gen.Upper, disabled)
			a.toggleCard(c, "ab", a.t("gen.lowercase"), "a-z", &a.gen.Lower, disabled)
			a.toggleCard(c, "12", a.t("gen.digits"), "0-9", &a.gen.Numbers, disabled)
			a.toggleCard(c, "!@", a.t("gen.symbols"), "!@#$%^&*", &a.gen.Symbols, disabled)
		})

		ui.Divider(c).Background(t.Border).Margin(6, 0, 0, 0)
		ui.Text(c, a.t("gen.excludeTitle")).FontSize(t.FontSize * 0.9286).FontWeight(600).TextColor(t.TextMuted)
		ui.Column(c).Gap(10).Children(func() {
			a.switchRow(c, "ban", a.t("gen.excludeSimilar"), &a.gen.ExcludeSimilar)
			a.switchRow(c, "shield-alert", a.t("gen.excludeAmbiguous"), &a.gen.ExcludeAmbiguous)
			a.switchRow(c, "repeat", a.t("gen.noRepeatedChars"), &a.gen.ExcludeDuplicate)
		})
	})
}

func (a *App) generatorStrength(c *ui.Context) {
	t := c.Theme()
	s := strengthFor(a.gen.Password, false)
	card := a.card(c)
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("activity")).FontSize(16).TextColor(t.Accent)
			ui.Text(c, a.t("gen.strengthDetailTitle")).FontSize(t.FontSize * 1.0).FontWeight(600)
		})
		ui.Column(c).Center().Children(func() {
			a.gauge(c, 140, 10, float32(s.Score)/100, func() {
				ui.Text(c, fmt.Sprint(s.Score)).FontSize(t.FontSize * 2.2857).FontWeight(700).TextColor(strengthColor(t, s))
				ui.Text(c, a.t("gen.scoreLabel")).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
			})
		})
		ui.Grid(c).Columns(2).Gap(12).Children(func() {
			for _, item := range [][2]string{
				{a.t("gen.crackTimeLabel"), a.crackTime(a.gen.Password)},
				{a.t("gen.entropyLabel"), fmt.Sprintf("%.0f bits", s.Entropy)},
			} {
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, item[0]).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
					ui.Text(c, item[1]).FontSize(t.FontSize * 0.9286).FontWeight(600)
				})
			}
		})
	})
}

func (a *App) generatorHistory(c *ui.Context) {
	t := c.Theme()
	card := a.card(c)
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("history")).FontSize(15).TextColor(t.Accent)
			ui.Text(c, a.t("gen.historyTitle")).FontSize(t.FontSize * 0.9286).FontWeight(600)
			ui.Spacer(c)
			if ui.Button(c, a.t("gen.clear")).Padding(3, 8).Radius(6).FontSize(t.FontSize * 0.8).Clicked() {
				a.gen.History = nil
			}
		})
		if len(a.gen.History) == 0 {
			ui.Text(c, a.t("gen.noHistory")).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted).PaddingY(6)
		}
		for _, p := range a.gen.History {
			p := p
			ui.Row(c).Padding(5, 0).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, maskPassword(p)).FontSize(t.FontSize * 0.8571).Font("monospace").Grow(1).SingleLine().TextColor(t.TextMuted)
				if a.iconButton(c, "copy", a.t("gen.copy")).Clicked() {
					a.copyText(p, a.t("common.field.password"))
				}
			})
		}
	})
}

func (a *App) generatorTips(c *ui.Context) {
	t := c.Theme()
	card := a.card(c)
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("lightbulb")).FontSize(15).TextColor(t.Warning)
			ui.Text(c, a.t("gen.tipsTitle")).FontSize(t.FontSize * 0.9286).FontWeight(600)
		})
		for _, tip := range [][2]string{
			{a.t("gen.tip1Title"), a.t("gen.tip1Rest")},
			{a.t("gen.tip2Title"), a.t("gen.tip2Rest")},
			{a.t("gen.tip3Title"), a.t("gen.tip3Rest")},
		} {
			ui.Row(c).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Icon(c, icon("circle-check")).FontSize(13).TextColor(t.Success)
				ui.RichText(c,
					ui.Span{Text: tip[0], Weight: 600},
					ui.Span{Text: tip[1], Color: t.TextMuted},
				).FontSize(t.FontSize * 0.8571)
			})
		}
	})
}

func pushHistory(list []string, p string) []string {
	if p == "" {
		return list
	}
	out := append([]string{p}, list...)
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func maskPassword(p string) string {
	r := []rune(p)
	if len(r) <= 8 {
		return p + "••••"
	}
	return string(r[:8]) + "•••••"
}
