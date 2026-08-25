package ui

import "charm.land/bubbles/v2/key"

// Keys is the global keymap shared across screens.
var Keys = struct {
	Filter  key.Binding
	Select  key.Binding
	Back    key.Binding
	Refresh key.Binding
}{
	Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
}
