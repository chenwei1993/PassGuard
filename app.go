package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

type entryForm struct {
	Site     string
	Username string
	Password string
	URL      string
	Notes    string
	Folder   string
}

type genState struct {
	Type             int // 0 random, 1 memorable, 2 pin
	Length           float64
	Upper            bool
	Lower            bool
	Numbers          bool
	Symbols          bool
	ExcludeSimilar   bool
	ExcludeAmbiguous bool
	ExcludeDuplicate bool
	Password         string
	Signature        string
	History          []string
}

type activity struct {
	Kind   string // export | import
	Name   string
	Detail string
	At     string
	OK     bool
}

type counts struct {
	All       int
	Favorites int
	Recent    int
	Strong    int
	Medium    int
	Weak      int
	Breached  int
	Folders   map[string]int
	Tags      map[string]int
}

// App is the state the native window shows.
type App struct {
	vault    *Vault
	win      *mygo.Window
	settings AppSettings
	locale   string

	phase  string // booting | setup | locked | unlocked
	status OpResult
	title  string

	entries []Entry

	busy      bool
	busyLabel string
	toasts    []string

	// Lock screen
	lockTab       int
	password      string
	confirm       string
	showPassword  bool
	remember      bool
	failAttempts  int
	cooldownUntil time.Time
	lastError     string

	// Navigation
	page      string
	notifOpen bool

	// Vault page
	view           string // all | favorites | recent
	securityFilter string // all | strong | medium | weak | breached
	folder         string // all or a folder
	tag            string
	search         string
	sort           string // recent | name | strength
	selectedIndex  int
	selectedID     string
	revealPassword bool
	focusSearch    bool
	listView       string
	listState      ui.ListState
	searchHandle   ui.Handle

	// Entry modal
	entryOpen        bool
	entryEditingID   string
	entryForm        entryForm
	showFormPassword bool
	formError        string

	deleteOpen bool
	deleteID   string

	// Generator
	gen genState

	// Security page
	secTab        int
	scanningUntil time.Time

	// Transfer page
	activity     []activity
	backupPwOpen bool
	backupFile   *VaultFile
	backupName   string
	backupPw     string
	importOpen   bool
	importSource string
	importList   []Entry
	importMode   int // 0 merge, 1 overwrite

	// Backup page
	restoreOpen      bool
	restoreName      string
	restorePw        string
	backups          []BackupInfo
	backupSetting    BackupSettingsView
	backupsLoaded    bool
	backupsLoading   bool
	backupDeleteOpen bool
	backupDeleteName string

	// Settings page
	settingsTab    int
	curPw          string
	newPw          string
	confirmPw      string
	showCurPw      bool
	showNewPw      bool
	showConfirmPw  bool
	deleteArmedAt  time.Time
	dbInfo         OpResult
	autoLockChoice string
	clipChoice     string

	// Session
	unlockAt     time.Time
	copiedAt     time.Time
	lastActivity time.Time
	idleTimer    *time.Timer
	copySeq      atomic.Int64
}

func NewApp() *App {
	a := &App{
		vault:          NewVault(),
		phase:          "booting",
		page:           "vault",
		view:           "all",
		securityFilter: "all",
		folder:         "all",
		sort:           "recent",
		selectedIndex:  -1,
		settingsTab:    0,
		listView:       "list",
		autoLockChoice: "5 分钟",
		clipChoice:     "30 秒",
	}
	a.loadSettings()
	a.locale = mygo.App.Locale()
	a.gen = genState{
		Type:             0,
		Length:           16,
		Upper:            true,
		Lower:            true,
		Numbers:          true,
		Symbols:          true,
		ExcludeDuplicate: true,
	}
	a.gen.Password = a.generatePassword()
	a.remember = false
	return a
}

