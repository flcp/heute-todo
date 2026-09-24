package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/config"
	"github.com/flcp/heute-todo/internal/store"
	"github.com/flcp/heute-todo/internal/todotxt"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case savedMsg:
		m.err = msg.err
		return m, nil
	case configSavedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	switch m.mode {
	case modeInsert:
		return m.updateInsertMode(msg)
	default:
		if m.normalState.pendingDelete {
			return m.updateDeletePending(msg)
		}
		if m.filter.focus != focusList {
			return m.updateFilterMode(msg)
		}
		return m.updateNormalMode(msg)
	}
}

func (m Model) updateNormalMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		m.moveCursorRelative(1)
	case "k", "up":
		m.moveCursorRelative(-1)
	case "g", "home":
		m.normalState.cursorPosition = 0
	case "G", "end":
		if n := m.visibleCount(); n > 0 {
			m.normalState.cursorPosition = n - 1
		}
	case "tab":
		m.filter.focus = focusProjects
		return m, nil
	case "shift+tab":
		m.filter.focus = focusContexts
		return m, nil
	case "o":
		cmd := m.enterAddMode(true)
		return m, cmd
	case "O":
		cmd := m.enterAddMode(false)
		return m, cmd
	case "i", "I":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.enterEditMode()
		return m, cmd
	case "a", "enter":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.enterEditMode()
		return m, cmd
	case " ":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.toggleDone()
		return m, cmd
	case "J":
		if len(m.todos) > 0 && m.sort == sortFree {
			cmd := m.moveSelected(1)
			return m, cmd
		}
	case "K":
		if len(m.todos) > 0 && m.sort == sortFree {
			cmd := m.moveSelected(-1)
			return m, cmd
		}
	case "d":
		if len(m.todos) > 0 {
			m.normalState.pendingDelete = true
		}
		return m, nil
	case "s":
		m.cycleSortMode()
		return m, m.saveConfigCmd()
	case "z":
		m.filter.hideDone = !m.filter.hideDone
		if last := m.visibleCount() - 1; m.normalState.cursorPosition > last {
			if last < 0 {
				last = 0
			}
			m.normalState.cursorPosition = last
		}
		return m, m.saveConfigCmd()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.setPriority(digitToPriority(int(key.String()[0] - '0')))
		return m, cmd
	}
	return m, nil
}

func (m Model) updateInsertMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			m.commit()
			m.exitInsert()
			cmd := m.saveCmd()
			return m, cmd
		case tea.KeyEsc:
			m.exitInsert()
			return m, nil
		case tea.KeyTab, tea.KeyShiftTab:
			m.tabToField(key.Type == tea.KeyTab)
			return m, nil
		}
	}
	// Forward typed characters and the cursor-blink tick to the text input.
	var cmd tea.Cmd
	m.editState.input, cmd = m.editState.input.Update(msg)
	return m, cmd
}

// tabToField advances to the next (forward) or previous detail field: it first
// drops an empty scaffold left behind on the current field, then scaffolds the
// target field if absent and moves the input cursor into its value slot.
func (m *Model) tabToField(forward bool) {
	v := stripEmptyField(m.editState.input.Value(), m.editState.field)
	next := advanceField(m.editState.field, forward)
	v, pos := ensureField(v, next)
	m.editState.input.SetValue(v)
	m.editState.input.SetCursor(pos)
	m.editState.field = next
}

// updateDeletePending handles the second keystroke of a dd delete: d confirms,
// ctrl+c quits, and any other key cancels.
func (m Model) updateDeletePending(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "d":
		cmd := m.deleteSelected()
		return m, cmd
	case "ctrl+c":
		return m, tea.Quit
	default:
		m.normalState.pendingDelete = false
		return m, nil
	}
}

// updateFilterMode handles input while a header filter box has focus: j/k move
// the box cursor, space/enter toggle a row's selection, Tab cycles focus and Esc
// returns to the list.
func (m Model) updateFilterMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.filter.focus = focusList
	case "tab":
		m.advanceFilterFocus(true)
	case "shift+tab":
		m.advanceFilterFocus(false)
	case "j", "down":
		m.moveFilterCursor(1)
	case "k", "up":
		m.moveFilterCursor(-1)
	case "g", "home":
		m.setFilterCursor(0)
	case "G", "end":
		m.setFilterCursor(m.filterRowCount() - 1)
	case " ", "enter":
		m.toggleFilterRow()
	case "!":
		m.soloFilterRow()
	}
	return m, nil
}

