package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewVaultPage(c *ui.Context) {
	t := c.Theme()
	list := a.filtered()
	if a.selectedIndex >= len(list) {
		a.selectedIndex = -1
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.vaultCategoryPanel(c)
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			a.vaultListToolbar(c, list)
			if a.listView == "grid" {
				a.vaultGrid(c, list)
			} else {
				a.vaultList(c, list)
			}
		})
		detail := ui.Column(c)
		detail.Width(380)
		detail.FillHeight()
		detail.Background(t.Surface)
		detail.BorderWidth(0, 0, 0, 1)
		detail.BorderColor(t.Border)
		detail.Children(func() {
			ui.Scroll(c).Grow(1).Children(func() {
				a.vaultDetail(c)
			})
		})
	})
}

// vaultCategoryPanel is the design's second column: quick access, the
// local encryption card, security filters, folders and tags.
func (a *App) vaultCategoryPanel(c *ui.Context) {
	t := c.Theme()
	cnt := a.computeCounts()
	col := ui.Column(c)
	col.Width(240)
	col.FillHeight()
	col.Background(t.Background)
	col.BorderWidth(0, 1, 0, 0)
	col.BorderColor(t.Border)
	col.Children(func() {
		ui.Scroll(c).Grow(1).Padding(20, 16).Children(func() {
			ui.Column(c).FillWidth().Gap(24).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					a.categoryTitle(c, a.t("vault.quickAccess"))
					allActive := a.view == "all" && a.folder == "all" && a.securityFilter == "all" && a.tag == ""
					a.categoryItem(c, "lock", a.t("vault.allPasswords"), cnt.All, allActive, t.Accent, func() {
						a.view = "all"
						a.folder = "all"
						a.securityFilter = "all"
						a.tag = ""
						a.resetSelection()
					})
					a.categoryItem(c, "star", a.t("vault.favorites"), cnt.Favorites, a.view == "favorites", t.Warning, func() {
						a.view = "favorites"
						a.folder = "all"
						a.securityFilter = "all"
						a.tag = ""
						a.resetSelection()
					})
					a.categoryItem(c, "clock", a.t("vault.recent"), cnt.Recent, a.view == "recent", infoColor(t), func() {
						a.view = "recent"
						a.folder = "all"
						a.securityFilter = "all"
						a.tag = ""
						a.resetSelection()
					})
				})

				a.localEncryptionCard(c)

				ui.Column(c).Gap(2).Children(func() {
					a.categoryTitle(c, a.t("vault.securityStatus"))
					a.categoryItem(c, "shield-check", a.t("vault.securityStrong"), cnt.Strong, a.securityFilter == "strong", t.Success, func() {
						a.securityFilter = "strong"
						a.view = "all"
						a.resetSelection()
					})
					a.categoryItem(c, "shield-alert", a.t("vault.securityMedium"), cnt.Medium, a.securityFilter == "medium", t.Warning, func() {
						a.securityFilter = "medium"
						a.view = "all"
						a.resetSelection()
					})
					a.categoryItem(c, "shield-x", a.t("vault.securityWeak"), cnt.Weak, a.securityFilter == "weak", t.Danger, func() {
						a.securityFilter = "weak"
						a.view = "all"
						a.resetSelection()
					})
					a.categoryItem(c, "alert-triangle", a.t("issue.breached"), cnt.Breached, a.securityFilter == "breached", t.Danger, func() {
						a.securityFilter = "breached"
						a.view = "all"
						a.resetSelection()
					})
				})

				ui.Column(c).Gap(2).Children(func() {
					a.categoryTitle(c, a.t("vault.folders"))
					for _, f := range folders {
						folder := f
						a.categoryItem(c, a.folderIcon(folder), a.folderLabel(folder), cnt.Folders[folder], a.folder == folder, folderColor(folder), func() {
							a.folder = folder
							a.view = "all"
							a.securityFilter = "all"
							a.tag = ""
							a.resetSelection()
						})
					}
				})

				ui.Column(c).Gap(8).Children(func() {
					a.categoryTitle(c, a.t("vault.tags"))
					ui.Row(c).Wrap().Gap(8).Padding(0, 8).Children(func() {
						for _, tag := range tags {
							tag := tag
							bg, fg, dot := a.tagColors(tag)
							active := a.tag == tag
							pill := ui.Row(c).Padding(5, 12).Gap(6).Radius(999).Background(bg).Cursor(ui.CursorPointer)
							if active {
								pill.Border(1, t.Accent)
							}
							pill.Label(tag)
							pill.OnClick(func() {
								if a.tag == tag {
									a.tag = ""
								} else {
									a.tag = tag
								}
								a.resetSelection()
							})
							pill.Children(func() {
								ui.Box(c).Size(8, 8).Radius(4).Background(dot)
								ui.Text(c, fmt.Sprintf("%s (%d)", tag, cnt.Tags[tag])).FontSize(12).FontWeight(500).TextColor(fg)
							})
						}
					})
				})
			})
		})
	})
}

