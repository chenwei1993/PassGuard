package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewSecurityPage(c *ui.Context) {
	t := c.Theme()
	issues := computeIssues(a.entries)
	stats := issueStats(issues)
	empty := len(a.entries) == 0
	score := 0
	if !empty {
		score = securityScore(issues)
	}
	rating := a.t("security.rating.none")
	ratingColor := t.TextMuted
	infoTitle := a.t("security.info.empty")
	switch {
	case empty:
	case score >= 80:
		rating = a.t("security.rating.excellent")
		ratingColor = t.Success
		infoTitle = a.t("security.info.excellent")
	case score >= 60:
		rating = a.t("security.rating.good")
		ratingColor = t.Success
		infoTitle = a.t("security.info.good")
	case score >= 40:
		rating = a.t("security.rating.fair")
		ratingColor = t.Warning
		infoTitle = a.t("security.info.improve")
	default:
		rating = a.t("security.rating.poor")
		ratingColor = t.Danger
		infoTitle = a.t("security.info.improve")
	}

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(1180).Padding(24).Gap(18).Children(func() {
			// Score
			card := a.card(c)
			card.Padding(28)
			card.Children(func() {
				ui.Row(c).Gap(32).AlignItems(ui.Center).Children(func() {
					a.gauge(c, 160, 12, float32(score)/100, func() {
						ui.Text(c, fmt.Sprint(score)).FontSize(t.FontSize * 3.4286).FontWeight(800).TextColor(t.Accent)
						ui.Text(c, a.t("security.scoreLabel")).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
						ui.Text(c, rating).FontSize(t.FontSize * 0.8571).FontWeight(600).TextColor(ratingColor)
					})
					ui.Column(c).Grow(1).Gap(12).Children(func() {
						ui.Text(c, infoTitle).FontSize(t.FontSize * 1.2857).FontWeight(700)
						desc := a.t("security.score.descEmpty")
						if !empty {
							desc = a.tf("security.score.descSummary", map[string]any{"total": len(a.entries), "issues": len(issues)})
						}
						ui.Text(c, desc).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted)
						ui.Row(c).Gap(12).Wrap().Children(func() {
							for _, st := range []struct {
								icon  string
								n     int
								label string
								color ui.Color
							}{
								{"shield-alert", stats["weak"], a.t("issue.weak"), t.Danger},
								{"copy", stats["duplicate"], a.t("issue.duplicate"), t.Warning},
								{"database-zap", stats["breached"], a.t("issue.breached"), t.Danger},
								{"clock-alert", stats["expired"], a.t("issue.expired"), infoColor(t)},
							} {
								box := ui.Column(c).Padding(12, 14).Gap(3).Radius(10).Background(t.Background).Border(1, t.Border).MinWidth(112)
								box.Children(func() {
									ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Icon(c, icon(st.icon)).FontSize(16).TextColor(st.color)
										ui.Text(c, fmt.Sprint(st.n)).FontSize(t.FontSize * 1.1429).FontWeight(700).TextColor(st.color)
									})
									ui.Text(c, st.label).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
								})
							}
						})
					})
				})
			})

			// Issues
			issueCard := a.card(c)
			issueCard.Children(func() {
				ui.Tabs(c, &a.secTab,
					fmt.Sprintf("%s  %d", a.t("security.tab.all"), len(issues)),
					fmt.Sprintf("%s  %d", a.t("issue.weak"), stats["weak"]),
					fmt.Sprintf("%s  %d", a.t("issue.duplicate"), stats["duplicate"]),
					fmt.Sprintf("%s  %d", a.t("issue.breached"), stats["breached"]),
					fmt.Sprintf("%s  %d", a.t("issue.expired"), stats["expired"]),
				)
				filtered := issues
				if a.secTab > 0 {
					filtered = nil
					kind := []string{"", "weak", "duplicate", "breached", "expired"}[a.secTab]
					for _, i := range issues {
						if i.Type == kind {
							filtered = append(filtered, i)
						}
					}
				}
				if len(filtered) == 0 {
					msg := a.t("security.issues.emptyTab")
					if empty {
						msg = a.t("security.issues.emptyVault")
					}
					ui.Text(c, msg).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted).PaddingY(18).TextAlign(ui.Center)
				}
				for _, raw := range filtered {
					issue := raw
					row := ui.Row(c).Padding(8, 4).Gap(12).AlignItems(ui.Center)
					row.BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
					row.Children(func() {
						site := a.t("common.unnamed")
						sub := "—"
						if issue.Entry != nil {
							site = issue.Entry.Site
							if issue.Entry.Username != "" {
								sub = issue.Entry.Username
							} else if issue.Entry.URL != "" {
								sub = issue.Entry.URL
							}
						}
						a.avatar(c, site, 36)
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, site).FontSize(t.FontSize * 0.9286).FontWeight(600).SingleLine()
							ui.Text(c, sub).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted).SingleLine()
						})
						a.issueTag(c, issue.Type, issue.Risk)
						if ui.Button(c, a.t("security.fix")).Padding(3, 10).Radius(6).FontSize(t.FontSize * 0.82).Clicked() {
							a.focusIssue(issue)
						}
					})
				}
			})

			// Suggestions
			sugCard := a.card(c)
			sugCard.Padding(20)
			sugCard.Children(func() {
				ui.Text(c, a.t("security.suggestionsTitle")).FontSize(t.FontSize * 1.0714).FontWeight(700)
				ui.Text(c, a.t("security.suggestionsSubtitle")).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)

				type sug struct {
					icon, tone, name, desc, status, action string
					pct                                    float32
					done, disabled                         bool
					act                                    func()
				}
				pct := func(n int) float32 {
					if len(a.entries) == 0 {
						return 100
					}
					v := float32(100 - (float64(n)/float64(len(a.entries)))*100)
					if v < 0 {
						v = 0
					}
					return v
				}
				rows := []sug{
					{icon: "shield-alert", tone: "danger", name: a.t("security.sug.weakName"),
						desc:   a.tf("security.sug.weakDesc", map[string]any{"n": stats["weak"]}),
						pct:    pct(stats["weak"]),
						status: a.tf("security.sug.weakStatus", map[string]any{"n": stats["weak"]}),
						action: a.t("security.sug.viewWeak"), disabled: stats["weak"] == 0,
						act: func() {
							a.securityFilter = "weak"
							a.view = "all"
							a.resetSelection()
							a.setPage("vault")
						}},
					{icon: "copy", tone: "warning", name: a.t("security.sug.duplicateName"),
						desc:   a.tf("security.sug.duplicateDesc", map[string]any{"n": stats["duplicate"]}),
						pct:    100,
						status: a.tf("security.sug.duplicateStatus", map[string]any{"n": stats["duplicate"]}),
						action: a.t("security.sug.duplicateAction"), disabled: true},
					{icon: "globe", tone: "danger", name: a.t("security.sug.breachedName"),
						desc:   a.tf("security.sug.breachedDesc", map[string]any{"n": stats["breached"]}),
						pct:    100,
						status: a.tf("security.sug.breachedStatus", map[string]any{"n": stats["breached"]}),
						action: a.t("security.sug.viewBreached"), disabled: stats["breached"] == 0,
						act: func() {
							a.securityFilter = "breached"
							a.view = "all"
							a.resetSelection()
							a.setPage("vault")
						}},
					{icon: "lock", tone: "success", name: a.t("security.sug.autoLockName"),
						desc: a.t("security.sug.autoLockDesc"), pct: 100,
						status: a.autoLockText(), action: a.t("security.sug.done"), done: true},
					{icon: "smartphone", tone: "primary", name: a.t("security.sug.twofaName"),
						desc: a.t("security.sug.twofaDesc"), pct: 35,
						status: a.t("security.sug.twofaStatus"), action: a.t("security.sug.learnMore"),
						act: func() {
							a.settingsTab = 1
							a.setPage("settings")
						}},
					{icon: "hard-drive-upload", tone: "primary", name: a.t("security.sug.backupName"),
						desc: a.t("security.sug.backupDesc"), pct: 80,
						status: a.t("security.sug.backupStatus"), action: a.t("security.sug.backupNow"),
						act: func() { a.setPage("backup") }},
				}
				for _, s := range rows {
					s := s
					item := ui.Column(c).Padding(14).Gap(10).Radius(10)
					if item.Hovered() {
						item.Background(t.Background)
					}
					item.Children(func() {
						ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
							iconBox := ui.Box(c).Size(36, 36).Radius(10).Center()
							iconBox.Background(suggestionBg(t, s.tone))
							iconBox.Children(func() {
								ui.Icon(c, icon(s.icon)).FontSize(18).TextColor(suggestionColor(t, s.tone))
							})
							ui.Column(c).Grow(1).Gap(2).Children(func() {
								ui.Text(c, s.name).FontSize(t.FontSize * 0.9643).FontWeight(600)
								ui.Text(c, s.desc).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
							})
						})
						bar := ui.Box(c).FillWidth().Height(5).Radius(999).Background(t.SurfaceHover)
						bar.Children(func() {
							ui.Box(c).Absolute().Top(0).Left(0).Bottom(0).WidthPercent(s.pct).Radius(999).
								Background(suggestionColor(t, s.tone))
						})
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							ui.Text(c, s.status).FontSize(t.FontSize * 0.8214).TextColor(t.TextMuted).FontWeight(500)
							ui.Spacer(c)
							btn := ui.Button(c, s.action).Padding(5, 12).Radius(6).FontSize(t.FontSize * 0.8214)
							if s.done {
								btn.Background(ui.RGBA(52, 211, 153, 0.10)).Border(1, ui.RGBA(52, 211, 153, 0.30)).TextColor(t.Success)
							}
							if btn.Clicked() {
								if s.done {
									a.toast(a.t("security.sug.done"))
								} else if s.disabled {
									a.toast(a.t("security.sug.noIssues"))
								} else if s.act != nil {
									s.act()
								} else {
									a.toast(a.tf("security.sug.actionToast", map[string]any{"name": s.name, "action": s.action}))
								}
							}
						})
					})
				}
			})
		})
	})
}

