package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	maxAttempts   = 5
	cooldownMs    = 60_000
	rememberKeyID = "vault-remember-key"
)

// OpResult is the reply of every vault method. The frontend reads `ok`,
// `code`, `message` and the other fields the operation provides.
type OpResult struct {
	OK           bool                `json:"ok"`
	Code         string              `json:"code,omitempty"`
	Message      string              `json:"message,omitempty"`
	AttemptsLeft int                 `json:"attemptsLeft,omitempty"`
	CooldownMs   int64               `json:"cooldownMs,omitempty"`
	Payload      *Payload            `json:"payload,omitempty"`
	Canceled     bool                `json:"canceled,omitempty"`
	FilePath     string              `json:"filePath,omitempty"`
	Content      string              `json:"content,omitempty"`
	FileName     string              `json:"fileName,omitempty"`
	File         any                 `json:"file,omitempty"`
	Entries      []Entry             `json:"entries,omitempty"`
	Backups      []BackupInfo        `json:"backups,omitempty"`
	Settings     *BackupSettingsView `json:"settings,omitempty"`
	Dir          string              `json:"dir,omitempty"`
	Exists       bool                `json:"exists,omitempty"`
	SizeBytes    int64               `json:"sizeBytes,omitempty"`
	CreatedAt    string              `json:"createdAt,omitempty"`
	UpdatedAt    string              `json:"updatedAt,omitempty"`
	CanRemember  bool                `json:"canRemember,omitempty"`
	Unlocked     bool                `json:"unlocked,omitempty"`
}

func okResult() OpResult { return OpResult{OK: true} }

func failResult(code, message string) OpResult {
	return OpResult{Code: code, Message: message}
}

// Vault is the password vault service the frontend calls.
type Vault struct {
	mu             sync.Mutex
	parentMu       sync.RWMutex
	parent         *mygo.Window
	file           *VaultFile
	key            []byte
	failCount      int
	cooldownUntil  int64
	backupSettings *backupConfig
}

func NewVault() *Vault { return &Vault{} }

// SetParent gives the native file dialogs the window they belong to, so
// they open over it, as modal sheets and owned dialogs do.
func (v *Vault) SetParent(w *mygo.Window) {
	v.parentMu.Lock()
	v.parent = w
	v.parentMu.Unlock()
}

func (v *Vault) parentWindow() *mygo.Window {
	v.parentMu.RLock()
	defer v.parentMu.RUnlock()
	return v.parent
}

func (v *Vault) withVaultFileLocked() *VaultFile {
	if v.file == nil {
		v.file = readVaultFileFrom(v.dataFile())
	}
	return v.file
}

func readVaultFileFrom(path string) *VaultFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f VaultFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil
	}
	return &f
}

// writeVaultFileLocked writes atomically: temporary file, then rename.
func (v *Vault) writeVaultFileLocked(file *VaultFile) error {
	if err := os.MkdirAll(v.dataDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return err
	}
	path := v.dataFile()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func secretsAvailable() bool {
	_, err := mygo.Secrets.Get(rememberKeyID)
	if err == nil {
		return true
	}
	return errors.Is(err, mygo.ErrSecretNotFound)
}

func buildRememberKey(key []byte) error {
	return mygo.Secrets.Set(rememberKeyID, []byte(base64.StdEncoding.EncodeToString(key)))
}

func readRememberKey() []byte {
	raw, err := mygo.Secrets.Get(rememberKeyID)
	if err != nil {
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return nil
	}
	return key
}

func deleteRememberKey() {
	_ = mygo.Secrets.Delete(rememberKeyID)
}

func rememberKeyExists() bool {
	_, err := mygo.Secrets.Get(rememberKeyID)
	return err == nil
}

// GetStatus reports whether a vault exists and how it can be unlocked.
func (v *Vault) GetStatus() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	f := v.withVaultFileLocked()
	return OpResult{
		OK:          true,
		Exists:      f != nil,
		CanRemember: secretsAvailable(),
		Unlocked:    v.key != nil,
	}
}

