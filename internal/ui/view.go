package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/flcp/heute-todo/internal/todotxt"
)

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	b.WriteString(m.renderBody())
	b.WriteString("\n\n")

	switch {
	case m.mode == modeInsert:
		b.WriteString(m.styles.Insert.Render("Add"))
		b.WriteByte(' ')
		b.WriteString(m.editState.input.View())
	case m.normalState.pendingDelete:
		b.WriteString(m.styles.Delete.Render("delete? press d to confirm, esc to cancel"))
	default:
		b.WriteString(m.styles.Help.Render("j/k move · g/G top/bottom · space done · o/O add · i/I/a edit · dd delete · q quit"))
	}
	if m.err != nil {
		b.WriteByte('\n')
		b.WriteString(m.styles.Err.Render(fmt.Sprintf("save failed: %v", m.err)))
	}
	return b.String()
}

// loremIpsum fills the side panel with placeholder copy.
const loremIpsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
	"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
	"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris."

// renderBody lays out the main area as two equal halves separated by a vertical
// rule: the todo list on the left and an info panel on the right.
func (m Model) renderBody() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	// A one-column rule splits the remaining width into two equal halves.
	const sepW = 1
	leftW := (width - sepW) / 2
	rightW := width - sepW - leftW
	if leftW < 10 {
		leftW = 10
	}
	if rightW < 10 {
		rightW = 10
	}

	var list strings.Builder
	if len(m.todos) == 0 {
		list.WriteString(m.styles.Empty.Render("(no tasks yet)"))
	}
	for i, t := range m.todos {
		if i > 0 {
			list.WriteByte('\n')
		}
		list.WriteString(m.renderRow(i, t))
	}

	todosPanel := m.styles.TodoPanel.Width(leftW)
	sidePanel := m.styles.SidePanel.Width(rightW)

	left := todosPanel.Render(list.String())
	right := sidePanel.Render(loremIpsum)

	// Match heights so the separator spans the taller half.
	h := lipgloss.Height(left)
	if rh := lipgloss.Height(right); rh > h {
		h = rh
	}

	left = todosPanel.Height(h).Render(list.String())
	right = sidePanel.Height(h).Render(loremIpsum)
	sep := m.styles.Separator.Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

// renderHeader lays out the top bar as three bordered panels: the app name on
// the left, the todo file path in the middle, and the open-task count on the
// right.
func (m Model) renderHeader() string {
	open := 0
	for _, t := range m.todos {
		if !t.Done {
			open++
		}
	}

	width := m.width
	if width <= 0 {
		width = 80
	}

	const name = "HEUTE"
	openText := fmt.Sprintf("%d open", open)

	// Each panel adds two columns of border, so three panels cost six columns.
	const borders = 6
	leftW := lipgloss.Width(name) + 2 // +2 for the panel's horizontal padding
	rightW := lipgloss.Width(openText) + 2
	midW := width - borders - leftW - rightW
	if midW < 8 {
		midW = 8
	}

	left := m.styles.HeaderName.Width(leftW).Render(name)
	mid := m.styles.HeaderPath.Width(midW).Render(truncateLeft(m.absPath(), midW-2))
	right := m.styles.HeaderCount.Width(rightW).Render(openText)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
}

// absPath returns the todo file's absolute path, falling back to the stored
// path if it cannot be resolved.
func (m Model) absPath() string {
	abs, err := filepath.Abs(m.path)
	if err != nil {
		return m.path
	}
	return abs
}

// truncateLeft shortens s to at most max display columns, dropping characters
// from the front and prefixing an ellipsis so the end (the file name) stays
// visible.
func truncateLeft(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return "…" + string(r[len(r)-(max-1):])
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
		return m.styles.Selected.Render(text)
	case t.Done:
		return m.styles.Done.Render(text)
	case badge != "":
		return m.styles.Priority.Render(text)
	default:
		return text
	}
}
