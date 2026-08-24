package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// OpKind classifies a staged filesystem operation.
type OpKind uint8

const (
	OpMkdir OpKind = iota
	OpDelete
	OpRename // Arg = new base name (same directory)
	OpCopy   // Arg = destination directory
	OpMove   // Arg = destination directory
	OpChmod  // Mode = new permission bits
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
	}
	return "?"
}

// Op is one staged operation, applied later on save.
type Op struct {
	Kind OpKind
	Path string // subject path (for OpMkdir: the full new path)
	Arg  string // kind-dependent argument
	Mode os.FileMode
}

// Label renders the operation as a short human phrase.
func (o Op) Label() string {
	switch o.Kind {
	case OpMkdir:
		return "create " + filepath.Base(o.Path)
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
	case OpDelete, OpChmod:
		if _, err := os.Lstat(op.Path); err != nil {
			return fmt.Errorf("%s vanished", filepath.Base(op.Path))
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
	case OpCopy, OpMove:
		if _, err := os.Lstat(op.Path); err != nil {
			return fmt.Errorf("%s vanished", filepath.Base(op.Path))
		}
		dst := filepath.Join(op.Arg, filepath.Base(op.Path))
		if filepath.Clean(dst) == filepath.Clean(op.Path) {
			return fmt.Errorf("cannot %s %s onto itself",
				op.Kind, filepath.Base(op.Path))
		}
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("%s already exists in %s",
				filepath.Base(op.Path), filepath.Base(op.Arg))
		}
		if op.Kind == OpCopy {
			if err := nestingErr(op.Path, op.Arg); err != nil {
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
	case OpDelete:
		return Delete(op.Path)
	case OpRename:
		return Rename(op.Path, op.Arg)
	case OpCopy:
		return Copy(op.Path, op.Arg)
	case OpMove:
		return Move(op.Path, op.Arg)
	case OpChmod:
		return Chmod(op.Path, op.Mode)
	}
	return fmt.Errorf("unknown op kind")
}