// Create makes a new vault and leaves it unlocked.
func (v *Vault) Create(password string, remember bool) OpResult {
	if utf8.RuneCountInString(password) < 8 {
		return failResult("invalid", "主密码至少 8 位")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, err := os.Stat(v.dataFile()); err == nil {
		return failResult("exists", "保险库已存在")
	}
	file, key, err := createVaultFile(password)
	if err != nil {
		return failResult("error", err.Error())
	}
	v.file = file
	v.failCount = 0
	v.key = key
	if remember && buildRememberKey(key) != nil {
		// Remembering is best effort, as it is in the Electron app.
	}
	if err := v.writeVaultFileLocked(file); err != nil {
		return failResult("error", err.Error())
	}
	return okResult()
}

// Unlock opens the vault with the master password.
func (v *Vault) Unlock(password string) OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	if cd := v.cooldownUntil - time.Now().UnixMilli(); cd > 0 {
		return OpResult{Code: "cooldown", CooldownMs: cd, Message: "尝试次数过多，请稍后重试"}
	}
	file := v.withVaultFileLocked()
	if file == nil {
		return failResult("not_found", "保险库不存在")
	}
	key, err := unlockFile(file, password)
	if err != nil {
		v.failCount++
		if v.failCount >= maxAttempts {
			v.failCount = 0
			v.cooldownUntil = time.Now().UnixMilli() + cooldownMs
			return OpResult{Code: "cooldown", CooldownMs: cooldownMs, Message: "尝试次数过多，请 60 秒后重试"}
		}
		return OpResult{Code: "wrong_password", AttemptsLeft: maxAttempts - v.failCount, Message: "主密码错误"}
	}
	v.failCount = 0
	v.key = key
	v.maybeAutoBackupLocked()
	payload, err := decryptPayload(key, file)
	if err != nil {
		return failResult("error", err.Error())
	}
	return OpResult{OK: true, Payload: &payload}
}

// UnlockRemembered opens the vault with the key the system credential store
// kept, so that auto unlock and Windows Hello work without the password.
func (v *Vault) UnlockRemembered() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	file := v.withVaultFileLocked()
	if file == nil {
		return failResult("not_found", "保险库不存在")
	}
	key := readRememberKey()
	if key == nil {
		return failResult("no_remember", "没有可用的记住密钥")
	}
	if !verifyPassword(key, file.Verifier) {
		return failResult("no_remember", "记住的密钥已失效，请使用主密码解锁")
	}
	v.key = key
	v.failCount = 0
	v.maybeAutoBackupLocked()
	payload, err := decryptPayload(key, file)
	if err != nil {
		return failResult("error", err.Error())
	}
	return OpResult{OK: true, Payload: &payload}
}

// Lock drops the session key.
func (v *Vault) Lock() OpResult {
	v.mu.Lock()
	v.key = nil
	v.mu.Unlock()
	return okResult()
}

// SaveData encrypts and stores the entries.
func (v *Vault) SaveData(payload Payload) OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return failResult("locked", "保险库已锁定")
	}
	file := v.withVaultFileLocked()
	if file == nil {
		return failResult("not_found", "保险库不存在")
	}
	env, err := encryptPayload(v.key, payload)
	if err != nil {
		return failResult("error", err.Error())
	}
	file.Vault = env
	file.UpdatedAt = nowISO()
	if err := v.writeVaultFileLocked(file); err != nil {
		return failResult("error", err.Error())
	}
	v.maybeAutoBackupLocked()
	return okResult()
}

// SetRemember keeps or forgets the key in the system credential store.
func (v *Vault) SetRemember(enabled bool) OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return failResult("locked", "保险库已锁定")
	}
	if enabled {
		if !secretsAvailable() {
			return failResult("unsupported", "当前系统不支持安全存储记住密钥")
		}
		if err := buildRememberKey(v.key); err != nil {
			return failResult("error", err.Error())
		}
	} else {
		deleteRememberKey()
		file := v.withVaultFileLocked()
		if file != nil {
			file.RememberKey = nil
			_ = v.writeVaultFileLocked(file)
		}
	}
	return okResult()
}

// ChangeMasterPassword re-encrypts the vault under a new password.
func (v *Vault) ChangeMasterPassword(current, next string) OpResult {
	if utf8.RuneCountInString(next) < 8 {
		return failResult("invalid", "新主密码至少 8 位")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	file := v.withVaultFileLocked()
	if file == nil {
		return failResult("not_found", "保险库不存在")
	}
	oldKey, err := unlockFile(file, current)
	if err != nil {
		return failResult("wrong_password", "当前主密码错误")
	}
	payload, err := decryptPayload(oldKey, file)
	if err != nil {
		return failResult("error", err.Error())
	}
	nf, newKey, err := createVaultFile(next)
	if err != nil {
		return failResult("error", err.Error())
	}
	env, err := encryptPayload(newKey, payload)
	if err != nil {
		return failResult("error", err.Error())
	}
	nf.Vault = env
	nf.CreatedAt = file.CreatedAt
	if rememberKeyExists() {
		_ = buildRememberKey(newKey)
	}
	if err := v.writeVaultFileLocked(nf); err != nil {
		return failResult("error", err.Error())
	}
	v.file = nf
	v.key = newKey
	return okResult()
}

// Repair checks that the encrypted payload can still be decrypted.
func (v *Vault) Repair() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return failResult("locked", "请先解锁保险库")
	}
	file := v.withVaultFileLocked()
	if _, err := decryptPayload(v.key, file); err != nil {
		return failResult("corrupt", "数据库损坏："+err.Error())
	}
	return OpResult{OK: true, Message: "数据库完整，未发现错误"}
}