func (a *App) tagColors(tag string) (bg, fg, dot ui.Color) {
	switch tag {
	case "重要":
		return ui.RGBA(248, 113, 113, 0.12), ui.Hex("#ef4444"), ui.Hex("#f87171")
	case "紧急":
		return ui.RGBA(251, 191, 36, 0.12), ui.Hex("#d97706"), ui.Hex("#fbbf24")
	default:
		return ui.RGBA(96, 165, 250, 0.12), ui.Hex("#3b82f6"), ui.Hex("#60a5fa")
	}
}

func (a *App) categoryTitle(c *ui.Context, title string) {
	ui.Text(c, strings.ToUpper(title)).
		FontSize(11).
		FontWeight(600).
		LetterSpacing(0.5).
		TextColor(c.Theme().TextMuted).
		Padding(0, 8).
		Margin(0, 0, 10, 0)
}

func (a *App) categoryItem(c *ui.Context, iconName, label string, count int, active bool, iconColor ui.Color, onClick func()) {
	t := c.Theme()
	row := ui.Row(c).Padding(8, 10).Gap(10).Radius(10).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Label(label)
	switch {
	case active:
		row.Background(a.sidebarActive())
	case row.Hovered():
		row.Background(t.SurfaceHover)
	}
	row.OnClick(onClick)
	row.Children(func() {
		labelColor := t.Text
		if active {
			iconColor = t.Accent
			labelColor = t.Accent
		}
		ui.Icon(c, icon(iconName)).FontSize(16).TextColor(iconColor)
		ui.Text(c, label).FontSize(t.FontSize * 0.9643).FontWeight(500).Grow(1).SingleLine().
			TextColor(labelColor)
		bg := a.surface3()
		col := t.TextMuted
		if active {
			bg = ui.RGBA(129, 140, 248, 0.15)
			col = t.Accent
		}
		ui.Box(c).Padding(2, 8).Radius(999).Background(bg).Children(func() {
			ui.Text(c, fmt.Sprint(count)).FontSize(12).FontWeight(500).TextColor(col)
		})
	})
}

func (a *App) localEncryptionCard(c *ui.Context) {
	t := c.Theme()
	card := ui.Column(c).Padding(14).Gap(10).Radius(10).
		Gradient(ui.RGBA(52, 211, 153, 0.08), ui.RGBA(129, 140, 248, 0.08), 135).
		Border(1, ui.RGBA(52, 211, 153, 0.2)).
		Margin(0, 0, 16, 0)
	card.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			iconWrap := ui.Box(c).Size(36, 36).Radius(6).Background(ui.RGBA(52, 211, 153, 0.15)).Center()
			iconWrap.Children(func() {
				ui.Icon(c, icon("hard-drive")).FontSize(18).TextColor(t.Success)
				overlay := ui.Box(c).Size(14, 14).Radius(3).Background(t.Surface).Center().Absolute().Bottom(-2).Right(-2)
				overlay.Children(func() {
					ui.Icon(c, icon("lock")).FontSize(9).TextColor(t.Accent)
				})
			})
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, a.t("vault.localEncrypted")).FontSize(13).FontWeight(600)
				ui.Text(c, a.t("vault.encryptionDesc")).FontSize(11.5).TextColor(t.TextMuted)
			})
		})
		ui.Text(c, a.t("vault.zeroKnowledge")).FontSize(11.5).TextColor(t.TextMuted)
		ui.Row(c).Justify(ui.Start).Children(func() {
			badge := ui.Row(c).Padding(3, 8).Gap(4).Radius(999).Background(ui.RGBA(52, 211, 153, 0.15))
			badge.Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Success)
				ui.Text(c, a.t("vault.runningSecurely")).FontSize(11).FontWeight(600).TextColor(t.Success)
			})
		})
	})
}

