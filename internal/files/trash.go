package files

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// errDifferentFS reports a path that lives on another filesystem: it
// cannot reach the home trash, and this tool never deletes data it
// cannot trash.
var errDifferentFS = errors.New("different filesystem — not trashed")

// TrashAvailable reports whether gio can perform the delete; the home
// trash remains a fallback even without it.
func TrashAvailable() bool {
	_, err := exec.LookPath("gio")
	return err == nil
}

// osRename is indirected so tests can simulate cross-device failures.
var osRename = os.Rename

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
// record. When neither channel can take the file (cross-device rename,
// unwritable trash), Trash refuses with an error and leaves the source
// untouched: nothing here ever deletes permanently.
func Trash(path string) error {
	if _, err := exec.LookPath("gio"); err == nil {
		if err := exec.Command("gio", "trash", "--", path).Run(); err == nil {
			return nil
		}
		// gio exists but failed; try the home trash below.
	}
	filesDir, infoDir, derr := homeTrashDirs()
	if derr != nil {
		return fmt.Errorf("cannot create trash: %w", derr)
	}
	target := uniqueTarget(filesDir, filepath.Base(path))
	if err := osRename(path, target); err != nil {
		return errDifferentFS
	}
	info := "[Trash Info]\nPath=" + urlEscape(path) +
		"\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	_ = os.WriteFile(filepath.Join(infoDir, filepath.Base(target)+".trashinfo"),
		[]byte(info), 0o600)
	return nil
}