func (a *App) generatePassword() string {
	g := &a.gen
	length := int(g.Length)
	var p string
	switch g.Type {
	case 1:
		p = generateMemorable(length, g.Upper, g.Numbers)
	case 2:
		if length < 4 {
			length = 4
		}
		if length > 12 {
			length = 12
		}
		p = generatePin(length)
	default:
		p = generateRandom(length, g.Upper, g.Lower, g.Numbers, g.Symbols, g.ExcludeSimilar, g.ExcludeAmbiguous, g.ExcludeDuplicate)
	}
	return p
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func today() string { return time.Now().Format("2006-01-02") }

func (a *App) normalizeEntry(e *Entry) {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.Site == "" {
		e.Site = a.t("common.unnamed")
	}
	if e.Folder == "" {
		e.Folder = "其他"
	}
	if e.Tags == nil {
		e.Tags = []string{}
	}
	if e.CreatedAt == "" {
		e.CreatedAt = today()
	}
	if e.UpdatedAt == "" {
		e.UpdatedAt = today()
	}
}

func (a *App) entryByID(id string) *Entry {
	for i := range a.entries {
		if a.entries[i].ID == id {
			return &a.entries[i]
		}
	}
	return nil
}

func (a *App) saveEntries() {
	r := a.vault.SaveData(Payload{Entries: a.entries})
	if !r.OK {
		a.toast(r.Message)
		return
	}
	a.refreshInfo()
}

func (a *App) refreshInfo() {
	a.dbInfo = a.vault.GetInfo()
}

// ---------- filters and counts ----------

func (a *App) recentEntries() []Entry {
	cutoff := time.Now().UnixMilli() - 30*dayMs
	var list []Entry
	for _, e := range a.entries {
		if e.LastUsedAt > cutoff {
			list = append(list, e)
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].LastUsedAt > list[j].LastUsedAt })
	return list
}

func (a *App) filtered() []Entry {
	list := make([]Entry, 0, len(a.entries))
	recent := a.recentEntries()
	recentIDs := make(map[string]bool, len(recent))
	for _, e := range recent {
		recentIDs[e.ID] = true
	}
	q := strings.ToLower(strings.TrimSpace(a.search))
	for _, e := range a.entries {
		if a.view == "favorites" && !e.Favorite {
			continue
		}
		if a.view == "recent" && !recentIDs[e.ID] {
			continue
		}
		if a.securityFilter != "all" {
			s := strengthFor(e.Password, e.Breached)
			if s.Level != a.securityFilter && !(a.securityFilter == "weak" && (s.Level == "weak" || s.Level == "breached")) {
				continue
			}
		}
		if a.folder != "all" && e.Folder != a.folder {
			continue
		}
		if a.tag != "" && !contains(e.Tags, a.tag) {
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(e.Site), q) &&
				!strings.Contains(strings.ToLower(e.Username), q) &&
				!strings.Contains(strings.ToLower(e.URL), q) {
				continue
			}
		}
		list = append(list, e)
	}
	switch a.sort {
	case "name":
		sort.SliceStable(list, func(i, j int) bool {
			return strings.ToLower(list[i].Site) < strings.ToLower(list[j].Site)
		})
	case "strength":
		sort.SliceStable(list, func(i, j int) bool {
			return strengthFor(list[i].Password, list[i].Breached).Score >
				strengthFor(list[j].Password, list[j].Breached).Score
		})
	default:
		sort.SliceStable(list, func(i, j int) bool { return list[i].LastUsedAt > list[j].LastUsedAt })
	}
	return list
}

