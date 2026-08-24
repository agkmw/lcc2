package screens

import "lcc2/internal/ui"

// ContextSource implementations: one line of context for the status
// bar's right slot, per screen.

func (p Processes) ContextHint() string {
	if e, ok := p.tbl.Selected(); ok && e < len(p.all) {
		return contextSel(p.all[e].Name, itoa(int(p.all[e].PID)))
	}
	return ""
}

func (f Files) ContextHint() string {
	if e, ok := f.selected(); ok {
		return contextSel(e.Name, "")
	}
	return ""
}

func (s Services) ContextHint() string {
	if name := s.selectedName(); name != "" {
		for _, u := range s.units {
			if u.Name == name {
				return contextSel(u.Name, u.Active)
			}
		}
	}
	return ""
}

func (u UsersGroups) ContextHint() string {
	if u.tab == "users" {
		if idx, ok := u.uTbl.Selected(); ok && idx < len(u.users) {
			return contextSel(u.users[idx].Name, itoa(u.users[idx].UID))
		}
		return ""
	}
	if idx, ok := u.gTbl.Selected(); ok && idx < len(u.groups) {
		return contextSel(u.groups[idx].Name, itoa(u.groups[idx].GID))
	}
	return ""
}

func (d Disks) ContextHint() string {
	if d.mode == "scan" && d.path != "" {
		return contextSel(filepathBase(d.path), "")
	}
	return ""
}

var _ = ui.ContextSource(nil)