// filterKeys returns the row keys of the currently focused filter box.
func (m Model) filterKeys() []string {
	if m.filter.focus == focusContexts {
		return m.contextKeys()
	}
	return m.projectKeys()
}

// filterRowCount is the number of rows in the focused filter box.
func (m Model) filterRowCount() int { return len(m.filterKeys()) }

// filterCursorPtr points at the cursor field for the focused filter box.
func (m *Model) filterCursorPtr() *int {
	if m.filter.focus == focusContexts {
		return &m.filter.contextCursor
	}
	return &m.filter.projectCursor
}

func (m *Model) moveFilterCursor(delta int) {
	m.setFilterCursor(*m.filterCursorPtr() + delta)
}

// setFilterCursor moves the focused box's cursor to pos, clamped to its rows.
func (m *Model) setFilterCursor(pos int) {
	n := m.filterRowCount()
	if n == 0 {
		*m.filterCursorPtr() = 0
		return
	}
	if pos < 0 {
		pos = 0
	}
	if pos > n-1 {
		pos = n - 1
	}
	*m.filterCursorPtr() = pos
}

// advanceFilterFocus cycles focus list → projects → contexts → list (or the
// reverse when forward is false).
func (m *Model) advanceFilterFocus(forward bool) {
	order := []filterFocus{focusList, focusProjects, focusContexts}
	i := 0
	for idx, f := range order {
		if f == m.filter.focus {
			i = idx
			break
		}
	}
	n := len(order)
	if forward {
		m.filter.focus = order[(i+1)%n]
	} else {
		m.filter.focus = order[(i-1+n)%n]
	}
}

// toggleFilterRow flips the selection of the row under the focused box's cursor,
// then keeps the list cursor within the (possibly smaller) visible set.
func (m *Model) toggleFilterRow() {
	keys := m.filterKeys()
	cursor := *m.filterCursorPtr()
	if cursor < 0 || cursor >= len(keys) {
		return
	}
	off := &m.filter.offContexts
	if m.filter.focus == focusProjects {
		off = &m.filter.offProjects
	}
	if *off == nil {
		*off = map[string]bool{}
	}
	key := keys[cursor]
	if (*off)[key] {
		delete(*off, key)
	} else {
		(*off)[key] = true
	}
	if last := m.visibleCount() - 1; m.normalState.cursorPosition > last {
		if last < 0 {
			last = 0
		}
		m.normalState.cursorPosition = last
	}
}

// soloFilterRow selects only the row under the focused box's cursor, deselecting
// every other row in that dimension. Toggling it again is left to the caller.
func (m *Model) soloFilterRow() {
	keys := m.filterKeys()
	cursor := *m.filterCursorPtr()
	if cursor < 0 || cursor >= len(keys) {
		return
	}
	off := map[string]bool{}
	for _, k := range keys {
		if k != keys[cursor] {
			off[k] = true
		}
	}
	if m.filter.focus == focusProjects {
		m.filter.offProjects = off
	} else {
		m.filter.offContexts = off
	}
	if last := m.visibleCount() - 1; m.normalState.cursorPosition > last {
		if last < 0 {
			last = 0
		}
		m.normalState.cursorPosition = last
	}
}

// cycleSortMode advances to the next sort mode, keeping the cursor on the same
// task after the reorder.
func (m *Model) cycleSortMode() {
	if len(m.todos) == 0 {
		m.sort = (m.sort + 1) % sortModeCount
		return
	}
	sourceIdx := m.cursorSourceIndex()
	m.sort = (m.sort + 1) % sortModeCount
	m.normalState.cursorPosition = m.displayIndexOf(sourceIdx)
}

// moveCursorRelative shifts the cursor by delta, clamped to the list bounds.
func (m *Model) moveCursorRelative(delta int) {
	n := m.visibleCount()
	if n == 0 {
		m.normalState.cursorPosition = 0
		return
	}
	m.normalState.cursorPosition += delta
	if m.normalState.cursorPosition < 0 {
		m.normalState.cursorPosition = 0
	}
	if last := n - 1; m.normalState.cursorPosition > last {
		m.normalState.cursorPosition = last
	}
}

// moveSelected swaps the selected todo with its visible neighbor in direction
// delta (+1 = down, -1 = up), keeping the cursor on the moved item, and
// persists. Cursor positions are display positions, so they are mapped back to
// source indices to skip over any filtered-out todos.
func (m *Model) moveSelected(delta int) tea.Cmd {
	indices := m.displayIndices()
	from := m.normalState.cursorPosition
	to := from + delta
	if from < 0 || from >= len(indices) || to < 0 || to >= len(indices) {
		return nil
	}
	i, j := indices[from], indices[to]
	m.todos[i], m.todos[j] = m.todos[j], m.todos[i]
	m.normalState.cursorPosition = to
	return m.saveCmd()
}

