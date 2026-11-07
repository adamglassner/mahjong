//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// installUpdate writes a script that waits for this game to close, copies the
// new files over the old ones and starts the game again. If the install
// folder needs administrator rights (Program Files), Windows asks first.
func installUpdate(src, dst, work string) error {
	script := filepath.Join(work, "apply-update.cmd")
	exe := filepath.Join(dst, "Mahjong.exe")
	body := strings.Join([]string{
		"@echo off",
		"set PID=" + strconv.Itoa(os.Getpid()),
		":wait",
		`tasklist /FI "PID eq %PID%" 2>nul | find "%PID%" >nul && (ping -n 2 127.0.0.1 >nul & goto wait)`,
		`robocopy "` + src + `" "` + dst + `" /E /R:5 /W:1 /NFL /NDL /NJH /NJS >nul`,
		`start "" "` + exe + `"`,
		`rmdir /S /Q "` + filepath.Join(work, "stage") + `" >nul 2>&1`,
		`del "` + filepath.Join(work, "update.zip") + `" >nul 2>&1`,
	}, "\r\n") + "\r\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		return err
	}
	hidden := &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	if dirWritable(dst) {
		cmd := exec.Command("cmd.exe", "/C", script)
		cmd.SysProcAttr = hidden
		return cmd.Start()
	}
	ps := `Start-Process -FilePath 'cmd.exe' -ArgumentList '/C','"` + script + `"' -Verb RunAs -WindowStyle Hidden`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = hidden
	if err := cmd.Run(); err != nil {
		return errors.New("Windows didn't allow the update (administrator permission is needed for this folder)")
	}
	return nil
}