func (a *App) computeCounts() counts {
	c := counts{Folders: map[string]int{}, Tags: map[string]int{"重要": 0, "紧急": 0, "待更新": 0}}
	recent := a.recentEntries()
	c.Recent = len(recent)
	for _, e := range a.entries {
		c.All++
		if e.Favorite {
			c.Favorites++
		}
		c.Folders[e.Folder]++
		switch strengthFor(e.Password, e.Breached).Level {
		case "breached":
			c.Breached++
		case "weak":
			c.Weak++
		case "medium":
			c.Medium++
		case "strong", "very-strong":
			c.Strong++
		}
		for _, t := range e.Tags {
			c.Tags[t]++
		}
	}
	return c
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (a *App) resetSelection() {
	a.selectedIndex = -1
	a.selectedID = ""
	a.revealPassword = false
}

// syncSelection records a new choice: it counts the use and saves, as the
// Electron app did when an entry was selected.
func (a *App) syncSelection() {
	list := a.filtered()
	if a.selectedIndex < 0 || a.selectedIndex >= len(list) {
		a.selectedIndex = -1
		a.selectedID = ""
		return
	}
	id := list[a.selectedIndex].ID
	if a.selectedID == id {
		return
	}
	a.selectedID = id
	if e := a.entryByID(id); e != nil {
		e.UsageCount++
		e.LastUsedAt = time.Now().UnixMilli()
		a.saveEntries()
	}
}

func (a *App) selectedEntry() *Entry {
	list := a.filtered()
	if a.selectedIndex < 0 || a.selectedIndex >= len(list) {
		return nil
	}
	return a.entryByID(list[a.selectedIndex].ID)
}

// ---------- navigation ----------

func (a *App) setPage(p string) {
	a.page = p
	a.notifOpen = false
	switch p {
	case "vault":
		a.resetSelection()
	case "backup":
		if !a.backupsLoaded {
			a.loadBackupsAsync()
		}
	case "settings":
		a.refreshInfo()
	}
}

func (a *App) toggleTheme() {
	if a.settings.ThemeMode == "dark" {
		a.settings.ThemeMode = "light"
	} else {
		a.settings.ThemeMode = "dark"
	}
	mygo.Theme.SetSource(a.themeSource())
	a.saveSettings()
}

func (a *App) setThemeMode(mode string) {
	a.settings.ThemeMode = mode
	mygo.Theme.SetSource(a.themeSource())
	a.saveSettings()
}

func (a *App) setFontSize(size string) {
	a.settings.FontSize = size
	a.saveSettings()
}

func (a *App) setLocale(tag string) {
	mygo.App.SetLocale(tag)
	a.locale = mygo.App.Locale()
}

// ---------- toasts and busy work ----------

func (a *App) toast(msg string) {
	if msg == "" {
		return
	}
	a.toasts = append(a.toasts, msg)
}

func (a *App) drainToasts(c *ui.Context) {
	for _, msg := range a.toasts {
		c.Toast(msg)
	}
	a.toasts = nil
}

// runAsync runs slow work (scrypt, dialogs, files) off the UI thread and
// applies its result on the main thread.
func (a *App) runAsync(label string, work func() OpResult, done func(OpResult)) {
	if a.busy {
		return
	}
	a.busy = true
	a.busyLabel = label
	win := a.win
	go func() {
		r := work()
		win.Update(func() {
			a.busy = false
			a.busyLabel = ""
			done(r)
		})
	}()
}

// ---------- lock lifecycle ----------

func (a *App) boot() {
	st := a.vault.GetStatus()
	win := a.win
	win.Update(func() {
		a.status = st
		if !st.Exists {
			a.phase = "setup"
			return
		}
		if !st.CanRemember {
			a.lockTab = 0
		}
		a.phase = "locked"
		a.refreshInfo()
	})
	if st.Exists && !a.settings.LockOnStartup && st.CanRemember {
		r := a.vault.UnlockRemembered()
		win.Update(func() {
			if r.OK {
				a.afterUnlock(r)
			}
		})
	}
}

func (a *App) afterUnlock(r OpResult) {
	if r.Payload != nil {
		a.entries = r.Payload.Entries
	} else {
		a.entries = nil
	}
	for i := range a.entries {
		a.normalizeEntry(&a.entries[i])
	}
	a.phase = "unlocked"
	a.password = ""
	a.confirm = ""
	a.failAttempts = 0
	a.lastError = ""
	a.cooldownUntil = time.Time{}
	a.resetSelection()
	a.unlockAt = time.Now()
	a.lastActivity = a.unlockAt
	a.refreshInfo()
	a.startIdleTimer()
	a.loadBackupsAsync()
	if a.remember {
		_ = a.vault.SetRemember(true)
		a.settings.LockOnStartup = false
		a.saveSettings()
	}
}

func (a *App) attemptUnlock(password string) {
	a.runAsync("unlock", func() OpResult { return a.vault.Unlock(password) }, func(r OpResult) {
		if r.OK {
			a.afterUnlock(r)
			return
		}
		switch r.Code {
		case "cooldown":
			a.cooldownUntil = time.Now().Add(time.Duration(r.CooldownMs) * time.Millisecond)
			a.failAttempts = 0
		case "wrong_password":
			a.failAttempts = r.AttemptsLeft
		default:
			a.lastError = r.Message
		}
		a.password = ""
	})
}

func (a *App) attemptHello() {
	a.runAsync("hello", func() OpResult { return a.vault.UnlockRemembered() }, func(r OpResult) {
		if r.OK {
			a.afterUnlock(r)
			return
		}
		a.lastError = r.Message
		a.lockTab = 0
	})
}

func (a *App) attemptCreate(password string) {
	a.runAsync("create", func() OpResult {
		r := a.vault.Create(password, a.remember)
		if r.OK {
			if a.remember {
				_ = a.vault.SetRemember(true)
			}
			return OpResult{OK: true, Payload: &Payload{Entries: []Entry{}}}
		}
		return r
	}, func(r OpResult) {
		if r.OK {
			if a.remember {
				a.settings.LockOnStartup = false
				a.saveSettings()
			}
			a.afterUnlock(r)
			a.toast(a.t("lock.vaultCreated"))
			return
		}
		a.lastError = r.Message
	})
}

func (a *App) lockVault() {
	if a.phase != "unlocked" {
		return
	}
	if a.settings.ClipClear && a.copiedAt.After(a.unlockAt) {
		mygo.Clipboard.Clear()
	}
	_ = a.vault.Lock()
	a.stopIdleTimer()
	a.entries = nil
	a.phase = "locked"
	a.resetSelection()
	a.password = ""
	a.confirm = ""
	a.showPassword = false
	a.lastError = ""
	a.lockTab = 0
	a.entryOpen = false
	a.deleteOpen = false
	a.restoreOpen = false
	a.importOpen = false
	a.backupPwOpen = false
	a.backupDeleteOpen = false
	a.notifOpen = false
}

func (a *App) deleteAllVault() {
	_ = a.vault.Destroy()
	a.entries = nil
	a.phase = "setup"
	a.resetSelection()
	a.stopIdleTimer()
	a.toast(a.t("settings.toast.vaultDeleted"))
}

// ---------- idle auto-lock ----------

func (a *App) idleDuration() time.Duration {
	if a.settings.AutoLockMinutes <= 0 {
		return 0
	}
	return time.Duration(a.settings.AutoLockMinutes) * time.Minute
}

func (a *App) touchActivity() {
	if a.phase != "unlocked" {
		return
	}
	a.lastActivity = time.Now()
	if d := a.idleDuration(); d > 0 && a.idleTimer != nil {
		a.idleTimer.Reset(d)
	}
}

func (a *App) startIdleTimer() {
	a.stopIdleTimer()
	d := a.idleDuration()
	if d <= 0 || a.win == nil {
		return
	}
	win := a.win
	a.idleTimer = time.AfterFunc(d, func() {
		win.Update(func() {
			if a.phase != "unlocked" {
				return
			}
			if time.Since(a.lastActivity) < d {
				return
			}
			a.lockVault()
		})
	})
}

func (a *App) stopIdleTimer() {
	if a.idleTimer != nil {
		a.idleTimer.Stop()
		a.idleTimer = nil
	}
}

// ---------- clipboard ----------

func (a *App) copyText(text, label string) {
	if text == "" {
		return
	}
	_ = mygo.Clipboard.Write(transfer.TextData(text), mygo.ClipboardOptions{Concealed: true})
	a.copiedAt = time.Now()
	seq := a.copySeq.Add(1)
	a.toast(a.tf("common.copiedField", map[string]any{"field": label}))
	secs := a.settings.ClipClearSecs
	if a.settings.ClipClear && secs > 0 {
		go func() {
			time.Sleep(time.Duration(secs) * time.Second)
			if a.copySeq.Load() == seq {
				mygo.Clipboard.Clear()
			}
		}()
	}
}

// ---------- entries ----------

func (a *App) openAddEntry(presetPassword string) {
	a.entryEditingID = ""
	a.entryForm = entryForm{Folder: "其他", Password: presetPassword}
	a.showFormPassword = false
	a.formError = ""
	a.entryOpen = true
}

func (a *App) openEditEntry(e *Entry) {
	if e == nil {
		return
	}
	a.entryEditingID = e.ID
	a.entryForm = entryForm{
		Site:     e.Site,
		Username: e.Username,
		Password: e.Password,
		URL:      e.URL,
		Notes:    e.Notes,
		Folder:   e.Folder,
	}
	a.showFormPassword = false
	a.formError = ""
	a.entryOpen = true
}

func (a *App) saveEntryForm() {
	f := a.entryForm
	if strings.TrimSpace(f.Site) == "" {
		a.formError = a.t("modal.siteRequired")
		return
	}
	if f.Password == "" {
		a.formError = a.t("modal.passwordRequired")
		return
	}
	if a.entryEditingID != "" {
		if e := a.entryByID(a.entryEditingID); e != nil {
			e.Site = strings.TrimSpace(f.Site)
			e.Username = strings.TrimSpace(f.Username)
			e.Password = f.Password
			e.URL = strings.TrimSpace(f.URL)
			e.Notes = strings.TrimSpace(f.Notes)
			e.Folder = f.Folder
			e.UpdatedAt = today()
			a.saveEntries()
			a.toast(a.t("vault.toastUpdated"))
		}
	} else {
		e := Entry{
			Site:       strings.TrimSpace(f.Site),
			Username:   strings.TrimSpace(f.Username),
			Password:   f.Password,
			URL:        strings.TrimSpace(f.URL),
			Notes:      strings.TrimSpace(f.Notes),
			Folder:     f.Folder,
			CreatedAt:  today(),
			UpdatedAt:  today(),
			LastUsedAt: time.Now().UnixMilli(),
		}
		a.normalizeEntry(&e)
		a.entries = append(a.entries, e)
		a.saveEntries()
		a.toast(a.t("vault.toastSaved"))
	}
	a.entryOpen = false
}

func (a *App) removeEntry(id string) {
	for i := range a.entries {
		if a.entries[i].ID == id {
			name := a.entries[i].Site
			a.entries = append(a.entries[:i], a.entries[i+1:]...)
			a.saveEntries()
			a.resetSelection()
			a.toast(a.tf("vault.toastDeleted", map[string]any{"name": name}))
			return
		}
	}
}

func (a *App) toggleFavorite(id string) {
	e := a.entryByID(id)
	if e == nil {
		return
	}
	e.Favorite = !e.Favorite
	a.saveEntries()
	if e.Favorite {
		a.toast(a.t("vault.toastAddedToFavorites"))
	} else {
		a.toast(a.t("vault.toastRemovedFromFavorites"))
	}
}

func (a *App) importEntries(list []Entry, overwrite bool) (added, skipped, updated int) {
	now := today()
	stamp := time.Now().UnixMilli()
	for _, raw := range list {
		site := strings.TrimSpace(raw.Site)
		if site == "" {
			continue
		}
		username := strings.TrimSpace(raw.Username)
		url := strings.TrimSpace(raw.URL)
		found := -1
		for i := range a.entries {
			if a.entries[i].Site == site && a.entries[i].Username == username && a.entries[i].URL == url {
				found = i
				break
			}
		}
		if found >= 0 {
			if overwrite {
				e := &a.entries[found]
				e.Password = raw.Password
				e.Notes = raw.Notes
				if raw.Folder != "" {
					e.Folder = raw.Folder
				}
				e.UpdatedAt = now
				updated++
			} else {
				skipped++
			}
			continue
		}
		e := raw
		e.Site = site
		e.Username = username
		e.URL = url
		e.CreatedAt = now
		e.UpdatedAt = now
		e.LastUsedAt = stamp
		a.normalizeEntry(&e)
		a.entries = append(a.entries, e)
		added++
	}
	if added > 0 || updated > 0 {
		a.saveEntries()
	}
	return
}

// ---------- backups ----------

func (a *App) loadBackupsAsync() {
	if a.win == nil || a.backupsLoading {
		return
	}
	a.backupsLoading = true
	win := a.win
	go func() {
		list := a.vault.BackupList()
		set := a.vault.BackupGetSettings()
		win.Update(func() {
			if list.OK {
				a.backups = list.Backups
			}
			if set.OK && set.Settings != nil {
				a.backupSetting = *set.Settings
			}
			a.backupsLoaded = true
			a.backupsLoading = false
		})
	}()
}

// backupDue tells whether no backup exists or the last one is older than the
// configured interval.
func (a *App) backupDue() (bool, int) {
	if !a.backupSetting.Enabled {
		return false, 0
	}
	interval := int64(a.backupSetting.IntervalHours * 3_600_000)
	if interval <= 0 {
		interval = 24 * 3_600_000
	}
	var last int64
	if len(a.backups) > 0 {
		last = a.backups[0].MTime
	}
	if last == 0 || time.Now().UnixMilli()-last > interval {
		days := 0
		if last > 0 {
			days = max(1, int((time.Now().UnixMilli()-last)/86_400_000))
		}
		return true, days
	}
	return false, 0
}

// ---------- notifications ----------

func (a *App) notificationIssues() []Issue {
	all := computeIssues(a.entries)
	list := make([]Issue, 0, len(all))
	for _, i := range all {
		switch i.Type {
		case "breached":
			if a.settings.NotifyBreach {
				list = append(list, i)
			}
		case "expired":
			if a.settings.NotifyExpiry {
				list = append(list, i)
			}
		default:
			if a.settings.NotifyWeak {
				list = append(list, i)
			}
		}
	}
	return list
}

func (a *App) notificationCount() int {
	n := len(a.notificationIssues())
	if a.settings.NotifyBackup {
		if due, _ := a.backupDue(); due {
			n++
		}
	}
	return n
}

// ---------- files and folders ----------

func (a *App) exportFileAsync() {
	label := a.t("transfer.exporting")
	if label == "transfer.exporting" {
		label = a.t("transfer.export")
	}
	a.runAsync(label, func() OpResult { return a.vault.ExportFile(context.Background()) }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		if r.Canceled {
			return
		}
		a.toast(a.tf("transfer.exportedBin", map[string]any{"path": r.FilePath}))
		a.pushActivity("export", a.t("transfer.encryptedBackup"), r.FilePath)
	})
}

