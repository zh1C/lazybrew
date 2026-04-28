package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zh1C/lazybrew/pkg/gui"
)

func main() {
	app := gui.NewApp()

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazybrew: %v\n", err)
		os.Exit(1)
	}
}
