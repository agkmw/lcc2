package ui

import "github.com/charmbracelet/bubbles/key"

// Keys is the global keymap shared across screens.
var Keys = struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Filter  key.Binding
	Select  key.Binding
	Back    key.Binding
	Help    key.Binding
	Quit    key.Binding
	Refresh key.Binding
}{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/left", "back")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/right", "open")),
	Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
}
