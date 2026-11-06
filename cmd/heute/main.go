package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/config"
	"github.com/flcp/heute-todo/internal/ui"
)

func main() {
	cfg, existed, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "heute: reading config: %v\n", err)
	}

	// Path precedence: CLI arg overrides the configured default; the resolved
	// path becomes the remembered default.
	loadedPath := cfg.Path
	path := loadedPath
	if path == "" {
		path = "todo.txt"
	}
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	cfg.Path = path

	// Materialize the config file on first run, or when a CLI arg changed the
	// remembered default, so it exists to edit and reflects the current file.
	if !existed || path != loadedPath {
		_ = config.Save(cfg)
	}

	model, err := ui.New(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "heute: %v\n", err)
		os.Exit(1)
	}
	model = model.WithConfig(cfg)

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "heute: %v\n", err)
		os.Exit(1)
	}
}
