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
	return Model{todos: todos}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func step(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestNavigationDownUpClamps(t *testing.T) {
	m := testModel(3)

	m = step(m, key("j"))
	if m.cursor != 1 {
		t.Fatalf("after j cursor = %d, want 1", m.cursor)
	}

	m = step(m, key("j"))
	m = step(m, key("j")) // should clamp at the last item
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want clamped 2", m.cursor)
	}

	m = step(m, key("k"))
	if m.cursor != 1 {
		t.Fatalf("after k cursor = %d, want 1", m.cursor)
	}
}

func TestNavigationTopBottom(t *testing.T) {
	m := testModel(5)

	m = step(m, key("G"))
	if m.cursor != 4 {
		t.Fatalf("after G cursor = %d, want 4", m.cursor)
	}

	m = step(m, key("g"))
	if m.cursor != 0 {
		t.Fatalf("after g cursor = %d, want 0", m.cursor)
	}
}

func TestNavigationEmptyListStaysAtZero(t *testing.T) {
	m := testModel(0)
	m = step(m, key("j"))
	m = step(m, key("G"))
	if m.cursor != 0 {
		t.Fatalf("empty-list cursor = %d, want 0", m.cursor)
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
