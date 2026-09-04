package ui

import (
	"strings"
	"testing"
)

func newTestForm() Form {
	f, _ := NewForm("test form", "users", []Field{
		{Label: "name", Value: "pre"},
		{Label: "shell", Placeholder: "/bin/bash"},
		{Label: "password", Password: true},
	})
	return f
}

func TestFormValuesPrefill(t *testing.T) {
	f := newTestForm()
	got := f.Values()
	want := []string{"pre", "", ""}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Values()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFormEscCancels(t *testing.T) {
	f := newTestForm()
	sub, done, _ := f.Update(keyMsg("esc"))
	if done != true || sub != false {
		t.Fatalf("esc: submitted=%v done=%v, want false/true", sub, done)
	}
}

func TestFormEnterSubmits(t *testing.T) {
	f := newTestForm()
	f.Update(keyMsg("x")) // types into field 0
	sub, done, _ := f.Update(keyMsg("enter"))
	if sub != true || done != true {
		t.Fatalf("enter: submitted=%v done=%v, want true/true", sub, done)
	}
	if v := f.Values()[0]; v != "prex" {
		t.Fatalf("typed value = %q, want %q", v, "prex")
	}
}

func TestFormTabCyclesFocus(t *testing.T) {
	f := newTestForm()
	for i, step := range []string{"tab", "tab", "tab"} {
		f.Update(keyMsg(step))
		if f.cur != (i+1)%3 {
			t.Fatalf("after %d tabs cur = %d, want %d", i+1, f.cur, (i+1)%3)
		}
	}
}

func TestFormUpGoesBack(t *testing.T) {
	f := newTestForm()
	f.Update(keyMsg("tab")) // -> 1
	f.Update(keyMsg("up"))  // -> 0
	if f.cur != 0 {
		t.Fatalf("cur = %d, want 0", f.cur)
	}
}

func TestFormPasswordMasked(t *testing.T) {
	f := newTestForm()
	f.fields[2].input.SetValue("secret")
	v := f.View()
	if strings.Contains(v, "secret") {
		t.Fatal("password field rendered unmasked")
	}
	if !strings.Contains(v, "*") {
		t.Fatal("password field rendered without mask glyph")
	}
}

func TestFormFieldNavDoesNotLeakKeys(t *testing.T) {
	f := newTestForm()
	f.Update(keyMsg("tab")) // focus shell
	sub, done, _ := f.Update(keyMsg("j"))
	if sub || done {
		t.Fatal("typing j submitted or cancelled the form")
	}
	if v := f.Values()[1]; v != "j" {
		t.Fatalf("second field = %q, want %q", v, "j")
	}
}
