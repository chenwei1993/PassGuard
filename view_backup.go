package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *App) viewBackupPage(c *ui.Context) {
	t := c.Theme()
	if !a.backupsLoaded {
		a.loadBackupsAsync()
	}
	interval := a.backupSetting.IntervalHours
	if interval <= 0 {
		interval = 24
	}
	maxKeep := a.backupSetting.MaxKeep
	if maxKeep <= 0 {
		maxKeep = 10
	}
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(1080).Padding(22).Gap(16).Children(func() {
			ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
				// Backup now
				card := a.card(c).Grow(1)
				card.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("hard-drive-upload")).FontSize(20).TextColor(t.Accent)
						ui.Column(c).Gap(1).Children(func() {
							ui.Text(c, a.t("backup.backupNow")).FontSize(t.FontSize * 1.05).Bold()
							ui.Text(c, a.t("backup.backupNowDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
						})
					})
					ui.Divider(c).Background(t.Border)
					if len(a.backups) > 0 {
						last := a.backups[0]
						ui.Text(c, a.tf("backup.lastBackup", map[string]any{
							"time":     fmtClock(last.MTime),
							"duration": a.fmtDuration(last.MTime),
						})).FontSize(t.FontSize * 0.9)
					} else {
						ui.Text(c, a.t("backup.noBackupYet")).FontSize(t.FontSize * 0.9)
					}
					ui.Text(c, a.t("backup.backupNowHint")).FontSize(t.FontSize * 0.82).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Children(func() {
						label := a.t("backup.backupNow")
						if a.busy {
							label = a.t("backup.backingUp")
						}
						if a.iconTextButton(c, "database-backup", label, true).Disabled(a.busy).Clicked() {
							a.backupNowAsync()
						}
						if a.iconTextButton(c, "folder-open", a.t("backup.openFolder"), false).Clicked() {
							a.openBackupFolder()
						}
					})
				})

				// Auto settings
				card2 := a.card(c).Grow(1)
				card2.Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("settings")).FontSize(20).TextColor(t.Accent)
						ui.Column(c).Gap(1).Children(func() {
							ui.Text(c, a.t("backup.autoSettings")).FontSize(t.FontSize * 1.05).Bold()
							ui.Text(c, a.t("backup.autoSettingsDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
						})
					})
					ui.Divider(c).Background(t.Border)
					enabled := a.backupSetting.Enabled
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, a.t("backup.enableAuto")).FontSize(t.FontSize * 0.9)
							ui.Text(c, a.t("backup.enableAutoDesc")).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
						})
						ui.Switch(c, &enabled)
					})
					if enabled != a.backupSetting.Enabled {
						a.backupSetting.Enabled = enabled
						a.saveBackupSettings()
					}
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, a.t("backup.interval")).FontSize(t.FontSize * 0.88)
							ui.Text(c, a.t("backup.intervalDesc")).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
						})
						opts := []struct {
							hours float64
							label string
						}{
							{24, a.t("backup.intervalDaily")},
							{168, a.t("backup.intervalWeekly")},
							{720, a.t("backup.intervalMonthly")},
						}
						labels := make([]string, len(opts))
						choice := 0
						for i, o := range opts {
							labels[i] = o.label
							if o.hours == interval {
								choice = i
							}
						}
						chooseOption(c, labels, choice, 150, func(i int) {
							a.backupSetting.IntervalHours = opts[i].hours
							a.saveBackupSettings()
						})
					})
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, a.t("backup.maxKeep")).FontSize(t.FontSize * 0.88)
							ui.Text(c, a.t("backup.maxKeepDesc")).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
						})
						keeps := []int{5, 10, 20, 50}
						labels := make([]string, len(keeps))
						choice := 0
						for i, n := range keeps {
							labels[i] = a.tf("backup.keepN", map[string]any{"n": n})
							if n == maxKeep {
								choice = i
							}
						}
						chooseOption(c, labels, choice, 150, func(i int) {
							a.backupSetting.MaxKeep = keeps[i]
							a.saveBackupSettings()
						})
					})
					ui.Text(c, a.t("backup.autoTip")).FontSize(t.FontSize * 0.8).TextColor(t.TextMuted)
				})
			})

			// History
			card := a.card(c)
			card.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("history")).FontSize(20).TextColor(t.Accent)
					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, a.t("backup.historyTitle")).FontSize(t.FontSize * 1.05).Bold()
						ui.Text(c, a.t("backup.historyDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
					})
					ui.Spacer(c)
					if len(a.backups) > 0 {
						total := int64(0)
						for _, b := range a.backups {
							total += b.Size
						}
						ui.Text(c, a.tf("backup.countSummary", map[string]any{"n": len(a.backups), "size": a.fmtSize(total)})).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
					}
				})
				ui.Divider(c).Background(t.Border)
				if len(a.backups) == 0 {
					ui.Column(c).Center().Padding(30).Gap(6).Children(func() {
						ui.Icon(c, icon("hard-drive")).FontSize(30).TextColor(t.TextMuted)
						ui.Text(c, a.t("backup.emptyTitle")).Bold()
						ui.Text(c, a.t("backup.emptyDesc")).FontSize(t.FontSize * 0.84).TextColor(t.TextMuted)
					})
				}
				for _, b := range a.backups {
					b := b
					ui.Row(c).Padding(8, 0).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
						ui.Icon(c, icon("file-lock-2")).FontSize(16).TextColor(t.Accent)
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							ui.Text(c, b.Name).FontSize(t.FontSize * 0.86).Bold().SingleLine()
							ui.Text(c, fmtClock(b.MTime)+" · "+a.fmtSize(b.Size)).FontSize(t.FontSize * 0.78).TextColor(t.TextMuted)
						})
						if ui.Button(c, a.t("backup.restore")).Padding(3, 10).Radius(6).FontSize(t.FontSize * 0.8).Clicked() {
							a.restoreName = b.Name
							a.restorePw = ""
							a.restoreOpen = true
						}
						if a.iconButton(c, "folder-open", a.t("backup.openContainingFolder")).Clicked() {
							a.openBackupFolder()
						}
						if a.iconButton(c, "trash-2", a.t("backup.deleteBackup")).Clicked() {
							a.backupDeleteName = b.Name
							a.backupDeleteOpen = true
						}
					})
				}
			})
		})
	})
}

func (a *App) saveBackupSettings() {
	s := a.backupSetting
	a.vault.BackupSaveSettings(s.Enabled, s.IntervalHours, s.MaxKeep)
	a.loadBackupsAsync()
	if s.Enabled && len(a.backups) == 0 && !a.busy {
		a.backupNowAsync()
	}
}