func (a *App) exportCSVAsync() {
	if len(a.entries) == 0 {
		a.toast(a.t("transfer.emptyVault"))
		return
	}
	csv := buildCSV(a.entries)
	a.runAsync(a.t("transfer.export"), func() OpResult { return a.vault.ExportCsv(context.Background(), csv) }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		if r.Canceled {
			return
		}
		a.toast(a.tf("transfer.exportedCsv", map[string]any{"path": r.FilePath}))
		a.pushActivity("export", a.t("transfer.csvFile"), r.FilePath)
	})
}

func (a *App) pickCSVAsync() {
	a.runAsync(a.t("transfer.import"), func() OpResult { return a.vault.ImportCsv(context.Background()) }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		if r.Canceled {
			return
		}
		list := mapCSVToEntries(parseCSV(r.Content))
		if len(list) == 0 {
			a.toast(a.t("transfer.csvNoEntries"))
			return
		}
		a.importSource = a.tf("transfer.sourceCsv", map[string]any{"name": r.FileName})
		a.importList = list
		a.importMode = 0
		a.importOpen = true
	})
}

func (a *App) pickBackupAsync() {
	a.runAsync(a.t("transfer.import"), func() OpResult { return a.vault.PickBackup(context.Background()) }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		if r.Canceled {
			return
		}
		file, _ := r.File.(*VaultFile)
		a.backupFile = file
		a.backupName = r.FileName
		a.backupPw = ""
		a.backupPwOpen = true
	})
}