func (a *App) issueTag(c *ui.Context, kind, risk string) {
	t := c.Theme()
	bg := ui.RGBA(251, 191, 36, 0.12)
	col := t.Warning
	switch {
	case kind == "breached" || risk == "high":
		bg = ui.RGBA(248, 113, 113, 0.12)
		col = t.Danger
	case risk == "low":
		bg = ui.RGBA(250, 204, 21, 0.12)
		col = ui.Hex("#eab308")
	}
	ui.Box(c).Padding(3, 10).Radius(999).Background(bg).Shrink(0).Children(func() {
		ui.Text(c, a.issueTypeLabel(kind)).FontSize(t.FontSize * 0.8214).FontWeight(500).TextColor(col)
	})
}

func suggestionBg(t *ui.Theme, tone string) ui.Color {
	switch tone {
	case "danger":
		return ui.RGBA(248, 113, 113, 0.12)
	case "warning":
		return ui.RGBA(251, 191, 36, 0.12)
	case "success":
		return ui.RGBA(52, 211, 153, 0.12)
	default:
		return ui.RGBA(129, 140, 248, 0.12)
	}
}

func suggestionColor(t *ui.Theme, tone string) ui.Color {
	switch tone {
	case "danger":
		return t.Danger
	case "warning":
		return t.Warning
	case "success":
		return t.Success
	case "info":
		return infoColor(t)
	default:
		return t.Accent
	}
}
