package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/scrypt"
)

// The vault file format is the one the Electron app wrote, so existing
// vault.bin files keep working: scrypt(主密码) → AES-256-GCM envelopes.
const (
	vaultVersion  = 1
	verifierPlain = "PasswordVault.verifier.v1"
)

// KDFParams describes the scrypt parameters stored in the vault file.
type KDFParams struct {
	Algo   string `json:"algo"`
	N      int    `json:"N"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	Keylen int    `json:"keylen"`
	Salt   string `json:"salt"`
}

// Envelope is one AES-256-GCM encrypted value; iv, tag and data are base64.
type Envelope struct {
	IV   string `json:"iv"`
	Tag  string `json:"tag"`
	Data string `json:"data"`
}

// VaultFile is the on-disk format of the vault.
type VaultFile struct {
	Version     int       `json:"version"`
	KDF         KDFParams `json:"kdf"`
	Verifier    Envelope  `json:"verifier"`
	Vault       *Envelope `json:"vault"`
	RememberKey *string   `json:"rememberKey"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
}

// Entry is one password, as the vault file stores it.
type Entry struct {
	ID         string   `json:"id"`
	Site       string   `json:"site"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	URL        string   `json:"url"`
	Notes      string   `json:"notes"`
	Folder     string   `json:"folder"`
	Tags       []string `json:"tags"`
	Favorite   bool     `json:"favorite"`
	Breached   bool     `json:"breached"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
	UsageCount int      `json:"usageCount"`
	LastUsedAt int64    `json:"lastUsedAt"`
}

// Payload is the plaintext the vault file holds.
type Payload struct {
	Entries []Entry `json:"entries"`
}

func defaultKDF() KDFParams {
	return KDFParams{Algo: "scrypt", N: 65536, R: 8, P: 1, Keylen: 32}
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

func deriveKey(password string, salt []byte, p KDFParams) ([]byte, error) {
	n, r, pp, keylen := p.N, p.R, p.P, p.Keylen
	if n == 0 {
		n = 65536
	}
	if r == 0 {
		r = 8
	}
	if pp == 0 {
		pp = 1
	}
	if keylen == 0 {
		keylen = 32
	}
	return scrypt.Key([]byte(password), salt, n, r, pp, keylen)
}

func encryptEnvelope(key, plaintext []byte) (Envelope, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return Envelope{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Envelope{}, err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return Envelope{}, err
	}
	sealed := gcm.Seal(nil, iv, plaintext, nil)
	n := len(sealed) - gcm.Overhead()
	return Envelope{IV: b64(iv), Tag: b64(sealed[n:]), Data: b64(sealed[:n])}, nil
}

func decryptEnvelope(key []byte, env Envelope) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	iv, err := unb64(env.IV)
	if err != nil {
		return nil, err
	}
	tag, err := unb64(env.Tag)
	if err != nil {
		return nil, err
	}
	data, err := unb64(env.Data)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, iv, append(data, tag...), nil)
}

func makeVerifier(key []byte) (Envelope, error) {
	return encryptEnvelope(key, []byte(verifierPlain))
}

func verifyPassword(key []byte, verifier Envelope) bool {
	plain, err := decryptEnvelope(key, verifier)
	if err != nil {
		return false
	}
	expect := []byte(verifierPlain)
	return len(plain) == len(expect) && subtle.ConstantTimeCompare(plain, expect) == 1
}

// createVaultFile derives a fresh salt and key from password.
func createVaultFile(password string) (*VaultFile, []byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, err
	}
	kdf := defaultKDF()
	kdf.Salt = b64(salt)
	key, err := deriveKey(password, salt, kdf)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := makeVerifier(key)
	if err != nil {
		return nil, nil, err
	}
	now := nowISO()
	return &VaultFile{
		Version:   vaultVersion,
		KDF:       kdf,
		Verifier:  verifier,
		Vault:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}, key, nil
}

// unlockFile returns the key when password opens the file.
func unlockFile(file *VaultFile, password string) ([]byte, error) {
	salt, err := unb64(file.KDF.Salt)
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(password, salt, file.KDF)
	if err != nil {
		return nil, err
	}
	if !verifyPassword(key, file.Verifier) {
		return nil, errors.New("wrong password")
	}
	return key, nil
}

func encryptPayload(key []byte, payload Payload) (*Envelope, error) {
	if payload.Entries == nil {
		payload.Entries = []Entry{}
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	env, err := encryptEnvelope(key, plain)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func decryptPayload(key []byte, file *VaultFile) (Payload, error) {
	if file == nil || file.Vault == nil {
		return Payload{Entries: []Entry{}}, nil
	}
	plain, err := decryptEnvelope(key, *file.Vault)
	if err != nil {
		return Payload{}, err
	}
	var payload Payload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return Payload{}, err
	}
	if payload.Entries == nil {
		payload.Entries = []Entry{}
	}
	return payload, nil
}

func validateVaultFile(file *VaultFile) error {
	if file == nil || file.Version == 0 || file.KDF.Salt == "" {
		return fmt.Errorf("not a vault file")
	}
	return nil
}
