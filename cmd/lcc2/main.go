// Command lcc2 is a keyboard-first Linux system utility TUI.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lcc2/internal/app"
	"lcc2/internal/screens"
	"lcc2/internal/session"
)

// Version is stamped here; --version reports it.
const Version = "0.3.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"lcc2 %s — keyboard-first Linux system utility TUI\n\n", Version)
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(),
			"\nKeys: 1-6/tab screens · / filter · ? help · q quit")
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("lcc2 " + Version)
		return
	}

	if os.Getenv("NO_COLOR") != "" {
		// Honor the standard: strip all styling before any model runs.
		lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
	}

	// Restore the last session: previous screen and Files prefs.
	st := session.Load()
	filesScr := screens.NewFiles()
	filesScr.Hydrate(st.Cwd, st.Hidden, st.SortKey, st.SortDesc)

	m := app.NewStartingAt(st.Screen,
		screens.NewOverview(),
		screens.NewProcesses(),
		screens.NewDisks(),
		filesScr,
		screens.NewServices(),
		screens.NewUsersGroups(),
	)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lcc2:", err)
		os.Exit(1)
	}
}