func (a *App) decryptBackupAsync() {
	file := a.backupFile
	if file == nil {
		return
	}
	pw := a.backupPw
	if pw == "" {
		a.toast(a.t("transfer.enterBackupPassword"))
		return
	}
	a.runAsync(a.t("transfer.decrypting"), func() OpResult {
		return a.vault.DecryptBackup(*file, pw)
	}, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		list := r.Entries
		for i := range list {
			if list[i].Folder == "" {
				list[i].Folder = "其他"
			}
		}
		a.importSource = a.tf("transfer.sourceBin", map[string]any{"name": a.backupName})
		a.importList = list
		a.importMode = 0
		a.importOpen = true
		a.backupPwOpen = false
		a.backupFile = nil
	})
}

func (a *App) backupNowAsync() {
	a.runAsync(a.t("backup.backingUp"), func() OpResult { return a.vault.BackupNow() }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		a.toast(a.tf("backup.created", map[string]any{"path": r.FilePath}))
		a.loadBackupsAsync()
	})
}

func (a *App) restoreBackupAsync() {
	name := a.restoreName
	pw := a.restorePw
	if name == "" {
		return
	}
	if pw == "" {
		a.toast(a.t("backup.enterMasterPassword"))
		return
	}
	a.runAsync(a.t("backup.restoring"), func() OpResult {
		return a.vault.BackupRestore(name, pw)
	}, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		a.restoreOpen = false
		a.restorePw = ""
		a.toast(a.t("backup.restored"))
		a.lockVault()
		a.page = "vault"
	})
}

