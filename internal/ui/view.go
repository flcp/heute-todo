package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/flcp/heute-todo/internal/todotxt"
)

// View implements tea.Model.
func (m Model) View() string {
	top := m.renderHeader() + "\n\n" + m.renderBody()
	if m.err != nil {
		top += "\n\n" + m.styles.Err.Render(fmt.Sprintf("save failed: %v", m.err))
	}

	cmd := m.renderCommandLine()

	// Pin the command line to the very bottom by padding the space between it
	// and the content above when we know the terminal height.
	gap := "\n\n"
	if m.height > 0 {
		if n := m.height - lipgloss.Height(top) - lipgloss.Height(cmd) + 1; n > 2 {
			gap = strings.Repeat("\n", n)
		}
	}
	return top + gap + cmd
}

// renderCommandLine renders the footer prompt as a full-width bordered panel,
// matching the header style.
func (m Model) renderCommandLine() string {
	var content string
	switch {
	case m.mode == modeInsert:
		content = m.styles.Insert.Render("Add") + " " + m.editState.input.View()
	case m.normalState.pendingDelete:
		content = m.styles.Delete.Render("delete? press d to confirm, esc to cancel")
	default:
		content = m.styles.Help.Render("j/k move · g/G top/bottom · space done · o/O add · i/I/a edit · dd delete · q quit")
	}

	width := m.width
	if width <= 0 {
		width = 80
	}
	return m.styles.CommandLine.Width(width - 2).Render(content)
}

// renderDetail renders the side panel: a detail view of the todo currently
// under the cursor, or a hint when the list is empty.
func (m Model) renderDetail() string {
	if len(m.todos) == 0 || m.normalState.cursorPosition >= len(m.todos) {
		return m.styles.Empty.Render("(no task selected)")
	}
	t := m.todos[m.normalState.cursorPosition]

	var b strings.Builder
	b.WriteString(m.styles.DetailTitle.Render(detailTitle(t)))
	b.WriteString("\n\n")

	priority := "—"
	if t.Priority >= 'A' && t.Priority <= 'Z' {
		priority = string(t.Priority)
	}
	b.WriteString(m.detailField("Priority", priority))

	if len(t.Projects) > 0 {
		b.WriteString(m.detailField("Project", strings.Join(t.Projects, ", ")))
	}
	if len(t.Contexts) > 0 {
		b.WriteString(m.detailField("Context", strings.Join(t.Contexts, ", ")))
	}
	if due, ok := t.Tags["due"]; ok {
		b.WriteString(m.detailField("Due date", formatDue(due, time.Now())))
	}

	return strings.TrimRight(b.String(), "\n")
}

// detailField renders a single "Label: value" line, terminated by a newline.
func (m Model) detailField(label, value string) string {
	return m.styles.DetailLabel.Render(label+":") + " " + value + "\n"
}

// detailTitle returns the task description with its +projects, @contexts and
// key:value tags stripped, leaving just the human-readable title.
func detailTitle(t todotxt.Todo) string {
	title, _ := splitDescription(t.Description)
	if title == "" {
		return "(untitled)"
	}
	return title
}

// splitDescription separates a raw todo description into its human-readable
// title and the trailing metadata (+projects, @contexts and key:value tags),
// each with the original token order preserved.
func splitDescription(desc string) (title, meta string) {
	var titleWords, metaWords []string
	for _, tok := range strings.Fields(desc) {
		switch {
		case len(tok) > 1 && (tok[0] == '+' || tok[0] == '@'):
			metaWords = append(metaWords, tok)
		case strings.ContainsRune(tok, ':') && !strings.ContainsAny(tok, " "):
			// Drop key:value tags (e.g. due:2026-09-20).
			if i := strings.IndexByte(tok, ':'); i > 0 && i < len(tok)-1 {
				metaWords = append(metaWords, tok)
				continue
			}
			titleWords = append(titleWords, tok)
		default:
			titleWords = append(titleWords, tok)
		}
	}
	return strings.Join(titleWords, " "), strings.Join(metaWords, " ")
}

// formatDue renders a due date with a relative countdown, e.g.
// "2026-09-20 (21d remaining)".
func formatDue(due string, now time.Time) string {
	d, err := time.Parse(todotxt.DateLayout, due)
	if err != nil {
		return due
	}
	// Compare on date boundaries, ignoring the time of day.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	target := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	days := int(target.Sub(today).Hours() / 24)

	var rel string
	switch {
	case days > 0:
		rel = fmt.Sprintf("%dd remaining", days)
	case days == 0:
		rel = "today"
	default:
		rel = fmt.Sprintf("%dd overdue", -days)
	}
	return fmt.Sprintf("%s (%s)", due, rel)
}

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

	detail := m.renderDetail()

	left := todosPanel.Render(list.String())
	right := sidePanel.Render(detail)

	// Match heights so the separator spans the taller half.
	h := lipgloss.Height(left)
	if rh := lipgloss.Height(right); rh > h {
		h = rh
	}

	left = todosPanel.Height(h).Render(list.String())
	right = sidePanel.Height(h).Render(detail)
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
	icon := "○"
	if t.Done {
		icon = "✓"
	}

	title, meta := splitDescription(t.Description)
	if title == "" {
		title = t.Description
	}

	hasPriority := t.Priority >= 'A' && t.Priority <= 'Z'

	// The selected row is highlighted as a whole, so it drops the per-segment
	// coloring and renders one plain line under the selection style.
	if i == m.normalState.cursorPosition {
		parts := []string{icon}
		if hasPriority {
			parts = append(parts, string(t.Priority))
		} else {
			parts = append(parts, " ")
		}
		parts = append(parts, title)
		if meta != "" {
			parts = append(parts, meta)
		}
		return m.styles.Selected.Render(strings.Join(parts, " "))
	}

	// Everything but the title is faint (border-colored); the priority letter
	// is the one exception, color-coded by urgency.
	parts := []string{m.styles.RowIcon.Render(icon)}
	if hasPriority {
		idx := int(t.Priority-'A')
		if idx >= len(m.styles.Priority) {
			idx = len(m.styles.Priority) - 1
		}
		parts = append(parts, m.styles.Priority[idx].Render(string(t.Priority)))
	} else {
		parts = append(parts, " ")
	}

	if t.Done {
		parts = append(parts, m.styles.Done.Render(title))
	} else {
		parts = append(parts, title)
	}
	if meta != "" {
		parts = append(parts, m.styles.RowMeta.Render(meta))
	}
	return strings.Join(parts, " ")
}
