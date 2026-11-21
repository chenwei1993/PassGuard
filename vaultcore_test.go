package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/egoist/mygo"
)

// TestUnlocksElectronVault proves the Go backend opens vault files the
// Electron app wrote: same scrypt key derivation and AES-256-GCM envelopes.
func TestUnlocksElectronVault(t *testing.T) {
	raw, err := os.ReadFile("testdata/electron-vault.json")
	if err != nil {
		t.Fatal(err)
	}
	var file VaultFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	key, err := unlockFile(&file, "test-password-123")
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	payload, err := decryptPayload(key, &file)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if len(payload.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(payload.Entries))
	}
	if payload.Entries[0].Site != "GitHub" {
		t.Fatalf("site = %v", payload.Entries[0].Site)
	}
	if _, err := unlockFile(&file, "wrong-password"); err == nil {
		t.Fatal("wrong password unlocked the vault")
	}
}

func TestVaultRoundTrip(t *testing.T) {
	file, key, err := createVaultFile("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	entries := []Entry{
		{Site: "GitHub", Username: "alice", Password: "p@ss", LastUsedAt: 1700000000000},
	}
	env, err := encryptPayload(key, Payload{Entries: entries})
	if err != nil {
		t.Fatal(err)
	}
	file.Vault = env

	opened, err := unlockFile(file, "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := decryptPayload(opened, file)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Entries) != 1 || payload.Entries[0].Site != "GitHub" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestDecryptRejectsTampering(t *testing.T) {
	file, key, err := createVaultFile("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	env, err := encryptPayload(key, Payload{Entries: []Entry{{Site: "GitHub"}}})
	if err != nil {
		t.Fatal(err)
	}
	// Flip one base64 character of the ciphertext.
	data := []byte(env.Data)
	if data[0] == 'A' {
		data[0] = 'B'
	} else {
		data[0] = 'A'
	}
	env.Data = string(data)
	file.Vault = env
	if _, err := decryptPayload(key, file); err == nil {
		t.Fatal("tampered payload decrypted")
	}
}

func TestBackupNames(t *testing.T) {
	for _, name := range []string{"vault-backup-20240101-120000.bin", "vault-backup-1-2.bin"} {
		if !validBackupName(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range []string{"vault-backup-.bin", "../vault-backup-20240101-120000.bin", "vault.bin"} {
		if validBackupName(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}

func TestVaultLifecycle(t *testing.T) {
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	v := NewVault()

	if r := v.GetStatus(); !r.OK || r.Exists {
		t.Fatalf("fresh status = %#v", r)
	}
	if r := v.Create("password123", false); !r.OK {
		t.Fatalf("create = %#v", r)
	}
	if r := v.SaveData(Payload{Entries: []Entry{{Site: "GitHub", Password: "p@ss"}}}); !r.OK {
		t.Fatalf("save = %#v", r)
	}
	if r := v.Lock(); !r.OK {
		t.Fatalf("lock = %#v", r)
	}
	if r := v.Unlock("wrong-password"); r.OK || r.AttemptsLeft != 4 {
		t.Fatalf("wrong unlock = %#v", r)
	}
	r := v.Unlock("password123")
	if !r.OK || r.Payload == nil || len(r.Payload.Entries) != 1 {
		t.Fatalf("unlock = %#v", r)
	}

	if r := v.BackupNow(); !r.OK || r.FilePath == "" {
		t.Fatalf("backup now = %#v", r)
	}
	list := v.BackupList()
	if !list.OK || len(list.Backups) != 1 {
		t.Fatalf("backup list = %#v", list)
	}
	name := list.Backups[0].Name

	if r := v.ChangeMasterPassword("password123", "next-password"); !r.OK {
		t.Fatalf("change password = %#v", r)
	}
	if r := v.Lock(); !r.OK {
		t.Fatal("lock failed")
	}
	if r := v.Unlock("password123"); r.OK {
		t.Fatal("old password still unlocks")
	}
	if r := v.Unlock("next-password"); !r.OK {
		t.Fatalf("new password: %#v", r)
	}

	if r := v.BackupRestore(name, "password123"); !r.OK {
		t.Fatalf("restore = %#v", r)
	}
	if r := v.Lock(); !r.OK {
		t.Fatal("lock failed")
	}
	if r := v.Unlock("password123"); !r.OK {
		t.Fatalf("restored password: %#v", r)
	}
}
