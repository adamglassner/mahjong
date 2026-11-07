//go:build !js

// Updates for the standalone copy: checks this game's GitHub releases and
// installs a newer one. Copies installed by Ironite Launcher are updated by
// the launcher instead. (c) 2026 Ironite
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const updateRepo = "adamglassner/mahjong"

type UpdState struct {
	State    string // idle, checking, none, available, downloading, ready, error
	Latest   string
	Notes    string
	Err      string
	Progress float64
	Managed  bool
	Checked  time.Time
}

func (s UpdState) status() string {
	switch s.State {
	case "checking":
		return "Checking GitHub…"
	case "none":
		return "You have the latest version (" + Version + ")."
	case "available":
		return "Version " + s.Latest + " is available."
	case "downloading":
		return fmt.Sprintf("Downloading %d%%…", int(s.Progress*100))
	case "error":
		return s.Err
	}
	return "Version " + Version
}

type Updater struct {
	mu     sync.Mutex
	st     UpdState
	asset  string
	digest string
	size   int64
	// Quit is called once the update is ready to be installed; the game closes.
	Quit func()
}

func newUpdater() *Updater {
	u := &Updater{}
	u.st.State = "idle"
	if exe, err := os.Executable(); err == nil {
		p := strings.ToLower(filepath.ToSlash(exe))
		u.st.Managed = strings.Contains(p, "/ironite/launcher/games/")
	}
	return u
}

func (u *Updater) snapshot() UpdState {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.st
}

func (u *Updater) set(fn func(s *UpdState)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fn(&u.st)
}

// newer reports whether version a is newer than b ("1.2.0" > "1.1.9").
func newer(a, b string) bool {
	pa, pb := splitVer(a), splitVer(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func splitVer(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var out [3]int
	for i, p := range strings.SplitN(v, ".", 3) {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

type ghRelease struct {
	Tag    string `json:"tag_name"`
	Body   string `json:"body"`
	Draft  bool   `json:"draft"`
	Pre    bool   `json:"prerelease"`
	Assets []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"assets"`
}

// Check asks GitHub for the latest release.
func (u *Updater) Check(manual bool) {
	if u.snapshot().Managed {
		return
	}
	u.set(func(s *UpdState) { s.State, s.Err = "checking", "" })
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+updateRepo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Ironite-Mahjong/"+Version)
	resp, err := httpClient.Do(req)
	if err != nil {
		u.set(func(s *UpdState) { s.State, s.Err = "error", "Couldn't reach GitHub. Check your internet connection." })
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		u.set(func(s *UpdState) { s.State = "none"; s.Checked = time.Now() })
		return
	}
	if resp.StatusCode != 200 {
		u.set(func(s *UpdState) { s.State, s.Err = "error", "GitHub said "+resp.Status+". Try again later." })
		return
	}
	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		u.set(func(s *UpdState) { s.State, s.Err = "error", "GitHub's answer couldn't be read." })
		return
	}
	latest := strings.TrimPrefix(rel.Tag, "v")
	asset, digest, size := "", "", int64(0)
	for _, as := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(as.Name), "-windows.zip") {
			asset, digest, size = as.URL, as.Digest, as.Size
		}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.st.Checked = time.Now()
	if rel.Draft || rel.Pre || !newer(latest, Version) || asset == "" {
		u.st.State = "none"
		return
	}
	u.asset, u.digest, u.size = asset, digest, size
	u.st.State, u.st.Latest, u.st.Notes = "available", latest, cleanNotes(rel.Body)
}

// cleanNotes turns the release's markdown into plain text for the dialog.
func cleanNotes(md string) string {
	var out []string
	for _, ln := range strings.Split(strings.ReplaceAll(md, "\r", ""), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "---" || strings.HasPrefix(ln, "Install and update") {
			break
		}
		ln = strings.TrimLeft(ln, "#")
		ln = strings.ReplaceAll(ln, "**", "")
		if strings.HasPrefix(ln, "- ") {
			ln = "•  " + ln[2:]
		}
		if ln = strings.TrimSpace(ln); ln != "" {
			out = append(out, ln)
		}
	}
	return strings.Join(out, "\n")
}

type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	u     *Updater
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.total > 0 {
		f := float64(p.n) / float64(p.total)
		p.u.set(func(s *UpdState) { s.Progress = f })
	}
	return n, err
}

// download fetches the release zip into dir and unpacks it; it returns the
// folder that holds the new Mahjong.exe.
func (u *Updater) download(dir string) (string, error) {
	u.mu.Lock()
	url, digest, size := u.asset, u.digest, u.size
	u.mu.Unlock()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Ironite-Mahjong/"+Version)
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return "", errors.New("the download failed. Check your internet connection")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("the download failed (" + resp.Status + ")")
	}
	if resp.ContentLength > 0 {
		size = resp.ContentLength
	}
	zp := filepath.Join(dir, "update.zip")
	f, err := os.Create(zp)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), &progressReader{r: resp.Body, total: size, u: u})
	f.Close()
	if err != nil {
		return "", errors.New("the download was interrupted")
	}
	if strings.HasPrefix(digest, "sha256:") && hex.EncodeToString(h.Sum(nil)) != strings.TrimPrefix(digest, "sha256:") {
		return "", errors.New("the download didn't match GitHub's checksum, so it wasn't installed")
	}
	zr, err := zip.OpenReader(zp)
	if err != nil {
		return "", errors.New("the download isn't a valid zip")
	}
	defer zr.Close()
	stage := filepath.Join(dir, "stage")
	exeDir := ""
	for _, zf := range zr.File {
		name := filepath.FromSlash(zf.Name)
		dst := filepath.Join(stage, name)
		if !strings.HasPrefix(dst, stage+string(os.PathSeparator)) {
			continue // never write outside the stage folder
		}
		if zf.FileInfo().IsDir() {
			os.MkdirAll(dst, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(dst), 0o755)
		rc, err := zf.Open()
		if err != nil {
			return "", err
		}
		out, err := os.Create(dst)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return "", err
		}
		if strings.EqualFold(filepath.Base(dst), "Mahjong.exe") {
			exeDir = filepath.Dir(dst)
		}
	}
	if exeDir == "" {
		return "", errors.New("the update didn't contain Mahjong.exe")
	}
	return exeDir, nil
}

// Apply downloads the update and hands over to a small script that swaps
// the files once the game has closed, then starts the new version.
func (u *Updater) Apply() {
	u.set(func(s *UpdState) { s.State, s.Progress, s.Err = "downloading", 0, "" })
	fail := func(err error) {
		u.set(func(s *UpdState) { s.State, s.Err = "error", "The update couldn't be installed: "+err.Error()+"." })
	}
	exe, err := os.Executable()
	if err != nil {
		fail(err)
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	dir, err := os.MkdirTemp("", "MahjongUpdate")
	if err != nil {
		fail(err)
		return
	}
	src, err := u.download(dir)
	if err != nil {
		fail(err)
		return
	}
	if err := installUpdate(src, filepath.Dir(exe), dir); err != nil {
		fail(err)
		return
	}
	u.set(func(s *UpdState) { s.State = "ready" })
	time.Sleep(400 * time.Millisecond)
	if u.Quit != nil {
		u.Quit()
	}
}

func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".mahjong-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// cleanupOldUpdate removes download folders left by earlier updates.
func cleanupOldUpdate() {
	ms, _ := filepath.Glob(filepath.Join(os.TempDir(), "MahjongUpdate*"))
	for _, m := range ms {
		if fi, err := os.Stat(m); err == nil && time.Since(fi.ModTime()) > time.Hour {
			os.RemoveAll(m)
		}
	}
}
