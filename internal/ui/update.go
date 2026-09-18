package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/store"
	"github.com/flcp/heute-todo/internal/todotxt"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if saved, ok := msg.(savedMsg); ok {
		m.err = saved.err
		return m, nil
	}
	switch m.mode {
	case modeInsert:
		return m.updateInsertMode(msg)
	default:
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
		cmd := m.enterAddMode(m.normalState.cursorPosition + 1)
		return m, cmd
	case "O":
		cmd := m.enterAddMode(m.normalState.cursorPosition)
		return m, cmd
	case "i", "I":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.enterEditMode(m.todos[m.normalState.cursorPosition], true)
		return m, cmd
	case "a":
		if len(m.todos) == 0 {
			return m, nil
		}
		cmd := m.enterEditMode(m.todos[m.normalState.cursorPosition], false)
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
		}
	}
	// Forward typed characters and the cursor-blink tick to the text input.
	var cmd tea.Cmd
	m.editState.input, cmd = m.editState.input.Update(msg)
	return m, cmd
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

// enterAddMode switches to insert mode to add a new todo, inserted at index at when
// committed.
func (m *Model) enterAddMode(at int) tea.Cmd {
	m.mode = modeInsert
	m.editState.isAddingItem = true
	m.editState.insertAt = at
	m.editState.input.Reset()
	return m.editState.input.Focus()
}

// enterEditMode switches to insert mode to edit todo t. The input cursor starts
// at the front of the line when cursorAtStart is true, otherwise at the end.
func (m *Model) enterEditMode(t todotxt.Todo, cursorAtStart bool) tea.Cmd {
	m.mode = modeInsert
	m.editState.isAddingItem = false
	m.editState.input.SetValue(t.String())
	if cursorAtStart {
		m.editState.input.CursorStart()
	} else {
		m.editState.input.CursorEnd()
	}
	return m.editState.input.Focus()
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