func (a *App) vaultListToolbar(c *ui.Context, list []Entry) {
	t := c.Theme()
	count := a.tf("vault.totalCount", map[string]any{"n": len(a.entries)})
	if len(list) != len(a.entries) {
		count = a.tf("vault.showingRange", map[string]any{"n": len(list), "total": len(a.entries)})
	}
	ui.Row(c).Padding(20, 20, 10).AlignItems(ui.Center).Gap(12).Children(func() {
		sort := a.toolbarButton(c, "arrow-up-down", a.sortLabel(a.sort), true)
		if sort.Clicked() {
			order := []string{"recent", "name", "strength"}
			next := 0
			for i, v := range order {
				if v == a.sort {
					next = (i + 1) % len(order)
				}
			}
			a.sort = order[next]
		}
		filter := a.toolbarButton(c, "funnel", a.t("vault.filter"), false)
		if filter.Clicked() {
			a.toast(a.t("vault.filtersComingSoon"))
		}
		ui.Spacer(c)
		ui.Text(c, count).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted)
	})
}

func (a *App) toolbarButton(c *ui.Context, iconName, label string, chevron bool) ui.Element {
	t := c.Theme()
	b := ui.Row(c).Padding(7, 12).Gap(6).Radius(10).Background(t.Surface).Border(1, t.Border).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	b.Label(label)
	if b.Hovered() {
		b.Border(1, t.Accent)
	}
	b.Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(14).TextColor(t.TextMuted)
		ui.Text(c, label).FontSize(t.FontSize * 0.9286).TextColor(t.Text).SingleLine()
		if chevron {
			ui.Icon(c, icon("chevron-down")).FontSize(13).TextColor(t.TextMuted)
		}
	})
	return b
}

func (a *App) vaultList(c *ui.Context, list []Entry) {
	t := c.Theme()
	a.listState.Key = func(i int) any { return list[i].ID }
	a.listState.Label = func(i int) string { return list[i].Site }
	a.listState.Selected = &a.selectedIndex

	// The list paints the theme's selection color behind the row. The cards
	// carry the selection themselves, so a neutral fill here keeps the accent
	// to the card's edge and stops it peeking around the rounded corners.
	listTheme := *t
	listTheme.Selection = a.surface2()
	listTheme.AccentText = t.Text
	listTheme.Background = ui.Transparent
	ui.Themed(c, &listTheme).Grow(1).Children(func() {
		el := ui.List(c, &a.listState, len(list)).Grow(1).Padding(20).Gap(10).FocusRing(false)
		el.Rows(func(row ui.ListRow) {
			i := row.Index
			if i < 0 || i >= len(list) {
				return
			}
			e := list[i]
			rc := row.Context
			card := ui.Row(rc).Padding(16).Gap(14).Radius(14).AlignItems(ui.Center).
				Background(t.Surface).Border(1, t.Border).Cursor(ui.CursorPointer)
			switch {
			case i == a.selectedIndex:
				// A neutral fill, with an accent edge: a filled indigo row is
				// too loud for a selection that the detail panel already shows.
				card.Border(1, t.Accent)
				card.Background(a.surface2())
				card.Shadow(0, 1, 3, 0, a.cardShadow())
			case card.Hovered():
				card.Border(1, t.Accent)
				card.Shadow(0, 4, 12, 0, a.cardShadow())
			}
			card.TextColor(t.Text)
			card.FocusRing(false)
			card.ContextMenu(func(m *ui.Menu) { a.entryMenu(m, e) })
			card.Children(func() {
				a.avatar(rc, e.Site, 44)
				ui.Column(rc).Grow(1).Gap(4).Children(func() {
					ui.Text(rc, e.Site).FontSize(t.FontSize).FontWeight(600).TextColor(t.Text).SingleLine()
					sub := e.Username
					if sub == "" {
						sub = e.URL
					}
					if sub == "" {
						sub = a.t("vault.noUsername")
					}
					ui.Text(rc, sub).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted).SingleLine()
				})
				ui.Row(rc).Gap(10).AlignItems(ui.Center).Shrink(0).Children(func() {
					a.folderBadge(rc, e.Folder)
					a.strengthBarsVertical(rc, strengthFor(e.Password, e.Breached))
					a.favoriteButton(rc, e)
					a.moreButton(rc, e)
				})
			})
		})
		if el.Changed() {
			a.syncSelection()
		}
		if el.Submitted() {
			if e := a.selectedEntry(); e != nil {
				a.openEditEntry(e)
			}
		}
		el.Children(func() {
			if len(list) == 0 {
				if len(a.entries) == 0 {
					a.emptyState(c, "shield-plus", a.t("vault.emptyTitle"), a.t("vault.emptyDesc"))
				} else {
					a.emptyState(c, "search", a.t("vault.noFilterResults"), "")
				}
			}
		})
	})
}