// Destroy deletes the vault file and forgets the key.
func (v *Vault) Destroy() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	_ = os.Remove(v.dataFile())
	deleteRememberKey()
	v.file = nil
	v.key = nil
	v.failCount = 0
	return okResult()
}

// ExportFile saves a copy of the encrypted vault file where the user chooses.
func (v *Vault) ExportFile(ctx context.Context) OpResult {
	path := v.dataFile()
	if _, err := os.Stat(path); err != nil {
		return failResult("not_found", "保险库文件不存在")
	}
	docs, _ := mygo.App.Path(mygo.PathDocuments)
	dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      v.parentWindow(),
		Title:       "导出加密保险库",
		DefaultPath: filepath.Join(docs, "vault-backup-"+time.Now().Format("2006-01-02")+".bin"),
		Filters:     []mygo.FileFilter{{Name: "守密 加密文件", Extensions: []string{"bin"}}},
	})
	if err != nil {
		return failResult("error", err.Error())
	}
	if dest == "" {
		return OpResult{OK: true, Canceled: true}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return failResult("error", err.Error())
	}
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return failResult("error", err.Error())
	}
	return OpResult{OK: true, FilePath: dest}
}

// ExportCsv saves the entries as CSV, with a UTF-8 BOM for Excel.
func (v *Vault) ExportCsv(ctx context.Context, csv string) OpResult {
	docs, _ := mygo.App.Path(mygo.PathDocuments)
	dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      v.parentWindow(),
		Title:       "导出为 CSV",
		DefaultPath: filepath.Join(docs, "vault-export-"+time.Now().Format("2006-01-02")+".csv"),
		Filters:     []mygo.FileFilter{{Name: "CSV 文件", Extensions: []string{"csv"}}},
	})
	if err != nil {
		return failResult("error", err.Error())
	}
	if dest == "" {
		return OpResult{OK: true, Canceled: true}
	}
	if err := os.WriteFile(dest, []byte("\uFEFF"+csv), 0o600); err != nil {
		return failResult("error", err.Error())
	}
	return OpResult{OK: true, FilePath: dest}
}

// decodeCsvBuffer reads UTF-8 with or without a BOM, else GBK, which Excel on
// Chinese Windows writes by default.
func decodeCsvBuffer(buf []byte) string {
	if len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		return string(buf[3:])
	}
	if utf8.Valid(buf) {
		return string(buf)
	}
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(buf); err == nil {
		return string(decoded)
	}
	return string(buf)
}

// ImportCsv lets the user pick a CSV file and returns its text.
func (v *Vault) ImportCsv(ctx context.Context) OpResult {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  v.parentWindow(),
		Title:   "选择要导入的 CSV 文件",
		Filters: []mygo.FileFilter{{Name: "CSV 文件", Extensions: []string{"csv", "txt"}}},
	})
	if err != nil {
		return failResult("error", err.Error())
	}
	if len(paths) == 0 {
		return OpResult{OK: true, Canceled: true}
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		return failResult("error", err.Error())
	}
	return OpResult{
		OK:       true,
		Content:  decodeCsvBuffer(raw),
		FilePath: paths[0],
		FileName: filepath.Base(paths[0]),
	}
}

// PickBackup lets the user choose an encrypted backup and returns its content.
func (v *Vault) PickBackup(ctx context.Context) OpResult {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  v.parentWindow(),
		Title:   "选择要导入的加密备份文件",
		Filters: []mygo.FileFilter{{Name: "守密 加密文件", Extensions: []string{"bin"}}},
	})
	if err != nil {
		return failResult("error", err.Error())
	}
	if len(paths) == 0 {
		return OpResult{OK: true, Canceled: true}
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		return failResult("bad_file", "无法读取备份文件："+err.Error())
	}
	var file VaultFile
	if err := json.Unmarshal(raw, &file); err != nil || validateVaultFile(&file) != nil {
		return failResult("bad_file", "不是有效的保险库备份文件")
	}
	return OpResult{OK: true, File: &file, FileName: filepath.Base(paths[0])}
}

// DecryptBackup opens a backup with its own master password and returns the
// plaintext entries for the page to merge.
func (v *Vault) DecryptBackup(file VaultFile, password string) OpResult {
	if err := validateVaultFile(&file); err != nil {
		return failResult("bad_file", "不是有效的保险库备份文件")
	}
	key, err := unlockFile(&file, password)
	if err != nil {
		return failResult("wrong_password", "备份文件主密码错误")
	}
	payload, err := decryptPayload(key, &file)
	if err != nil {
		return failResult("bad_file", "备份文件解密失败："+err.Error())
	}
	return OpResult{OK: true, Entries: payload.Entries}
}

