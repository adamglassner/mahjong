package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateVersions(t *testing.T) {
	if !newer("1.0.1", "1.0.0") || !newer("v2.0.0", "1.9.9") || newer("1.0.0", "1.0.0") || newer("0.9.0", "1.0.0") || !newer("1.10.0", "1.9.0") {
		t.Fatal("version compare")
	}
	n := cleanNotes("### Added\n- **New** puzzle\n\n---\nInstall and update this game with Ironite Launcher")
	if n != "Added\n•  New puzzle" {
		t.Fatalf("notes %q", n)
	}
}

func TestUpdateDownload(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"Mahjong/Mahjong.exe", "Mahjong/README.txt", "../evil.txt"} {
		w, _ := zw.Create(name)
		w.Write([]byte("x " + name))
	}
	zw.Close()
	sum := sha256.Sum256(buf.Bytes())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(buf.Bytes()) }))
	defer srv.Close()
	u := newUpdater()
	u.asset, u.digest = srv.URL, "sha256:"+hex.EncodeToString(sum[:])
	dir := t.TempDir()
	exeDir, err := u.download(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(exeDir, "Mahjong.exe")); err != nil {
		t.Fatal("exe not unpacked")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Fatal("zip escaped the stage folder")
	}
	u.digest = "sha256:00"
	if _, err := u.download(t.TempDir()); err == nil {
		t.Fatal("bad checksum accepted")
	}
}
