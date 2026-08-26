package files

import (
	"fmt"
	"os"
	"strconv"
)

// PermBits models the classic 3x3 permission matrix
// (user/group/other × read/write/execute) plus the special bits.
type PermBits struct {
	U, G, O [3]bool
	Special int // setuid(4) | setgid(2) | sticky(1); 0 renders no digit
}

// ParsePermBits extracts the permission matrix from a file mode.
func ParsePermBits(m os.FileMode) PermBits {
	bit := func(shift uint) [3]bool {
		v := int(m>>shift) & 7
		return [3]bool{v&4 != 0, v&2 != 0, v&1 != 0}
	}
	special := 0
	if m&os.ModeSetuid != 0 {
		special |= 4
	}
	if m&os.ModeSetgid != 0 {
		special |= 2
	}
	if m&os.ModeSticky != 0 {
		special |= 1
	}
	return PermBits{U: bit(6), G: bit(3), O: bit(0), Special: special}
}

// Octal renders the mode as an octal string like "644"; when special
// bits are set it grows a leading digit ("4755"). Without this, staging
// a chmod from the editor silently cleared setuid/setgid/sticky.
func (p PermBits) Octal() string {
	d := func(v [3]bool) int {
		n := 0
		for _, x := range v {
			n = n*2 + b2i(x)
		}
		return n
	}
	head := ""
	if p.Special != 0 {
		head = strconv.Itoa(p.Special)
	}
	return head + fmt.Sprintf("%d%d%d", d(p.U), d(p.G), d(p.O))
}

// Symbolic renders the mode as "rwxr-xr--".
func (p PermBits) Symbolic() string {
	s := func(v [3]bool) string {
		out := ""
		out += yn(v[0], 'r')
		out += yn(v[1], 'w')
		out += yn(v[2], 'x')
		return out
	}
	return s(p.U) + s(p.G) + s(p.O)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func yn(b bool, c rune) string {
	if b {
		return string(c)
	}
	return "-"
}

// Toggle flips one cell of the matrix; who is 0..2 (u/g/o),
// which is 0..2 (r/w/x).
func (p *PermBits) Toggle(who, which int) {
	switch who {
	case 0:
		p.U[which] = !p.U[which]
	case 1:
		p.G[which] = !p.G[which]
	default:
		p.O[which] = !p.O[which]
	}
}