func (a *App) deleteBackupAsync(name string) {
	a.runAsync(a.t("backup.title"), func() OpResult { return a.vault.BackupDelete(name) }, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		a.toast(a.t("backup.deleted"))
		a.loadBackupsAsync()
	})
}

func (a *App) changeMasterPasswordAsync() {
	if a.curPw == "" {
		a.toast(a.t("settings.toast.enterCurrentPassword"))
		return
	}
	if len([]rune(a.newPw)) < 8 {
		a.toast(a.t("settings.toast.newPasswordTooShort"))
		return
	}
	if a.newPw != a.confirmPw {
		a.toast(a.t("settings.toast.passwordMismatch"))
		return
	}
	cur, next := a.curPw, a.newPw
	a.runAsync(a.t("settings.security.reencrypting"), func() OpResult {
		return a.vault.ChangeMasterPassword(cur, next)
	}, func(r OpResult) {
		if !r.OK {
			a.toast(r.Message)
			return
		}
		a.curPw, a.newPw, a.confirmPw = "", "", ""
		a.toast(a.t("settings.toast.masterPasswordUpdated"))
	})
}

func (a *App) pushActivity(kind, name, detail string) {
	a.activity = append([]activity{{
		Kind:   kind,
		Name:   name,
		Detail: detail,
		At:     time.Now().Format("15:04:05"),
		OK:     true,
	}}, a.activity...)
	if len(a.activity) > 6 {
		a.activity = a.activity[:6]
	}
}

func (a *App) openDataFolder() {
	dir := a.vault.dataDir()
	_ = os.MkdirAll(dir, 0o755)
	_ = mygo.Shell.OpenPath(dir)
}

func (a *App) openBackupFolder() {
	dir := a.vault.backupDir()
	_ = os.MkdirAll(dir, 0o755)
	_ = mygo.Shell.OpenPath(dir)
}

// folderLabel says a stored folder name in the page's language.
func (a *App) folderLabel(f string) string {
	switch f {
	case "工作":
		return a.t("nav.folderWork")
	case "个人":
		return a.t("nav.folderPersonal")
	case "财务":
		return a.t("nav.folderFinance")
	case "社交":
		return a.t("nav.folderSocial")
	case "购物":
		return a.t("nav.folderShopping")
	case "其他":
		return a.t("nav.folderOther")
	}
	return f
}

func (a *App) riskLabel(r string) string {
	switch r {
	case "high":
		return a.t("risk.high")
	case "medium":
		return a.t("risk.medium")
	default:
		return a.t("risk.low")
	}
}
