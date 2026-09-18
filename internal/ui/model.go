// Package ui implements the Bubble Tea model, update loop and view for the
// heute todo TUI.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/store"
	"github.com/flcp/heute-todo/internal/todotxt"
)

// Model is the root Bubble Tea model. For this milestone it holds the loaded
// todos and the navigation cursor; editing state arrives in later milestones.
type Model struct {
	path   string
	todos  []todotxt.Todo
	cursor int
}

// New creates a Model backed by the todo.txt file at path, loading any existing
// todos.
func New(path string) (Model, error) {
	todos, err := store.Load(path)
	if err != nil {
		return Model{}, err
	}
	return Model{path: path, todos: todos}, nil
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