// OpenDataFolder shows the vault directory in the system file manager.
func (v *Vault) OpenDataFolder() OpResult {
	dir := v.dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return failResult("error", err.Error())
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		return failResult("error", err.Error())
	}
	return okResult()
}

// GetInfo describes the vault file for the settings and status pages.
func (v *Vault) GetInfo() OpResult {
	dir := v.dataDir()
	path := v.dataFile()
	st, err := os.Stat(path)
	if err != nil {
		return OpResult{OK: true, Dir: dir, Exists: false}
	}
	var createdAt, updatedAt string
	if file := readVaultFileFrom(path); file != nil {
		createdAt = file.CreatedAt
		updatedAt = file.UpdatedAt
	}
	return OpResult{
		OK:        true,
		Dir:       dir,
		File:      path,
		Exists:    true,
		SizeBytes: st.Size(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// ClearClipboard empties the system clipboard.
func (v *Vault) ClearClipboard() OpResult {
	mygo.Clipboard.Clear()
	return okResult()
}

// CopyText copies confidential text and keeps it out of clipboard history
// where the platform supports the mark.
func (v *Vault) CopyText(text string) {
	_ = mygo.Clipboard.Write(transfer.TextData(text), mygo.ClipboardOptions{Concealed: true})
}

// BackupNow makes a local backup of the encrypted vault.
func (v *Vault) BackupNow() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	path, err := v.backupNowLocked()
	if err != nil {
		return failResult("not_found", err.Error())
	}
	return OpResult{OK: true, FilePath: path}
}

// BackupList lists the local backups, newest first.
func (v *Vault) BackupList() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	return OpResult{OK: true, Backups: v.listBackupsLocked()}
}

// BackupDelete removes one local backup.
func (v *Vault) BackupDelete(name string) OpResult {
	if !validBackupName(name) {
		return failResult("not_found", "备份文件不存在")
	}
	path := filepath.Join(v.backupDir(), name)
	if _, err := os.Stat(path); err != nil {
		return failResult("not_found", "备份文件不存在")
	}
	if err := os.Remove(path); err != nil {
		return failResult("error", err.Error())
	}
	return okResult()
}

// BackupRestore replaces the current vault with a backup after checking its
// master password.
func (v *Vault) BackupRestore(name, password string) OpResult {
	if !validBackupName(name) {
		return failResult("not_found", "备份文件不存在")
	}
	path := filepath.Join(v.backupDir(), name)
	if _, err := os.Stat(path); err != nil {
		return failResult("not_found", "备份文件不存在")
	}
	if utf8.RuneCountInString(password) < 4 {
		return failResult("invalid", "请输入主密码")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return failResult("bad_file", "恢复失败："+err.Error())
	}
	var backup VaultFile
	if err := json.Unmarshal(raw, &backup); err != nil || validateVaultFile(&backup) != nil {
		return failResult("bad_file", "不是有效的保险库备份文件")
	}
	key, err := unlockFile(&backup, password)
	if err != nil {
		return failResult("wrong_password", "主密码错误，或该备份由旧主密码加密")
	}
	if _, err := decryptPayload(key, &backup); err != nil {
		return failResult("bad_file", "恢复失败："+err.Error())
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.writeVaultFileLocked(&backup); err != nil {
		return failResult("bad_file", "恢复失败："+err.Error())
	}
	v.file = &backup
	v.key = key
	if rememberKeyExists() {
		_ = buildRememberKey(key)
	}
	return OpResult{OK: true, Message: "已恢复到所选备份版本"}
}

// BackupOpenFolder shows the backups directory in the file manager.
func (v *Vault) BackupOpenFolder() OpResult {
	dir := v.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return failResult("error", err.Error())
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		return failResult("error", err.Error())
	}
	return okResult()
}

// BackupGetSettings returns the automatic backup settings.
func (v *Vault) BackupGetSettings() OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	cfg := v.readBackupSettingsLocked()
	return OpResult{OK: true, Settings: &BackupSettingsView{
		Enabled:       cfg.Enabled,
		IntervalHours: float64(cfg.IntervalMs) / 3_600_000,
		MaxKeep:       cfg.MaxKeep,
	}}
}

// BackupSaveSettings stores the automatic backup settings.
func (v *Vault) BackupSaveSettings(enabled bool, intervalHours float64, maxKeep int) OpResult {
	v.mu.Lock()
	defer v.mu.Unlock()
	cfg := v.readBackupSettingsLocked()
	cfg.Enabled = enabled
	if intervalHours >= 1 {
		cfg.IntervalMs = int64(intervalHours * 3_600_000)
	}
	if maxKeep >= 1 {
		cfg.MaxKeep = maxKeep
	}
	if err := v.writeBackupSettingsLocked(cfg); err != nil {
		return failResult("error", err.Error())
	}
	return okResult()
}