// enterAddMode switches to insert mode to add a new todo, placed below the
// selected todo when below is true, otherwise above it.
func (m *Model) enterAddMode(below bool) tea.Cmd {
	at := m.cursorSourceIndex()
	if below {
		at++
	}
	m.mode = modeInsert
	m.editState.isAddingItem = true
	m.editState.insertAt = at
	m.editState.field = fieldTitle
	m.editState.input.Reset()
	return m.editState.input.Focus()
}

// enterEditMode switches to insert mode to edit the selected todo, starting on
// the title field with the cursor at the title's value.
func (m *Model) enterEditMode() tea.Cmd {
	m.mode = modeInsert
	m.editState.isAddingItem = false
	m.editState.field = fieldTitle
	currentTodo := m.todos[m.cursorSourceIndex()]
	value := currentTodo.String()
	m.editState.input.SetValue(value)
	pos, _ := locateField(value, fieldTitle)
	m.editState.input.SetCursor(pos)
	return m.editState.input.Focus()
}

// setPriority sets the priority letter of the selected todo, keeps the cursor on
// it after any re-sort, and persists.
func (m *Model) setPriority(p byte) tea.Cmd {
	i := m.cursorSourceIndex()
	m.todos[i].Priority = p
	m.normalState.cursorPosition = m.displayIndexOf(i)
	return m.saveCmd()
}

// toggleDone flips the done state of the selected todo (stamping or clearing the
// completion date) and persists the change.
func (m *Model) toggleDone() tea.Cmd {
	i := m.cursorSourceIndex()
	m.todos[i] = m.todos[i].Toggled(time.Now())
	m.normalState.cursorPosition = m.displayIndexOf(i)
	return m.saveCmd()
}

// deleteSelected removes the todo under the cursor, clamps the cursor to the new
// bounds, and persists the change.
func (m *Model) deleteSelected() tea.Cmd {
	m.normalState.pendingDelete = false
	i := m.cursorSourceIndex()
	m.todos = append(m.todos[:i], m.todos[i+1:]...)
	if last := m.visibleCount() - 1; m.normalState.cursorPosition > last {
		m.normalState.cursorPosition = last
	}
	if m.normalState.cursorPosition < 0 {
		m.normalState.cursorPosition = 0
	}
	return m.saveCmd()
}

// commit applies the input text as a new or edited todo. Blank input is ignored.
func (m *Model) commit() {
	// Drop an unfilled scaffold (e.g. a trailing "due:") left on the current
	// field so it never persists.
	value := stripEmptyField(m.editState.input.Value(), m.editState.field)
	newTodo, ok := todotxt.Parse(value)
	if !ok {
		return
	}

	var index int
	if m.editState.isAddingItem {
		index = m.editState.insertAt
		if index < 0 {
			index = 0
		}
		if index > len(m.todos) {
			index = len(m.todos)
		}
		if newTodo.CreatedAt == nil {
			now := time.Now()
			newTodo.CreatedAt = &now
		}
		m.todos = append(m.todos, todotxt.Todo{})
		copy(m.todos[index+1:], m.todos[index:])
	} else {
		if len(m.todos) == 0 {
			return
		}
		index = m.cursorSourceIndex()
	}

	m.todos[index] = newTodo
	m.normalState.cursorPosition = m.displayIndexOf(index)
}

// exitInsert returns to normal mode and clears the input.
func (m *Model) exitInsert() {
	m.mode = modeNormal
	m.editState.field = fieldTitle
	m.editState.input.Blur()
	m.editState.input.Reset()
}

// savedMsg reports the result of an autosave.
type savedMsg struct{ err error }

// saveCmd persists the current todos to disk. It snapshots the slice so the
// write runs safely off the update loop.
func (m Model) saveCmd() tea.Cmd {
	path := m.path
	snapshot := make([]todotxt.Todo, len(m.todos))
	copy(snapshot, m.todos)
	return func() tea.Msg {
		return savedMsg{err: store.Save(path, snapshot)}
	}
}

// configSavedMsg reports the result of persisting the config.
type configSavedMsg struct{ err error }

// saveConfigCmd persists the model's current preferences (sort, done visibility,
// theme, path) to the config file off the update loop.
func (m Model) saveConfigCmd() tea.Cmd {
	cfg := m.toConfig()
	return func() tea.Msg {
		return configSavedMsg{err: config.Save(cfg)}
	}
}
