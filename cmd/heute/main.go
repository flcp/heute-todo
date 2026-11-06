package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/config"
	"github.com/flcp/heute-todo/internal/ui"
)

func main() {
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: heute [--set-path <path>]")
			fmt.Println()
			fmt.Println("  --set-path <path>  Set and save the todo.txt file path")
			fmt.Println("  --help, -h         Show this help message")
			os.Exit(0)
		case "--set-path":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "heute: --set-path requires a path argument")
				os.Exit(1)
			}
			cfg, _, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "heute: reading config: %v\n", err)
			}
			cfg.Path = args[i+1]
			if err := config.Save(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "heute: saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("heute: path set to %s\n", cfg.Path)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "heute: unknown flag %q\n", args[i])
			fmt.Fprintln(os.Stderr, "Run 'heute --help' for usage.")
			os.Exit(1)
		}
	}

	cfg, existed, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "heute: reading config: %v\n", err)
	}

	path := cfg.Path
	if path == "" {
		path = "todo.txt"
		cfg.Path = path
	}

	if !existed {
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
