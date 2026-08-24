// Package files implements the file-manager data layer: directory
// listings, metadata and safe primitives for basic file operations.
package files

import (
	"fmt"
	"io"
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
		if info, err := de.Info(); err == nil {
			e.Size = info.Size()
			e.Mode = info.Mode().Perm()
			e.ModTime = info.ModTime()
			if st, ok := info.Sys().(*syscall.Stat_t); ok {
				e.UID = st.Uid
				e.GID = st.Gid
			}
		} else if de.Type()&os.ModeSymlink != 0 {
			if tgt, err := os.Stat(path); err == nil { // follow for metadata
				e.IsDir = tgt.IsDir()
				e.Size = tgt.Size()
				e.Mode = tgt.Mode().Perm()
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

// Delete removes a file or a directory tree after caller confirmation.
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

// nestingErr reports when dstDir lies inside src (or equals it).
func nestingErr(src, dstDir string) error {
	cleanSrc := filepath.Clean(src)
	rel, err := filepath.Rel(cleanSrc, filepath.Clean(dstDir))
	if err != nil || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil // not nested
	}
	return fmt.Errorf("cannot copy %s into itself", filepath.Base(cleanSrc))
}

// Copy recursively copies src into dstDir keeping its base name.
// Refuses same-path pastes, self-nesting copies and overwrites.
func Copy(src, dstDir string) error {
	dst := filepath.Join(dstDir, filepath.Base(src))
	if filepath.Clean(dst) == filepath.Clean(src) {
		return fmt.Errorf("cannot copy %s onto itself", filepath.Base(src))
	}
	if err := nestingErr(src, dstDir); err != nil {
		return err
	}
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
		if err := Copy(filepath.Join(src, e.Name()), dst); err != nil {
			return err
		}
	}
	return nil
}

// Move moves src into dstDir; refuses overwrites. Falls back to
// copy+delete across filesystems only after all guards pass.
func Move(src, dstDir string) error {
	dst := filepath.Join(dstDir, filepath.Base(src))
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dst))
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := Copy(src, dstDir); err != nil {
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