func (a *App) entryMenu(m *ui.Menu, e Entry) {
	if m.Item(a.t("vault.edit")).Chosen() {
		a.openEditEntry(&e)
	}
	if m.Item(a.t("vault.copyPassword")).Chosen() {
		a.copyText(e.Password, a.t("common.field.password"))
	}
	fav := a.t("vault.favorite")
	if e.Favorite {
		fav = a.t("vault.unfavorite")
	}
	if m.Item(fav).Chosen() {
		a.toggleFavorite(e.ID)
	}
	m.Separator()
	if m.Item(a.t("vault.delete")).Danger().Chosen() {
		a.deleteID = e.ID
		a.deleteOpen = true
	}
}

func (a *App) moreButton(c *ui.Context, e Entry) {
	t := c.Theme()
	b := ui.Box(c).Size(28, 26).Radius(7).Center().Cursor(ui.CursorPointer)
	b.Label(a.t("vault.moreActions"))
	b.Tooltip(a.t("vault.moreActions"))
	if b.Hovered() {
		b.Background(a.surface3())
	}
	b.PopoverMenuButton(func(m *ui.Menu) { a.entryMenu(m, e) })
	b.Children(func() {
		col := t.TextMuted
		if b.Hovered() {
			col = t.Text
		}
		ui.Icon(c, icon("ellipsis-vertical")).FontSize(16).TextColor(col)
	})
}

func (a *App) folderBadge(c *ui.Context, folder string) {
	t := c.Theme()
	ui.Box(c).Padding(3, 8).Radius(6).Background(a.surface3()).Shrink(0).Children(func() {
		ui.Text(c, a.folderLabel(folder)).FontSize(t.FontSize * 0.7857).FontWeight(500).TextColor(t.TextMuted).SingleLine()
	})
}

func (a *App) favoriteButton(c *ui.Context, e Entry) {
	t := c.Theme()
	star := ui.Box(c).Size(28, 26).Radius(7).Center().Cursor(ui.CursorPointer)
	star.Label(a.t("vault.favorite"))
	if star.Hovered() {
		star.Background(a.surface3())
	}
	star.OnClick(func() { a.toggleFavorite(e.ID) })
	star.Children(func() {
		col := t.TextMuted
		if e.Favorite {
			col = t.Warning
		}
		ui.Icon(c, icon("star")).FontSize(16).TextColor(col)
	})
}

func (a *App) vaultGrid(c *ui.Context, list []Entry) {
	t := c.Theme()
	if len(list) == 0 {
		if len(a.entries) == 0 {
			a.emptyState(c, "shield-plus", a.t("vault.emptyTitle"), a.t("vault.emptyDesc"))
		} else {
			a.emptyState(c, "search", a.t("vault.noFilterResults"), "")
		}
		return
	}
	ui.Scroll(c).Grow(1).Padding(20).Children(func() {
		ui.Grid(c).Columns(3).Gap(12).Children(func() {
			for i, entry := range list {
				i, e := i, entry
				selected := i == a.selectedIndex
				card := ui.Column(c).Padding(14).Gap(10).Radius(14).Background(t.Surface).Border(1, t.Border).Cursor(ui.CursorPointer)
				switch {
				case selected:
					card.Background(a.surface2())
					card.Border(1, t.Accent)
					card.Shadow(0, 1, 3, 0, a.cardShadow())
				case card.Hovered():
					card.Border(1, t.Accent)
					card.Shadow(0, 4, 12, 0, a.cardShadow())
				}
				card.ContextMenu(func(m *ui.Menu) { a.entryMenu(m, e) })
				card.OnClick(func() {
					a.selectedIndex = i
					a.syncSelection()
				})
				card.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						a.avatar(c, e.Site, 40)
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, e.Site).FontSize(t.FontSize * 0.9643).FontWeight(600).SingleLine()
							sub := e.Username
							if sub == "" {
								sub = e.URL
							}
							if sub == "" {
								sub = a.t("vault.noUsername")
							}
							ui.Text(c, sub).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted).SingleLine()
						})
						a.favoriteButton(c, e)
					})
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						a.folderBadge(c, e.Folder)
						ui.Spacer(c)
						a.strengthBarsVertical(c, strengthFor(e.Password, e.Breached))
					})
				})
			}
		})
	})
}

