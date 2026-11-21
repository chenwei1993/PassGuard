package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

const defaultUpdateAPI = "https://api.github.com/repos/chenwei1993/PassGuard/releases/latest"

var (
	updateHTTP   = &http.Client{Timeout: 20 * time.Second}
	downloadHTTP = &http.Client{Timeout: 10 * time.Minute}
	// installerAsset matches the NSIS installer of a release, whose name
	// carries the version, e.g. passguard-setup-2.4.2-amd64.exe.
	installerAsset = regexp.MustCompile(`(?i)^passguard-setup-.*-amd64\.exe$`)
)

// updateState is what the About page shows about GitHub Releases. The
// fields below the check are filled by downloadUpdate.
type updateState struct {
	checking    bool
	checked     bool
	info        *updateInfo
	err         string
	downloading bool
	progress    int64
	total       int64
	readyPath   string
	readyHash   string
	verified    bool
	installErr  string
}

// updateInfo is the latest release, with the assets an update needs.
type updateInfo struct {
	Version      string
	Name         string
	URL          string
	PublishedAt  time.Time
	AssetName    string
	AssetURL     string
	AssetSize    int64
	ChecksumsURL string
}

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// updateAPIURL is the latest-release endpoint, which PASSGUARD_UPDATE_API
// can point at a local server to try the update flow.
func updateAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("PASSGUARD_UPDATE_API")); v != "" {
		return v
	}
	return defaultUpdateAPI
}

// checkForUpdates asks GitHub for the latest release. A manual check shows
// its error; an automatic one stays quiet, as an offline launch should not.
func (a *App) checkForUpdates(manual bool) {
	if a.upd.checking || a.upd.downloading {
		return
	}
	a.upd.checking = true
	a.upd.err = ""
	win := a.win
	if win == nil {
		a.upd.checking = false
		return
	}
	go func() {
		info, err := fetchLatestRelease(updateAPIURL())
		win.Update(func() {
			a.upd.checking = false
			a.upd.checked = true
			if err != nil {
				if manual {
					a.upd.err = err.Error()
				}
				return
			}
			a.upd.err = ""
			if info != nil && isNewerVersion(info.Version, appVersion) {
				a.upd.info = info
				a.toast(a.tf("update.toastAvailable", map[string]any{"version": info.Version}))
			} else {
				a.upd.info = nil
			}
		})
	}()
}

// fetchLatestRelease reads the release at url. It returns the release even
// when it has no installer, so the page can still link to it.
func fetchLatestRelease(url string) (*updateInfo, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "PassGuard/"+appVersion)
	resp, err := updateHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, errors.New("no release found")
	default:
		return nil, fmt.Errorf("GitHub returned %s", resp.Status)
	}
	var rel githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	return updateFromRelease(rel), nil
}

// updateFromRelease picks the installer and checksum assets of a release.
func updateFromRelease(rel githubRelease) *updateInfo {
	info := &updateInfo{
		Version:     normalizeVersion(rel.TagName),
		Name:        rel.Name,
		URL:         rel.HTMLURL,
		PublishedAt: rel.PublishedAt,
	}
	if info.Version == "" {
		info.Version = normalizeVersion(rel.Name)
	}
	for _, asset := range rel.Assets {
		switch {
		case installerAsset.MatchString(asset.Name) && info.AssetURL == "":
			info.AssetName = asset.Name
			info.AssetURL = asset.BrowserDownloadURL
			info.AssetSize = asset.Size
		case (strings.EqualFold(asset.Name, "SHA256SUMS.txt") || strings.EqualFold(asset.Name, "checksums.txt")) && info.ChecksumsURL == "":
			info.ChecksumsURL = asset.BrowserDownloadURL
		}
	}
	return info
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	return v
}

