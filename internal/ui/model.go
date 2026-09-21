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

// sortMode controls how todos are ordered in the list view.
type sortMode int

const (
	sortFree     sortMode = iota // file order
	sortPriority                 // priority order (A first, unprioritized last)
)

// Model is the root Bubble Tea model.
type Model struct {
	path   string
	todos  []todotxt.Todo
	mode   mode
	sort   sortMode
	width  int // terminal width, used to lay out the header panels
	height int // terminal height, used to pin the command line to the bottom

	normalState normalState
	editState   editState

	styles Styles // active theme's rendered styles

	err error // last autosave error, shown in the footer
}

// normalState holds state used while navigating in normal mode.
type normalState struct {
	cursorPosition int
	pendingDelete  bool // armed by the first d of a dd delete
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
		styles:    buildStylesWithPalette(paletteFor(DefaultTheme)),
	}, nil
}

// WithTheme returns a copy of the model styled with the named theme, falling
// back to Nord when the name is not registered.
func (m Model) WithTheme(name string) Model {
	m.styles = buildStylesWithPalette(paletteFor(name))
	return m
}

// newInput builds the textinput used to add and edit todos in insert mode.
func newInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "(A) Buy milk +errands @home due:2026-09-20"
	return ti
}

// displayIndices returns a slice mapping display-position → m.todos index.
func (m Model) displayIndices() []int {
	if m.sort == sortPriority {
		return todotxt.SortIndicesByPriority(m.todos)
	}
	idx := make([]int, len(m.todos))
	for i := range idx {
		idx[i] = i
	}
	return idx
}

// cursorSourceIndex returns the m.todos index for the item under the cursor.
func (m Model) cursorSourceIndex() int {
	if len(m.todos) == 0 {
		return 0
	}
	indices := m.displayIndices()
	if m.normalState.cursorPosition >= len(indices) {
		return 0
	}
	return indices[m.normalState.cursorPosition]
}

// displayIndexOf returns the display position for the given m.todos source index.
func (m Model) displayIndexOf(sourceIdx int) int {
	for dispIdx, srcIdx := range m.displayIndices() {
		if srcIdx == sourceIdx {
			return dispIdx
		}
	}
	return 0
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
