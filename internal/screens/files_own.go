package screens

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/files"
	"lcc/internal/ui"
)

// osGeteuid is the test seam over os.Geteuid; screen tests run
// unprivileged, so root-gated paths stub it.
var osGeteuid = os.Geteuid

// Ownership editing: O opens a two-field form (owner, group; blank =
// unchanged) and stages one OpChown on the cursor entry. Resolution
// to ids happens at submit time through the system user database.

// openChown gates on root — chown to arbitrary ids requires it — then
// floats the form over the listing.
func (f Files) openChown() (ui.Screen, tea.Cmd) {
	if f.chownForm != nil {
		return f, nil
	}
	if osGeteuid() != 0 {
		return f, ui.ErrToast("needs root - restart the app with sudo")
	}
	e, ok := f.selected()
	if !ok {
		return f, nil
	}
	fch, cmd := ui.NewForm("chown", "files", []ui.Field{
		{Label: "owner", Placeholder: files.UserName(e.UID)},
		{Label: "group", Placeholder: files.GroupName(e.GID)},
	})
	fch.SetWidth(clampInt(f.w-8, 44, 60))
	f.chownForm = &fch
	return f, cmd
}

// chownSubmit resolves the entered names and stages one op per
// target: the marked set, or the entry the form opened on.
func (f Files) chownSubmit() (ui.Screen, tea.Cmd) {
	vals := f.chownForm.Values()
	f.chownForm = nil
	owner, group := strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1])
	if owner == "" && group == "" {
		return f, ui.InfoToast("nothing to change")
	}
	uid, gid, err := files.ResolveOwner(owner, group)
	if err != nil {
		return f, ui.ErrToast("chown: " + err.Error())
	}
	arg := strings.Trim(owner+":"+group, ":")
	ts := f.targets()
	ops := make([]files.Op, 0, len(ts))
	for _, e := range ts {
		ops = append(ops, files.Op{
			Kind: files.OpChown, Path: e.Path, UID: uid, GID: gid, Arg: arg,
		})
	}
	cmd := f.stageOps(ops, fmt.Sprintf("staged chown %s - %d paths", arg, len(ops)))
	return f, cmd
}
