package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/ui"
)

func main() {
	path := "todo.txt"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	model, err := ui.New(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "heute: %v\n", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "heute: %v\n", err)
		os.Exit(1)
	}
}
