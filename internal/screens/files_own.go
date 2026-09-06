package screens

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/files"
	"lcc2/internal/ui"
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
	f.chownTarget = e.Path
	return f, cmd
}

// chownSubmit resolves the entered names and stages the op.
func (f Files) chownSubmit() (ui.Screen, tea.Cmd) {
	vals := f.chownForm.Values()
	target := f.chownTarget
	f.chownForm = nil
	owner, group := strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1])
	if owner == "" && group == "" {
		return f, ui.InfoToast("nothing to change")
	}
	uid, gid, err := files.ResolveOwner(owner, group)
	if err != nil {
		return f, ui.ErrToast("chown: " + err.Error())
	}
	if err := f.stager.Stage(files.Op{
		Kind: files.OpChown, Path: target, UID: uid, GID: gid,
		Arg: strings.Trim(owner+":"+group, ":"),
	}); err != nil {
		return f, ui.ErrToast(err.Error())
	}
	f.syncTable()
	return f, ui.InfoToast("staged chown " + strings.Trim(owner+":"+group, ":") +
		" " + filepath.Base(target))
}
