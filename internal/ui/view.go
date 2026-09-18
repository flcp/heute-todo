package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/flcp/heute-todo/internal/todotxt"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("57"))

	doneStyle = lipgloss.NewStyle().
			Faint(true).
			Strikethrough(true)

	priorityStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	emptyStyle = lipgloss.NewStyle().Faint(true)

	helpStyle = lipgloss.NewStyle().Faint(true)

	insertStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	errStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))
)

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	done := 0
	for _, t := range m.todos {
		if t.Done {
			done++
		}
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("HEUTE TODO — %d tasks, %d done", len(m.todos), done)))
	b.WriteString("\n\n")

	if len(m.todos) == 0 {
		b.WriteString(emptyStyle.Render("  (no tasks yet)"))
		b.WriteByte('\n')
	}
	for i, t := range m.todos {
		b.WriteString(m.renderRow(i, t))
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	if m.mode == modeInsert {
		b.WriteString(insertStyle.Render("Add"))
		b.WriteByte(' ')
		b.WriteString(m.editState.input.View())
	} else {
		b.WriteString(helpStyle.Render("j/k move · g/G top/bottom · o/O add · i/I/a edit · q quit"))
	}
	if m.err != nil {
		b.WriteByte('\n')
		b.WriteString(errStyle.Render(fmt.Sprintf("save failed: %v", m.err)))
	}
	return b.String()
}

func (m Model) renderRow(i int, t todotxt.Todo) string {
	prefix := "  "
	if i == m.normalState.cursorPosition {
		prefix = "> "
	}

	check := "[ ]"
	if t.Done {
		check = "[x]"
	}

	var badge string
	if t.Priority >= 'A' && t.Priority <= 'Z' {
		badge = fmt.Sprintf("(%c) ", t.Priority)
	}

	text := fmt.Sprintf("%s%s %s%s", prefix, check, badge, t.Description)

	switch {
	case i == m.normalState.cursorPosition:
		return selectedStyle.Render(text)
	case t.Done:
		return doneStyle.Render(text)
	case badge != "":
		return priorityStyle.Render(text)
	default:
		return text
	}
}
