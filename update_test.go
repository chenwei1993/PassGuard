package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v2.4.2", "2.4.1", true},
		{"2.4.1", "2.4.1", false},
		{"V2.10.0", "2.9.9", true},
		{"2.4", "2.4.0", false},
		{"v2.5.0-beta", "2.4.9", true},
		{"v2.4.1", "2.4.2", false},
		{"", "2.4.1", false},
	}
	for _, c := range cases {
		if got := isNewerVersion(c.latest, c.current); got != c.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestUpdateFromRelease(t *testing.T) {
	rel := githubRelease{
		TagName: "v2.4.2",
		Name:    "守密 PassGuard 2.4.2",
		HTMLURL: "https://example.com/release",
		Assets: []struct {
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "SHA256SUMS.txt", BrowserDownloadURL: "https://example.com/sums"},
			{Name: "PassGuard.exe", Size: 12, BrowserDownloadURL: "https://example.com/portable"},
			{Name: "passguard-setup-2.4.2-amd64.exe", Size: 42, BrowserDownloadURL: "https://example.com/setup"},
		},
	}
	info := updateFromRelease(rel)
	if info.Version != "2.4.2" {
		t.Fatalf("version = %q", info.Version)
	}
	if info.AssetName != "passguard-setup-2.4.2-amd64.exe" || info.AssetSize != 42 {
		t.Fatalf("installer = %q %d", info.AssetName, info.AssetSize)
	}
	if info.ChecksumsURL != "https://example.com/sums" {
		t.Fatalf("checksums = %q", info.ChecksumsURL)
	}
}

func TestFetchLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v2.5.0",
			"name": "PassGuard 2.5.0",
			"html_url": "https://example.com/release",
			"assets": [
				{"name": "passguard-setup-2.5.0-amd64.exe", "size": 123, "browser_download_url": "https://example.com/setup"},
				{"name": "SHA256SUMS.txt", "size": 10, "browser_download_url": "https://example.com/sums"}
			]
		}`))
	}))
	defer srv.Close()

	info, err := fetchLatestRelease(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "2.5.0" || info.AssetSize != 123 || info.ChecksumsURL == "" {
		t.Fatalf("info = %#v", info)
	}
}

func TestDownloadReleaseAssetVerifiesChecksum(t *testing.T) {
	payload := []byte("a small stand-in for the installer")
	sum := sha256.Sum256(payload)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/setup.exe":
			_, _ = w.Write(payload)
		case "/sums":
			_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  passguard-setup-9.9.9-amd64.exe\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	info := updateInfo{
		AssetName:    "passguard-setup-9.9.9-amd64.exe",
		AssetURL:     srv.URL + "/setup.exe",
		AssetSize:    int64(len(payload)),
		ChecksumsURL: srv.URL + "/sums",
	}
	path, got, verified, err := downloadReleaseAsset(info, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	if !verified || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("verified = %v, hash = %q", verified, got)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(payload) {
		t.Fatalf("downloaded data = %q, %v", data, err)
	}

	// A checksum that names another file must fail.
	info.AssetName = "passguard-setup-9.9.9-other.exe"
	if _, _, _, err := downloadReleaseAsset(info, nil); err == nil {
		t.Fatal("download with no matching checksum succeeded")
	}
}
