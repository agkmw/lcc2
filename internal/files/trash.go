package files

import (
	"errors"
	"fmt"
	"net/url"
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

// TrashDir returns the user's home trash "files" directory when it
// exists — lookup only, never creates.
func TrashDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	d := filepath.Join(home, ".local", "share", "Trash", "files")
	if _, err := os.Stat(d); err != nil {
		return "", false
	}
	return d, true
}

// InTrash reports whether path lives inside the home trash. Such
// paths are already deleted: applying a delete op to them is a
// permanent remove, not a re-trash.
func InTrash(path string) bool {
	d, ok := TrashDir()
	if !ok {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(d, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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

// TrashInfo reads a trashed entry's record: its original path and the
// deletion time. ok is false when the entry has no (readable) record.
func TrashInfo(trashed string) (origin string, deleted time.Time, ok bool) {
	info := filepath.Join(filepath.Dir(trashed), "..", "info",
		filepath.Base(trashed)+".trashinfo")
	data, err := os.ReadFile(info)
	if err != nil {
		return "", time.Time{}, false
	}
	for _, ln := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(ln, "Path="):
			if p, err := url.PathUnescape(strings.TrimPrefix(ln, "Path=")); err == nil {
				origin = p
			}
		case strings.HasPrefix(ln, "DeletionDate="):
			deleted, _ = time.Parse("2006-01-02T15:04:05",
				strings.TrimPrefix(ln, "DeletionDate="))
		}
	}
	return origin, deleted, origin != ""
}

// Restore moves a trashed entry back to its recorded original path,
// recreating any directories that vanished with it, and drops the
// .trashinfo record. Refuses to overwrite whatever now sits at the
// origin.
func Restore(trashed string) error {
	origin, _, ok := TrashInfo(trashed)
	if !ok {
		return fmt.Errorf("no trash record for %s", filepath.Base(trashed))
	}
	if _, err := os.Lstat(origin); err == nil {
		return fmt.Errorf("%s already exists at the original location", filepath.Base(origin))
	}
	if err := os.MkdirAll(filepath.Dir(origin), 0o755); err != nil {
		return err
	}
	if err := osRename(trashed, origin); err != nil {
		return err
	}
	info := filepath.Join(filepath.Dir(trashed), "..", "info",
		filepath.Base(trashed)+".trashinfo")
	_ = os.Remove(info)
	return nil
}
