package files

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TrashAvailable reports whether a non-permanent delete is possible:
// gio on PATH (home trash is created on demand as a second resort).
func TrashAvailable() bool {
	_, err := exec.LookPath("gio")
	return err == nil
}

// homeTrashDirs returns the files/ and info/ directories of the user's
// freedesktop trash, creating them on first use.
func homeTrashDirs() (filesDir, infoDir string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	base := filepath.Join(home, ".local", "share", "Trash")
	filesDir, infoDir = filepath.Join(base, "files"), filepath.Join(base, "info")
	for _, d := range []string{filesDir, infoDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return "", "", err
		}
	}
	return filesDir, infoDir, nil
}

// uniqueTarget finds a non-colliding name inside dir: "name",
// "name.2", "name.3", ...
func uniqueTarget(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Lstat(p); err == nil {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		for i := 2; ; i++ {
			p = filepath.Join(dir, fmt.Sprintf("%s.%d%s", base, i, ext))
			if _, err := os.Lstat(p); err != nil {
				break
			}
		}
	}
	return p
}

// urlEscape percent-encodes bytes that are illegal in a trashinfo
// Path= URL; spaces, control characters and '%'.
func urlEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= 0x20 || c == '%' {
			b.WriteString(fmt.Sprintf("%%%02X", c))
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Trash moves path into the freedesktop trash — via gio when present,
// otherwise a rename into ~/.local/share/Trash with a .trashinfo
// record. When neither works it deletes permanently. Returns true
// when the deletion was permanent so callers can warn.
func Trash(path string) (permanent bool, err error) {
	if _, err := exec.LookPath("gio"); err == nil {
		if err := exec.Command("gio", "trash", "--", path).Run(); err == nil {
			return false, nil
		}
		// gio exists but failed; try the home trash below.
	}
	filesDir, infoDir, derr := homeTrashDirs()
	if derr != nil {
		return true, os.RemoveAll(path)
	}
	target := uniqueTarget(filesDir, filepath.Base(path))
	if err := os.Rename(path, target); err != nil {
		// Cross-device or unreadable parent: last resort.
		return true, os.RemoveAll(path)
	}
	info := "[Trash Info]\nPath=" + urlEscape(path) +
		"\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	_ = os.WriteFile(filepath.Join(infoDir, filepath.Base(target)+".trashinfo"),
		[]byte(info), 0o600)
	return false, nil
}
