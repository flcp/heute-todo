// Package ui implements the Bubble Tea model, update loop and view for the
// heute todo TUI.
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/store"
	"github.com/flcp/heute-todo/internal/todotxt"
)

// mode is the editor's modal state.
type mode int

const (
	modeNormal mode = iota
	modeInsert
)

// Model is the root Bubble Tea model.
type Model struct {
	path  string
	todos []todotxt.Todo
	mode  mode

	normalState normalState
	editState   editState
}

// normalState holds state used while navigating in normal mode.
type normalState struct {
	cursorPosition int
}

// editState holds state used while adding or editing a todo in insert mode.
type editState struct {
	input        textinput.Model
	insertAt     int  // index a newly added todo lands at
	isAddingItem bool // add a new todo rather than replacing the selected one
}

// New creates a Model backed by the todo.txt file at path, loading any existing
// todos.
func New(path string) (Model, error) {
	todos, err := store.Load(path)
	if err != nil {
		return Model{}, err
	}
	return Model{
		path:      path,
		todos:     todos,
		editState: editState{input: newInput()},
	}, nil
}

// newInput builds the textinput used to add and edit todos in insert mode.
func newInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "(A) Buy milk +errands @home due:2026-09-20"
	return ti
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
