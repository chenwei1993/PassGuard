package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// backupConfig is what backup-settings.json holds, as the Electron app wrote it.
type backupConfig struct {
	Enabled    bool  `json:"enabled"`
	IntervalMs int64 `json:"intervalMs"`
	MaxKeep    int   `json:"maxKeep"`
}

func defaultBackupConfig() backupConfig {
	return backupConfig{Enabled: false, IntervalMs: 24 * 60 * 60 * 1000, MaxKeep: 10}
}

// BackupInfo describes one file in the backups directory.
type BackupInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
}

// BackupSettingsView is the shape the settings page reads and writes.
type BackupSettingsView struct {
	Enabled       bool    `json:"enabled"`
	IntervalHours float64 `json:"intervalHours"`
	MaxKeep       int     `json:"maxKeep"`
}

func (v *Vault) dataDir() string {
	return filepath.Join(userDataDir(), "PasswordVault")
}

func (v *Vault) dataFile() string {
	return filepath.Join(v.dataDir(), "vault.bin")
}

func (v *Vault) backupDir() string {
	return filepath.Join(userDataDir(), "Backups")
}

func (v *Vault) backupSettingsFile() string {
	return filepath.Join(userDataDir(), "backup-settings.json")
}

var backupNameRe = regexp.MustCompile(`^vault-backup-[\d-]+\.bin$`)

// readBackupSettings reads the settings once. Callers hold v.mu.
func (v *Vault) readBackupSettingsLocked() backupConfig {
	if v.backupSettings != nil {
		return *v.backupSettings
	}
	cfg := defaultBackupConfig()
	raw, err := os.ReadFile(v.backupSettingsFile())
	if err == nil {
		_ = json.Unmarshal(raw, &cfg)
	}
	v.backupSettings = &cfg
	return cfg
}

// writeBackupSettingsLocked stores the settings. Callers hold v.mu.
func (v *Vault) writeBackupSettingsLocked(cfg backupConfig) error {
	v.backupSettings = &cfg
	if err := os.MkdirAll(userDataDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(v.backupSettingsFile(), raw, 0o600)
}

func backupTimestamp(t time.Time) string {
	return t.Format("20060102-150405")
}

// backupNowLocked copies the encrypted vault file into the backups directory.
// Callers hold v.mu.
func (v *Vault) backupNowLocked() (string, error) {
	src := v.dataFile()
	if _, err := os.Stat(src); err != nil {
		return "", errors.New("保险库文件不存在")
	}
	if err := os.MkdirAll(v.backupDir(), 0o755); err != nil {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(v.backupDir(), "vault-backup-"+backupTimestamp(time.Now())+".bin")
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", err
	}
	v.trimOldBackupsLocked()
	return dest, nil
}

// listBackupsLocked lists backups newest first. Callers hold v.mu.
func (v *Vault) listBackupsLocked() []BackupInfo {
	if err := os.MkdirAll(v.backupDir(), 0o755); err != nil {
		return []BackupInfo{}
	}
	entries, err := os.ReadDir(v.backupDir())
	if err != nil {
		return []BackupInfo{}
	}
	list := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !backupNameRe.MatchString(name) {
			continue
		}
		p := filepath.Join(v.backupDir(), name)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		list = append(list, BackupInfo{
			Name:  name,
			Path:  p,
			Size:  st.Size(),
			MTime: st.ModTime().UnixMilli(),
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].MTime > list[j].MTime })
	return list
}

// trimOldBackupsLocked keeps the newest maxKeep files. Callers hold v.mu.
func (v *Vault) trimOldBackupsLocked() {
	cfg := v.readBackupSettingsLocked()
	list := v.listBackupsLocked()
	if cfg.MaxKeep <= 0 {
		cfg.MaxKeep = 10
	}
	for _, b := range list[min(cfg.MaxKeep, len(list)):] {
		_ = os.Remove(b.Path)
	}
}

// maybeAutoBackupLocked backs up after a change when the interval passed.
// Callers hold v.mu.
func (v *Vault) maybeAutoBackupLocked() {
	cfg := v.readBackupSettingsLocked()
	if !cfg.Enabled {
		return
	}
	list := v.listBackupsLocked()
	var last int64
	if len(list) > 0 {
		last = list[0].MTime
	}
	if time.Now().UnixMilli()-last < cfg.IntervalMs {
		return
	}
	_, _ = v.backupNowLocked()
}

func validBackupName(name string) bool {
	return backupNameRe.MatchString(name) && filepath.Base(name) == name
}
