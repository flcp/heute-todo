package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/todotxt"
)

func testModel(n int) Model {
	todos := make([]todotxt.Todo, n)
	for i := range todos {
		todos[i] = todotxt.Todo{Description: string(rune('a' + i))}
	}
	return Model{todos: todos, editState: editState{input: newInput()}}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func enter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

func esc() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEsc} }

func step(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestNavigationDownUpClamps(t *testing.T) {
	m := testModel(3)

	m = step(m, key("j"))
	if m.normalState.cursorPosition != 1 {
		t.Fatalf("after j cursor = %d, want 1", m.normalState.cursorPosition)
	}

	m = step(m, key("j"))
	m = step(m, key("j")) // should clamp at the last item
	if m.normalState.cursorPosition != 2 {
		t.Fatalf("cursor = %d, want clamped 2", m.normalState.cursorPosition)
	}

	m = step(m, key("k"))
	if m.normalState.cursorPosition != 1 {
		t.Fatalf("after k cursor = %d, want 1", m.normalState.cursorPosition)
	}
}

func TestNavigationTopBottom(t *testing.T) {
	m := testModel(5)

	m = step(m, key("G"))
	if m.normalState.cursorPosition != 4 {
		t.Fatalf("after G cursor = %d, want 4", m.normalState.cursorPosition)
	}

	m = step(m, key("g"))
	if m.normalState.cursorPosition != 0 {
		t.Fatalf("after g cursor = %d, want 0", m.normalState.cursorPosition)
	}
}

func TestNavigationEmptyListStaysAtZero(t *testing.T) {
	m := testModel(0)
	m = step(m, key("j"))
	m = step(m, key("G"))
	if m.normalState.cursorPosition != 0 {
		t.Fatalf("empty-list cursor = %d, want 0", m.normalState.cursorPosition)
	}
}

func TestQuitCommand(t *testing.T) {
	m := testModel(1)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("expected a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected a quit message")
	}
}

func TestAddBelowInsertsAndSelects(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("o"))
	if m.mode != modeInsert {
		t.Fatal("o should enter insert mode")
	}

	m = step(m, key("buy milk"))
	m = step(m, enter())

	if m.mode != modeNormal {
		t.Fatal("enter should return to normal mode")
	}
	if len(m.todos) != 3 {
		t.Fatalf("len = %d, want 3", len(m.todos))
	}
	if m.todos[1].Description != "buy milk" {
		t.Fatalf("todos[1] = %q, want \"buy milk\"", m.todos[1].Description)
	}
	if m.normalState.cursorPosition != 1 {
		t.Fatalf("cursor = %d, want 1", m.normalState.cursorPosition)
	}
}

func TestAddAboveInsertsAtCursor(t *testing.T) {
	m := testModel(2)     // a, b
	m = step(m, key("j")) // select b (index 1)

	m = step(m, key("O"))
	m = step(m, key("preface"))
	m = step(m, enter())

	if m.todos[1].Description != "preface" {
		t.Fatalf("todos[1] = %q, want \"preface\"", m.todos[1].Description)
	}
	if m.normalState.cursorPosition != 1 {
		t.Fatalf("cursor = %d, want 1", m.normalState.cursorPosition)
	}
}

func TestEditReplacesSelected(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("i"))
	if m.mode != modeInsert || m.editState.isAddingItem {
		t.Fatal("i should enter insert mode for editing")
	}

	m.editState.input.SetValue("(A) reworded")
	m = step(m, enter())

	if len(m.todos) != 2 {
		t.Fatalf("len = %d, want 2 (edit must not add)", len(m.todos))
	}
	if m.todos[0].Priority != 'A' || m.todos[0].Description != "reworded" {
		t.Fatalf("todos[0] = %+v, want (A) \"reworded\"", m.todos[0])
	}
}

func TestEscCancelsInsert(t *testing.T) {
	m := testModel(1) // a

	m = step(m, key("o"))
	m = step(m, key("junk"))
	m = step(m, esc())

	if m.mode != modeNormal {
		t.Fatal("esc should return to normal mode")
	}
	if len(m.todos) != 1 {
		t.Fatalf("len = %d, want 1 (esc must not add)", len(m.todos))
	}
}

func TestBlankInsertIgnored(t *testing.T) {
	m := testModel(1) // a

	m = step(m, key("o"))
	m = step(m, enter()) // committed with empty input

	if m.mode != modeNormal {
		t.Fatal("should return to normal mode")
	}
	if len(m.todos) != 1 {
		t.Fatalf("len = %d, want 1 (blank must not add)", len(m.todos))
	}
}