// isNewerVersion reports whether latest is a later X.Y.Z than current.
func isNewerVersion(latest, current string) bool {
	a, b := versionNumbers(latest), versionNumbers(current)
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func versionNumbers(v string) [3]int {
	var out [3]int
	for i, part := range strings.SplitN(normalizeVersion(v), ".", 3) {
		n, _ := strconv.Atoi(strings.TrimFunc(part, func(r rune) bool {
			return r < '0' || r > '9'
		}))
		out[i] = n
	}
	return out
}

// downloadUpdate saves the installer next to the system's temporary files,
// verifies its checksum when the release has one, and leaves it ready to
// install.
func (a *App) downloadUpdate() {
	info := a.upd.info
	if info == nil || info.AssetURL == "" || a.upd.downloading {
		return
	}
	a.upd.downloading = true
	a.upd.err = ""
	a.upd.installErr = ""
	a.upd.progress = 0
	a.upd.total = info.AssetSize
	a.upd.readyPath = ""
	a.upd.readyHash = ""
	a.upd.verified = false
	win := a.win
	if win == nil {
		a.upd.downloading = false
		return
	}
	current := *info
	go func() {
		path, sum, verified, err := downloadReleaseAsset(current, func(n int64) {
			win.Update(func() { a.upd.progress = n })
		})
		win.Update(func() {
			a.upd.downloading = false
			if err != nil {
				a.upd.err = err.Error()
				return
			}
			a.upd.progress = a.upd.total
			a.upd.readyPath = path
			a.upd.readyHash = sum
			a.upd.verified = verified
		})
	}()
}

func downloadReleaseAsset(info updateInfo, progress func(int64)) (path, sum string, verified bool, err error) {
	req, err := http.NewRequest(http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return "", "", false, err
	}
	req.Header.Set("User-Agent", "PassGuard/"+appVersion)
	resp, err := downloadHTTP.Do(req)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", false, fmt.Errorf("download returned %s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "PassGuard-update-*.exe")
	if err != nil {
		return "", "", false, err
	}
	remove := func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}
	total := resp.ContentLength
	if total <= 0 {
		total = info.AssetSize
	}
	hasher := sha256.New()
	writer := &progressWriter{
		w:     io.MultiWriter(tmp, hasher),
		total: total,
		on:    progress,
	}
	if _, err := io.Copy(writer, resp.Body); err != nil {
		remove()
		return "", "", false, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", "", false, err
	}
	if info.AssetSize > 0 && writer.n != info.AssetSize {
		os.Remove(tmp.Name())
		return "", "", false, fmt.Errorf("downloaded %d of %d bytes", writer.n, info.AssetSize)
	}
	sum = hex.EncodeToString(hasher.Sum(nil))
	if info.ChecksumsURL != "" {
		want, err := fetchChecksum(info.ChecksumsURL, info.AssetName)
		if err != nil {
			os.Remove(tmp.Name())
			return "", "", false, err
		}
		if !strings.EqualFold(want, sum) {
			os.Remove(tmp.Name())
			return "", "", false, errors.New("the downloaded file failed its SHA-256 check")
		}
		verified = true
	}
	return tmp.Name(), sum, verified, nil
}

// progressWriter reports how much it has written, at most a few times a
// second, so the UI thread is not flooded.
type progressWriter struct {
	w     io.Writer
	n     int64
	total int64
	last  time.Time
	on    func(int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.n += int64(n)
	if p.on != nil && time.Since(p.last) > 120*time.Millisecond {
		p.last = time.Now()
		p.on(p.n)
	}
	return n, err
}

func fetchChecksum(url, name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "PassGuard/"+appVersion)
	resp, err := updateHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums returned %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(strings.TrimPrefix(fields[1], "*"), name) {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("no checksum for %s", name)
}

// installUpdate opens the downloaded installer and quits so that it can
// replace this executable. The installer updates in place and offers to
// launch the new version when it is done.
func (a *App) installUpdate() {
	if a.upd.readyPath == "" {
		return
	}
	if _, err := os.Stat(a.upd.readyPath); err != nil {
		a.upd.installErr = err.Error()
		return
	}
	cmd := exec.Command(a.upd.readyPath)
	if err := cmd.Start(); err != nil {
		a.upd.installErr = err.Error()
		return
	}
	mygo.App.Quit()
}

func (a *App) openReleasePage() {
	if a.upd.info != nil && a.upd.info.URL != "" {
		_ = mygo.Shell.OpenExternal(a.upd.info.URL)
	}
}
