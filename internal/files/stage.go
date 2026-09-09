package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OpKind classifies a staged filesystem operation.
type OpKind uint8

const (
	OpMkdir OpKind = iota
	OpDelete
	OpRename  // Arg = new base name (same directory)
	OpCopy    // Arg = full destination path (deduped at stage time)
	OpMove    // Arg = full destination path (deduped at stage time)
	OpChmod   // Mode = new permission bits
	OpChown   // UID/GID = new owner/group, -1 = unchanged; Arg = display string
	OpCreate  // Path = full new empty-file path
	OpRestore // Path = trashed path; Arg = recorded original path
)

func (k OpKind) String() string {
	switch k {
	case OpMkdir:
		return "mkdir"
	case OpDelete:
		return "delete"
	case OpRename:
		return "rename"
	case OpCopy:
		return "copy"
	case OpMove:
		return "move"
	case OpChmod:
		return "chmod"
	case OpChown:
		return "chown"
	case OpCreate:
		return "create"
	case OpRestore:
		return "restore"
	}
	return "?"
}

// Op is one staged operation, applied later on save.
type Op struct {
	Kind OpKind
	Path string // subject path (for OpMkdir/OpCreate: the full new path)
	Arg  string // kind-dependent argument
	Mode os.FileMode
	UID  int // OpChown: new uid, -1 = unchanged
	GID  int // OpChown: new gid, -1 = unchanged
}

// Label renders the operation as a short human phrase.
func (o Op) Label() string {
	switch o.Kind {
	case OpMkdir:
		return "mkdir " + filepath.Base(o.Path)
	case OpCreate:
		return "create " + filepath.Base(o.Path)
	case OpRestore:
		return "restore " + filepath.Base(o.Path) + " -> " + filepath.Dir(o.Arg)
	case OpDelete:
		return "delete " + filepath.Base(o.Path)
	case OpRename:
		return "rename " + filepath.Base(o.Path) + " -> " + filepath.Base(o.Arg)
	case OpCopy:
		return "copy " + filepath.Base(o.Path) + " -> " + filepath.Base(o.Arg)
	case OpMove:
		return "move " + filepath.Base(o.Path) + " -> " + filepath.Base(o.Arg)
	case OpChmod:
		return "chmod " + o.Mode.String() + " " + filepath.Base(o.Path)
	case OpChown:
		return "chown " + o.Arg + " " + filepath.Base(o.Path)
	}
	return "unknown"
}

// Stager accumulates operations and applies them only on command —
// the oil.nvim model: edits are proposals until saved.
//
// Validation happens at Stage time (existence, clobber, self-nesting)
// so obvious mistakes surface before saving; the primitives re-check
// at apply time, so races fail safely with stop-on-error semantics.
type Stager struct {
	ops []Op
}

// NewStager creates an empty operation queue.
func NewStager() *Stager { return &Stager{} }

