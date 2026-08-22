// Command lcc2 is a keyboard-first Linux system utility TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lcc2/internal/app"
	"lcc2/internal/screens"
)

func main() {
	m := app.New(
		screens.NewOverview(),
		screens.NewProcesses(),
		screens.NewDisks(),
		screens.NewFiles(),
		screens.NewServices(),
		screens.NewUsersGroups(),
	)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lcc2:", err)
		os.Exit(1)
	}
}
