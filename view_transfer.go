package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewTransferPage(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(1080).Padding(22).Gap(16).Children(func() {
			ui.Row(c).Gap(16).AlignItems(ui.Stretch).Children(func() {
				// Export
				card := a.card(c).Grow(1)
				card.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("download")).FontSize(20).TextColor(t.Accent)
						ui.Column(c).Gap(1).Children(func() {
							ui.Text(c, a.t("transfer.exportData")).FontSize(t.FontSize * 1.05).Bold()
							ui.Text(c, a.t("transfer.exportDataDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
						})
					})
					ui.Divider(c).Background(t.Border)
					a.transferOption(c, "file-down", a.t("transfer.encryptedBackup"), a.t("transfer.encryptedBackupDesc"), a.t("transfer.export"), true, func() {
						a.exportFileAsync()
					})
					a.transferOption(c, "file-text", a.t("transfer.csvFile"), a.t("transfer.csvDesc"), a.t("transfer.export"), true, func() {
						a.exportCSVAsync()
					})
				})

				// Import
				card2 := a.card(c).Grow(1)
				card2.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("upload")).FontSize(20).TextColor(t.Success)
						ui.Column(c).Gap(1).Children(func() {
							ui.Text(c, a.t("transfer.importData")).FontSize(t.FontSize * 1.05).Bold()
							ui.Text(c, a.t("transfer.importDataDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
						})
					})
					ui.Divider(c).Background(t.Border)
					a.transferOption(c, "file-up", a.t("transfer.encryptedBackup"), a.t("transfer.importBinDesc"), a.t("transfer.import"), false, func() {
						a.pickBackupAsync()
					})
					a.transferOption(c, "file-text", a.t("transfer.csvFile"), a.t("transfer.importCsvDesc"), a.t("transfer.import"), false, func() {
						a.pickCSVAsync()
					})
				})
			})

			// Recent activity
			card := a.card(c)
			card.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("history")).FontSize(20).TextColor(t.Accent)
					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, a.t("transfer.recentActivity")).FontSize(t.FontSize * 1.05).Bold()
						ui.Text(c, a.t("transfer.recentActivityDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
					})
				})
				if len(a.activity) == 0 {
					ui.Text(c, a.t("transfer.noActivity")).FontSize(t.FontSize * 0.9).TextColor(t.TextMuted).PaddingY(20).TextAlign(ui.Center)
				}
				for _, h := range a.activity {
					ui.Row(c).Padding(7, 0).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
						col := t.Success
						name := "upload"
						if h.Kind == "export" {
							col = t.Accent
							name = "download"
						}
						ui.Icon(c, icon(name)).FontSize(14).TextColor(col)
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, h.Name).FontSize(t.FontSize * 0.86).Bold()
							ui.Text(c, h.Detail).FontSize(t.FontSize * 0.78).TextColor(t.TextMuted).SingleLine()
						})
						ui.Text(c, h.At).FontSize(t.FontSize * 0.76).TextColor(t.TextMuted)
					})
				}
			})
		})
	})
}

func (a *App) transferOption(c *ui.Context, iconName, title, desc, action string, export bool, onClick func()) {
	t := c.Theme()
	row := ui.Row(c).Padding(10).Gap(10).Radius(9).Border(1, t.Border).AlignItems(ui.Center)
	if row.Hovered() {
		row.Border(1, t.Accent)
		row.Background(t.Background)
	}
	row.Children(func() {
		col := t.Success
		if export {
			col = t.Accent
		}
		ui.Icon(c, icon(iconName)).FontSize(18).TextColor(col)
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(t.FontSize * 0.92).Bold()
			ui.Text(c, desc).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
		})
		if ui.Button(c, action).Padding(4, 12).Radius(7).FontSize(t.FontSize * 0.84).Clicked() {
			onClick()
		}
	})
}

func fmtClock(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04")
}

func (a *App) fmtDuration(ms int64) string {
	if ms <= 0 {
		return a.t("backup.neverBackedUp")
	}
	diff := timeNow().UnixMilli() - ms
	switch {
	case diff < 60_000:
		return a.t("backup.justNow")
	case diff < 3_600_000:
		return a.tf("backup.minutesAgo", map[string]any{"n": diff / 60_000})
	case diff < 86_400_000:
		return a.tf("backup.hoursAgo", map[string]any{"n": diff / 3_600_000})
	default:
		return a.tf("backup.daysAgo", map[string]any{"n": diff / 86_400_000})
	}
}
