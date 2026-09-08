// Package files implements the file-manager data layer: directory
// listings, metadata and safe primitives for basic file operations.
package files

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Entry is one row in the file manager listing.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	Mode    os.FileMode
	UID     uint32
	GID     uint32
	ModTime time.Time
	Link    string // symlink target, when the entry is a link
}

var (
	uidNames = map[uint32]string{}
	gidNames = map[uint32]string{}
)

// UserName resolves a uid to a name, falling back to the number.
func UserName(uid uint32) string {
	if n, ok := uidNames[uid]; ok {
		return n
	}
	if u, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
		uidNames[uid] = u.Username
		return u.Username
	}
	n := "#" + strconv.Itoa(int(uid))
	uidNames[uid] = n
	return n
}

// GroupName resolves a gid to a name, falling back to the number.
func GroupName(gid uint32) string {
	if n, ok := gidNames[gid]; ok {
		return n
	}
	if g, err := user.LookupGroupId(strconv.Itoa(int(gid))); err == nil {
		gidNames[gid] = g.Name
		return g.Name
	}
	n := "#" + strconv.Itoa(int(gid))
	gidNames[gid] = n
	return n
}

// List reads a directory sorted dirs-first, names case-insensitive.
func List(dir string, showHidden bool) ([]Entry, error) {
	dirents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(dirents))
	for _, de := range dirents {
		if !showHidden && len(de.Name()) > 0 && de.Name()[0] == '.' {
			continue
		}
		path := filepath.Join(dir, de.Name())
		e := Entry{Name: de.Name(), Path: path, IsDir: de.IsDir()}
		if de.Type()&os.ModeSymlink != 0 {
			if tgt, err := os.Readlink(path); err == nil {
				e.Link = tgt
			}
		}
		if info, err := de.Info(); err == nil {
			e.Size = info.Size()
			e.Mode = info.Mode() // keep type bits: d/l prefixes in the UI
			e.ModTime = info.ModTime()
			if st, ok := info.Sys().(*syscall.Stat_t); ok {
				e.UID = st.Uid
				e.GID = st.Gid
			}
		} else if de.Type()&os.ModeSymlink != 0 {
			if tgt, err := os.Stat(path); err == nil { // follow for metadata
				e.IsDir = tgt.IsDir()
				e.Size = tgt.Size()
				e.Mode = tgt.Mode()
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// Home returns the user's home directory (falls back to "/").
func Home() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return "/"
}

// Delete permanently removes a file or directory tree. Prefer Trash
// so deletes are recoverable.
func Delete(path string) error {
	return os.RemoveAll(path)
}

// Rename renames within the same directory; refuses to clobber.
func Rename(oldPath, newName string) error {
	dst := filepath.Join(filepath.Dir(oldPath), filepath.Base(newName))
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dst))
	}
	return os.Rename(oldPath, dst)
}

// Mkdir creates a new directory inside parent.
func Mkdir(parent, name string) error {
	return os.Mkdir(filepath.Join(parent, name), 0755)
}

// nestingErr reports when dstDir lies inside src (or equals it) —
// copying a directory into its own subtree would recurse unbounded.
func nestingErr(src, dstDir string) error {
	cleanSrc := filepath.Clean(src)
	rel, err := filepath.Rel(cleanSrc, filepath.Clean(dstDir))
	if err != nil {
		return nil // unrelated trees; the primitives re-check anyway
	}
	if rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("cannot copy %s into itself", filepath.Base(cleanSrc))
	}
	return nil
}

// Copy copies src to the full destination path dst. Refuses
// same-path copies, self-nesting copies and overwrites.
func Copy(src, dst string) error {
	dst = filepath.Clean(dst)
	if dst == filepath.Clean(src) {
		return fmt.Errorf("cannot copy %s onto itself", filepath.Base(src))
	}
	if err := nestingErr(src, filepath.Dir(dst)); err != nil {
		return err
	}
	return copyTree(src, dst)
}

// copyTree copies src (file or directory) to dst, walking
// directories depth-first and preserving child names.
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst, info.Mode())
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dst))
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()),
			filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// CreateFile makes a new empty file; it refuses to clobber.
func CreateFile(path string) error {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return fh.Close()
}

// Move moves src to the full destination path dst; refuses
// overwrites. Falls back to copy+delete across filesystems only after
// all guards pass.
func Move(src, dst string) error {
	dst = filepath.Clean(dst)
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dst))
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := Copy(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyFile(src, dst string, mode os.FileMode) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dst))
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Chmod applies a new permission mode to path.
func Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

// Chown applies a new owner and/or group to path; -1 leaves that id
// unchanged. Non-root callers may only chown their own files to
// groups they belong to; the OS enforces this.
func Chown(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}

// WalkTree lists every path under root, including root itself.
// Symlinks are skipped: applying chmod through them would silently
// follow to their targets.
func WalkTree(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != root && d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		out = append(out, p)
		return nil
	})
	return out, err
}

// ResolveOwner maps owner/group names to ids through the system user
// database; empty means unchanged (-1).
func ResolveOwner(owner, group string) (uid, gid int, err error) {
	uid, gid = -1, -1
	if owner != "" {
		u, err := user.Lookup(owner)
		if err != nil {
			return -1, -1, fmt.Errorf("unknown user %q", owner)
		}
		if uid, err = strconv.Atoi(u.Uid); err != nil {
			return -1, -1, fmt.Errorf("bad uid for %q", owner)
		}
	}
	if group != "" {
		g, err := user.LookupGroup(group)
		if err != nil {
			return -1, -1, fmt.Errorf("unknown group %q", group)
		}
		if gid, err = strconv.Atoi(g.Gid); err != nil {
			return -1, -1, fmt.Errorf("bad gid for %q", group)
		}
	}
	return uid, gid, nil
}
