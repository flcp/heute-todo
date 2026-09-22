// Package ui implements the Bubble Tea model, update loop and view for the
// heute todo TUI.
package ui

import (
	"sort"

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
	sortFree      sortMode = iota // file order
	sortPriority                  // priority order (A first, unprioritized last)
	sortName                      // alphabetical by description
	sortModeCount                 // sentinel: total number of sort modes
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
	filter      filterState

	styles Styles // active theme's rendered styles

	err error // last autosave error, shown in the footer
}

// normalState holds state used while navigating in normal mode.
type normalState struct {
	cursorPosition int
	pendingDelete  bool // armed by the first d of a dd delete
}

// filterFocus identifies where keyboard input is directed: the todo list or one
// of the header filter boxes.
type filterFocus int

const (
	focusList filterFocus = iota
	focusProjects
	focusContexts
)

// filterState holds the project/context filter selections and which box (if any)
// currently has focus. Deselected keys are stored (rather than selected ones) so
// that newly-added tags default to visible ("all selected"). The empty-string
// key "" represents the "(none)" row: untagged todos.
type filterState struct {
	focus         filterFocus
	projectCursor int
	contextCursor int
	offProjects   map[string]bool
	offContexts   map[string]bool
	hideDone      bool // hide completed tasks from the list
}

// editField identifies a field in the detail panel that edit mode can jump the
// cursor to with Tab.
type editField int

const (
	fieldTitle editField = iota
	fieldPriority
	fieldProject
	fieldContext
	fieldDue
	fieldDetails
)

// editFieldOrder is the Tab cycle through the detail panel's editable fields.
// Created/Done dates are shown but are read-only and skipped.
var editFieldOrder = []editField{fieldTitle, fieldPriority, fieldProject, fieldContext, fieldDue, fieldDetails}

// advanceField returns the field after (forward) or before f in editFieldOrder,
// wrapping around the ends.
func advanceField(f editField, forward bool) editField {
	i := int(f)
	n := len(editFieldOrder)
	if forward {
		return editFieldOrder[(i+1)%n]
	}
	return editFieldOrder[(i-1+n)%n]
}

// editState holds state used while adding or editing a todo in insert mode.
type editState struct {
	input        textinput.Model
	insertAt     int       // index a newly added todo lands at
	isAddingItem bool      // add a new todo rather than replacing the selected one
	field        editField // field the Tab cursor currently targets
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

// displayIndices returns a slice mapping display-position → m.todos index, in
// the active sort order and with the project/context filter applied.
func (m Model) displayIndices() []int {
	var order []int
	switch m.sort {
	case sortPriority:
		order = todotxt.SortIndicesByPriority(m.todos)
	case sortName:
		order = todotxt.SortIndicesByName(m.todos)
	default:
		order = make([]int, len(m.todos))
		for i := range order {
			order[i] = i
		}
	}
	filtered := order[:0:0]
	for _, i := range order {
		if m.passesFilter(m.todos[i]) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

// visibleCount is the number of todos currently shown after filtering.
func (m Model) visibleCount() int {
	return len(m.displayIndices())
}

// noneKey is the internal key for the "(none)" filter row: todos with no tag in
// that dimension.
const noneKey = ""

// projectKeys returns the filter rows for projects: the "(none)" key followed by
// the sorted unique +projects present across all todos.
func (m Model) projectKeys() []string {
	return tagKeys(m.todos, func(t todotxt.Todo) []string { return t.Projects })
}

// contextKeys returns the filter rows for contexts: the "(none)" key followed by
// the sorted unique @contexts present across all todos.
func (m Model) contextKeys() []string {
	return tagKeys(m.todos, func(t todotxt.Todo) []string { return t.Contexts })
}

// tagKeys collects the sorted unique tags produced by tagsOf, prefixed with the
// "(none)" key.
func tagKeys(todos []todotxt.Todo, tagsOf func(todotxt.Todo) []string) []string {
	set := map[string]bool{}
	for _, t := range todos {
		for _, tag := range tagsOf(t) {
			set[tag] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return append([]string{noneKey}, names...)
}

// passesFilter reports whether t survives both the project and context filters.
// A dimension passes when any of the todo's keys in that dimension (its tags, or
// the "(none)" key when it has none) is currently selected.
func (m Model) passesFilter(t todotxt.Todo) bool {
	if m.filter.hideDone && t.Done {
		return false
	}
	return passesDimension(t.Projects, m.filter.offProjects) &&
		passesDimension(t.Contexts, m.filter.offContexts)
}

func passesDimension(tags []string, off map[string]bool) bool {
	if len(tags) == 0 {
		return !off[noneKey]
	}
	for _, tag := range tags {
		if !off[tag] {
			return true
		}
	}
	return false
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