// Stage validates and appends one operation.
func (s *Stager) Stage(op Op) error {
	switch op.Kind {
	case OpMkdir:
		if _, err := os.Lstat(op.Path); err == nil {
			return fmt.Errorf("%s already exists", filepath.Base(op.Path))
		}
	case OpDelete, OpChmod, OpChown:
		if _, err := os.Lstat(op.Path); err != nil {
			return fmt.Errorf("%s vanished", filepath.Base(op.Path))
		}
	case OpRestore:
		if !InTrash(op.Path) {
			return fmt.Errorf("%s is not in the trash", filepath.Base(op.Path))
		}
		if _, err := os.Lstat(op.Path); err != nil {
			return fmt.Errorf("%s vanished", filepath.Base(op.Path))
		}
		if op.Arg == "" {
			return fmt.Errorf("no trash record for %s", filepath.Base(op.Path))
		}
		if _, err := os.Lstat(op.Arg); err == nil {
			return fmt.Errorf("%s already exists at the original location",
				filepath.Base(op.Arg))
		}
	case OpRename:
		if op.Arg == "" || op.Arg == "." || op.Arg == ".." ||
			filepath.Base(op.Arg) != op.Arg {
			return fmt.Errorf("bad name %q", op.Arg)
		}
		dst := filepath.Join(filepath.Dir(op.Path), op.Arg)
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("%s already exists", op.Arg)
		}
	case OpCreate:
		base := filepath.Base(op.Path)
		if base == "" || base == "." || base == ".." || base == "/" {
			return fmt.Errorf("bad name %q", base)
		}
		if _, err := os.Lstat(op.Path); err == nil {
			return fmt.Errorf("%s already exists", base)
		}
		if _, err := os.Lstat(filepath.Dir(op.Path)); err != nil {
			return fmt.Errorf("missing directory %s", filepath.Base(filepath.Dir(op.Path)))
		}
	case OpCopy, OpMove:
		if _, err := os.Lstat(op.Path); err != nil {
			return fmt.Errorf("%s vanished", filepath.Base(op.Path))
		}
		dst := filepath.Clean(op.Arg)
		if filepath.Clean(dst) == filepath.Clean(op.Path) {
			return fmt.Errorf("cannot %s %s onto itself",
				op.Kind, filepath.Base(op.Path))
		}
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("%s already exists in %s",
				filepath.Base(dst), filepath.Base(filepath.Dir(dst)))
		}
		if op.Kind == OpCopy {
			if err := nestingErr(op.Path, filepath.Dir(dst)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown op kind")
	}
	s.ops = append(s.ops, op)
	return nil
}

// Ops returns a copy of the queued operations, in save order.
func (s *Stager) Ops() []Op {
	out := make([]Op, len(s.ops))
	copy(out, s.ops)
	return out
}

// UniqueDst resolves the paste destination for src inside dir: the
// plain base name when free, else name.2, name.3, ... skipping names
// that exist on disk or are already claimed by queued copy/move ops
// (the save applies in order, so both would clobber). Pasting into
// the same directory is the ordinary case, not an error.
func (s *Stager) UniqueDst(src, dir string) string {
	claimed := map[string]bool{}
	for _, op := range s.ops {
		if op.Kind == OpCopy || op.Kind == OpMove {
			claimed[filepath.Clean(op.Arg)] = true
		}
	}
	base := filepath.Base(src)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	p := filepath.Join(dir, base)
	for i := 2; ; i++ {
		if !claimed[filepath.Clean(p)] {
			if _, err := os.Lstat(p); err != nil {
				return p
			}
		}
		p = filepath.Join(dir, fmt.Sprintf("%s.%d%s", stem, i, ext))
	}
}

// Len reports how many operations are queued.
func (s *Stager) Len() int { return len(s.ops) }

// Undo removes and returns the most recently queued operation.
func (s *Stager) Undo() (Op, bool) {
	if len(s.ops) == 0 {
		return Op{}, false
	}
	op := s.ops[len(s.ops)-1]
	s.ops = s.ops[:len(s.ops)-1]
	return op, true
}

// DropFirst removes the first n operations (the ones already applied
// when a save stops on an error mid-queue).
func (s *Stager) DropFirst(n int) {
	if n >= len(s.ops) {
		s.ops = nil
		return
	}
	s.ops = s.ops[n:]
}

// Clear drops every queued operation.
func (s *Stager) Clear() { s.ops = nil }

// ApplyOp executes a single staged operation immediately. It is the
// only execution path; the UI chains calls so failures can stop the
// run while unapplied operations stay queued.
func ApplyOp(op Op) error {
	switch op.Kind {
	case OpMkdir:
		return Mkdir(filepath.Dir(op.Path), filepath.Base(op.Path))
	case OpCreate:
		return CreateFile(op.Path)
	case OpRestore:
		return Restore(op.Path)
	case OpDelete:
		if InTrash(op.Path) {
			return os.RemoveAll(op.Path) // already trashed: purge for good
		}
		return Trash(op.Path)
	case OpRename:
		return Rename(op.Path, op.Arg)
	case OpCopy:
		return Copy(op.Path, op.Arg)
	case OpMove:
		return Move(op.Path, op.Arg)
	case OpChmod:
		return Chmod(op.Path, op.Mode)
	case OpChown:
		return Chown(op.Path, op.UID, op.GID)
	}
	return fmt.Errorf("unknown op kind")
}
