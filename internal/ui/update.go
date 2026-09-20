package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/store"
	"github.com/flcp/heute-todo/internal/todotxt"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case savedMsg:
		m.err = msg.err
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
		if len(m.todos) > 0 {
			m.normalState.cursorPosition = len(m.todos) - 1
		}
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
		cmd := m.enterEditMode(true)
		return m, cmd
	case "a":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.enterEditMode(false)
		return m, cmd
	case " ":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.toggleDone()
		return m, cmd
	case "J":
		if len(m.todos) > 0 {
			cmd := m.moveSelected(1)
			return m, cmd
		}
	case "K":
		if len(m.todos) > 0 {
			cmd := m.moveSelected(-1)
			return m, cmd
		}
	case "d":
		if len(m.todos) > 0 {
			m.normalState.pendingDelete = true
		}
		return m, nil
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
		}
	}
	// Forward typed characters and the cursor-blink tick to the text input.
	var cmd tea.Cmd
	m.editState.input, cmd = m.editState.input.Update(msg)
	return m, cmd
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

// moveCursorRelative shifts the cursor by delta, clamped to the list bounds.
func (m *Model) moveCursorRelative(delta int) {
	if len(m.todos) == 0 {
		m.normalState.cursorPosition = 0
		return
	}
	m.normalState.cursorPosition += delta
	if m.normalState.cursorPosition < 0 {
		m.normalState.cursorPosition = 0
	}
	if last := len(m.todos) - 1; m.normalState.cursorPosition > last {
		m.normalState.cursorPosition = last
	}
}

// moveSelected swaps the selected todo with its neighbor in direction delta
// (+1 = down, -1 = up), keeping the cursor on the moved item, and persists.
func (m *Model) moveSelected(delta int) tea.Cmd {
	i := m.normalState.cursorPosition
	j := i + delta
	if j < 0 || j >= len(m.todos) {
		return nil
	}
	m.todos[i], m.todos[j] = m.todos[j], m.todos[i]
	m.normalState.cursorPosition = j
	return m.saveCmd()
}

// enterAddMode switches to insert mode to add a new todo, placed below the
// selected todo when below is true, otherwise above it.
func (m *Model) enterAddMode(below bool) tea.Cmd {
	at := m.normalState.cursorPosition
	if below {
		at++
	}
	m.mode = modeInsert
	m.editState.isAddingItem = true
	m.editState.insertAt = at
	m.editState.input.Reset()
	return m.editState.input.Focus()
}

// enterEditMode switches to insert mode to edit the selected todo. The input
// cursor starts at the front of the line when cursorAtStart is true, otherwise
// at the end.
func (m *Model) enterEditMode(cursorAtStart bool) tea.Cmd {
	m.mode = modeInsert
	m.editState.isAddingItem = false
	currentTodo := m.todos[m.normalState.cursorPosition]
	m.editState.input.SetValue(currentTodo.String())
	if cursorAtStart {
		m.editState.input.CursorStart()
	} else {
		m.editState.input.CursorEnd()
	}
	return m.editState.input.Focus()
}

// toggleDone flips the done state of the selected todo (stamping or clearing the
// completion date) and persists the change.
func (m *Model) toggleDone() tea.Cmd {
	i := m.normalState.cursorPosition
	m.todos[i] = m.todos[i].Toggled(time.Now())
	return m.saveCmd()
}

// deleteSelected removes the todo under the cursor, clamps the cursor to the new
// bounds, and persists the change.
func (m *Model) deleteSelected() tea.Cmd {
	m.normalState.pendingDelete = false
	i := m.normalState.cursorPosition
	m.todos = append(m.todos[:i], m.todos[i+1:]...)
	if last := len(m.todos) - 1; m.normalState.cursorPosition > last {
		m.normalState.cursorPosition = last
	}
	if m.normalState.cursorPosition < 0 {
		m.normalState.cursorPosition = 0
	}
	return m.saveCmd()
}

// commit applies the input text as a new or edited todo. Blank input is ignored.
func (m *Model) commit() {
	newTodo, ok := todotxt.Parse(m.editState.input.Value())
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
		m.todos = append(m.todos, todotxt.Todo{})
		copy(m.todos[index+1:], m.todos[index:])
	} else {
		if m.normalState.cursorPosition >= len(m.todos) {
			return
		}
		index = m.normalState.cursorPosition
	}

	m.todos[index] = newTodo
	m.normalState.cursorPosition = index
}

// exitInsert returns to normal mode and clears the input.
func (m *Model) exitInsert() {
	m.mode = modeNormal
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