func (a *App) strengthBarsVertical(c *ui.Context, s Strength) {
	t := c.Theme()
	col := strengthColor(t, s)
	ui.Row(c).Gap(2).Children(func() {
		for i := 1; i <= 5; i++ {
			bar := ui.Box(c).Width(4).Height(20).Radius(2)
			if i <= s.Bars {
				bar.Background(col)
			} else {
				bar.Background(t.Border)
			}
		}
	})
}

func (a *App) vaultDetail(c *ui.Context) {
	t := c.Theme()
	e := a.selectedEntry()
	if e == nil {
		a.emptyState(c, "shield", a.t("vault.selectToView"), "")
		return
	}
	s := strengthFor(e.Password, e.Breached)

	ui.Column(c).FillWidth().Children(func() {
		// Header
		ui.Column(c).Padding(28, 24, 20).Gap(8).Center().BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			a.avatar(c, e.Site, 72)
			ui.Text(c, e.Site).FontSize(t.FontSize * 1.2857).FontWeight(700).TextAlign(ui.Center)
			if e.URL != "" {
				u := e.URL
				if !strings.Contains(u, "://") {
					u = "https://" + u
				}
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Link(c, e.URL, u).FontSize(t.FontSize * 0.9286)
					if a.iconButton(c, "copy", a.t("vault.copyUrl")).Clicked() {
						a.copyText(e.URL, a.t("common.field.url"))
					}
				})
			}
		})

		ui.Column(c).Padding(20, 24).Gap(18).Children(func() {
			// Username
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, a.t("vault.usernameEmail")).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
				row := ui.Row(c).Padding(10, 12).Gap(6).Radius(10).Background(a.surface3())
				row.Children(func() {
					value := e.Username
					if value == "" {
						value = "—"
					}
					ui.Text(c, value).FontSize(t.FontSize).Font("monospace").LetterSpacing(0.3).Grow(1).SingleLine().Selectable()
					if e.Username != "" {
						if a.iconButton(c, "copy", a.t("vault.copy")).Clicked() {
							a.copyText(e.Username, a.t("common.field.username"))
						}
					}
				})
			})

			// Password
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, a.t("common.field.password")).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
				row := ui.Row(c).Padding(10, 12).Gap(6).Radius(10).Background(a.surface3())
				row.Children(func() {
					shown := "••••••••••••"
					if a.revealPassword {
						shown = e.Password
					}
					txt := ui.Text(c, shown).FontSize(t.FontSize).Font("monospace").LetterSpacing(0.3).
						Grow(1).SingleLine().Selectable()
					if !a.revealPassword {
						txt.LetterSpacing(4)
					}
					if a.iconButton(c, "eye", a.t("vault.show")).Clicked() {
						a.revealPassword = !a.revealPassword
					}
					if a.iconButton(c, "copy", a.t("vault.copy")).Clicked() {
						a.copyText(e.Password, a.t("common.field.password"))
					}
				})
				a.strengthRow(c, s)
			})

			// Info
			info := [][2]string{
				{a.t("vault.createdAt"), e.CreatedAt},
				{a.t("vault.updatedAt"), e.UpdatedAt},
				{a.t("vault.folder"), a.folderLabel(e.Folder)},
				{a.t("vault.usageCount"), a.tf("vault.timesUsed", map[string]any{"n": e.UsageCount})},
			}
			ui.Grid(c).Columns(2).Gap(12).Children(func() {
				for _, item := range info {
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, item[0]).FontSize(t.FontSize * 0.7857).TextColor(t.TextMuted)
						ui.Text(c, item[1]).FontSize(t.FontSize * 0.9286)
					})
				}
			})

			// Storage
			card := ui.Column(c).Padding(12).Gap(4).Radius(10).Background(a.surface3())
			card.Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("hard-drive")).FontSize(15).TextColor(t.Success)
					ui.Text(c, a.t("vault.localStorage")).FontSize(t.FontSize * 0.9286).FontWeight(600)
				})
				ui.Text(c, a.t("vault.encryptedDatabase")+" · AES-256-GCM").FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Success)
					ui.Text(c, a.t("vault.secure")).FontSize(t.FontSize * 0.8571).TextColor(t.Success)
				})
			})

			// Strength
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, a.t("vault.strengthDetail")).FontSize(t.FontSize * 0.9643).FontWeight(600)
				card := ui.Column(c).Padding(12).Gap(8).Radius(10).Background(a.surface3())
				card.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.End).Children(func() {
						ui.Text(c, fmt.Sprint(s.Score)).FontSize(t.FontSize * 1.714).FontWeight(700).TextColor(strengthColor(t, s))
						ui.Text(c, a.strengthLabel(s)).FontSize(t.FontSize * 0.9286).TextColor(strengthColor(t, s))
					})
					a.strengthBarsWide(c, s)
					ui.Text(c, a.t("vault.safetyTips")).FontSize(t.FontSize * 0.8571).FontWeight(600)
					for _, tip := range []struct {
						ok   bool
						text string
					}{
						{len([]rune(e.Password)) >= 16, a.t("vault.tipLength")},
						{hasLower(e.Password) && hasUpper(e.Password), a.t("vault.tipMixedCase")},
						{hasDigit(e.Password) && hasSymbol(e.Password), a.t("vault.tipDigitsSymbols")},
						{!e.Breached, a.t("vault.tipNotBreached")},
					} {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							name := "circle-x"
							col := t.Danger
							if tip.ok {
								name = "circle-check"
								col = t.Success
							}
							ui.Icon(c, icon(name)).FontSize(13).TextColor(col)
							ui.Text(c, tip.text).FontSize(t.FontSize * 0.8571).TextColor(t.TextMuted)
						})
					}
				})
			})

			if e.Notes != "" {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, a.t("vault.notes")).FontSize(t.FontSize * 0.9643).FontWeight(600)
					ui.Text(c, e.Notes).FontSize(t.FontSize * 0.9286).TextColor(t.TextMuted).Selectable()
				})
			}

			if len(e.Tags) > 0 {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, a.t("vault.tags")).FontSize(t.FontSize * 0.9643).FontWeight(600)
					ui.Row(c).Wrap().Gap(6).Children(func() {
						for _, tag := range e.Tags {
							tag := tag
							bg, fg, dot := a.tagColors(tag)
							pill := ui.Row(c).Padding(4, 10).Gap(5).Radius(999).Background(bg)
							pill.Children(func() {
								ui.Box(c).Size(6, 6).Radius(3).Background(dot)
								ui.Text(c, tag).FontSize(11.5).FontWeight(500).TextColor(fg)
							})
						}
					})
				})
			}

			ui.Row(c).Gap(8).Children(func() {
				if a.iconTextButton(c, "pencil", a.t("vault.edit"), false).Clicked() {
					a.openEditEntry(e)
				}
				if a.iconTextButton(c, "copy", a.t("vault.copyPassword"), false).Clicked() {
					a.copyText(e.Password, a.t("common.field.password"))
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				if a.iconTextButton(c, "trash-2", a.t("vault.delete"), false).Clicked() {
					a.deleteID = e.ID
					a.deleteOpen = true
				}
				if a.iconTextButton(c, "link", a.t("vault.share"), false).Clicked() {
					a.toast(a.t("vault.shareComingSoon"))
				}
			})
		})
	})
}

func (e *Entry) UsernameOrDash() string {
	if e.Username == "" {
		return "—"
	}
	return e.Username
}

func (a *App) strengthBarsWide(c *ui.Context, s Strength) {
	t := c.Theme()
	col := strengthColor(t, s)
	ui.Row(c).Gap(3).Children(func() {
		for i := 1; i <= 5; i++ {
			bar := ui.Box(c).Height(5).Radius(3).Grow(1)
			if i <= s.Bars {
				bar.Background(col)
			} else {
				bar.Background(t.SurfaceHover)
			}
		}
	})
}

func hasLower(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func hasUpper(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool { return strings.ContainsAny(s, "0123456789") }

func hasSymbol(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}
