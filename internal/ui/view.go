package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
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
	var modeContent string
	switch {
	case m.filter.focus != focusList:
		dim := "projects"
		if m.filter.focus == focusContexts {
			dim = "contexts"
		}
		modeContent = m.styles.Insert.Render("filter "+dim) + "  " +
			m.helpItem("j/k", "navigate") + "  " +
			m.helpItem("⎵ ", "toggle") + "  " +
			m.helpItem("!", "only") + "  " +
			m.helpItem("⇥", "next") + "  " +
			m.helpItem("esc", "back")
	case m.mode == modeInsert:
		label := "Add"
		if !m.editState.isAddingItem {
			label = "Edit"
		}
		hint := m.styles.Help.Render("  ·  ") + m.helpItem("⇥", "next field")
		modeContent = m.styles.Insert.Render(label) + " " + m.editState.input.View() + hint
	case m.normalState.pendingDelete:
		modeContent = m.styles.Delete.Render("delete? press d to confirm, esc to cancel")
	default:
		nav := []string{
			m.helpItem("j/k", "navigate"),
			m.helpItem("g", "top"),
			m.helpItem("G", "bottom"),
		}
		if m.sort == sortFree {
			nav = append(nav[:1], append([]string{m.helpItem("J/K", "move")}, nav[1:]...)...)
		}
		act := []string{
			m.helpItem("⎵ ", "done"),
			//m.helpItem("o/O", "add"),
			m.helpItem("o", "add"),
			// m.helpItem("i/I/a", "edit"),
			m.helpItem("a", "edit"),
			m.helpItem("d", "delete"),
		}
		misc := []string{
			m.helpItem("⇥", "filter"),
			m.helpItem("s", "sort"),
			m.helpItem("q", "quit"),
		}
		sep := m.styles.Help.Render("  ·  ")
		join := func(items []string) string { return strings.Join(items, "  ") }
		modeContent = join(nav) + sep + join(act) + sep + join(misc)
	}

	width := m.width
	if width <= 0 {
		width = 80
	}

	// Inner content area = panel Width(width-2) minus left/right padding (2).
	innerW := width - 4
	path := m.styles.PathInline.Render(truncateLeft(m.absPath(), innerW/3))
	pathW := lipgloss.Width(path)
	helpW := innerW - pathW
	if helpW < 0 {
		helpW = 0
	}
	content := lipgloss.NewStyle().Width(helpW).Render(modeContent) + path

	return m.styles.CommandLine.Width(width - 2).Render(content)
}

// renderDetail renders the side panel: a detail view of the todo currently
// under the cursor, or a hint when the list is empty. While editing, it renders
// the live edit buffer instead and highlights the field Tab currently targets.
func (m Model) renderDetail() string {
	inserting := m.mode == modeInsert
	var t todotxt.Todo
	if inserting {
		if parsed, ok := todotxt.Parse(m.editState.input.Value()); ok {
			t = parsed
		}
	} else {
		if len(m.todos) == 0 || m.normalState.cursorPosition >= len(m.todos) {
			return m.styles.Empty.Render("(no task selected)")
		}
		t = m.todos[m.cursorSourceIndex()]
	}

	active := func(f editField) bool { return inserting && m.editState.field == f }

	var b strings.Builder
	title := detailTitle(t)
	if active(fieldTitle) {
		b.WriteString(m.styles.Selected.Render(title))
	} else {
		b.WriteString(m.styles.DetailTitle.Render(title))
	}
	b.WriteString("\n\n")

	priority := " "
	if t.Priority >= 'A' && t.Priority <= 'Z' {
		priority = string(t.Priority)
	}
	b.WriteString(m.detailField("Priority", priority, active(fieldPriority)))

	projects := " "
	if len(t.Projects) > 0 {
		projects = strings.Join(t.Projects, ", ")
	}
	b.WriteString(m.detailField("Project", projects, active(fieldProject)))

	contexts := " "
	if len(t.Contexts) > 0 {
		contexts = strings.Join(t.Contexts, ", ")
	}
	b.WriteString(m.detailField("Context", contexts, active(fieldContext)))

	due := " "
	if d, ok := t.Tags["due"]; ok {
		due = formatDue(d, time.Now())
	}
	b.WriteString(m.detailField("Due date", due, active(fieldDue)))

	details := " "
	if d, ok := t.Tags["details"]; ok {
		details = d
	}
	b.WriteString(m.detailField("Details", details, active(fieldDetails)))

	if t.CreatedAt != nil {
		b.WriteString(m.styles.DetailLabel.Render("Created:") + " " + m.styles.DateCreated.Render(t.CreatedAt.Format(todotxt.DateLayout)) + "\n")
	}
	if t.CompletedAt != nil {
		b.WriteString(m.styles.DetailLabel.Render("Done:") + " " + m.styles.DateDone.Render(t.CompletedAt.Format(todotxt.DateLayout)) + "\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// detailField renders a single "Label: value" line, terminated by a newline.
// When active, the whole line is drawn in the inverse selection style to mark
// the field Tab currently targets.
func (m Model) detailField(label, value string, active bool) string {
	empty := strings.TrimSpace(value) == ""
	if active {
		text := label + ": " + value
		if empty {
			text = label + ": —"
		}
		return m.styles.Selected.Render(text) + "\n"
	}
	if empty {
		return m.styles.Empty.Render(label+": —") + "\n"
	}
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
	for _, tok := range todotxt.Fields(desc) {
		switch {
		case len(tok) > 1 && (tok[0] == '+' || tok[0] == '@'):
			metaWords = append(metaWords, tok)
		case isKeyValue(tok):
			metaWords = append(metaWords, tok)
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
	contentW := leftW - 2 // TodoPanel has PaddingRight(2)
	for dispIdx, srcIdx := range m.displayIndices() {
		if dispIdx > 0 {
			list.WriteByte('\n')
		}
		list.WriteString(m.renderRow(dispIdx, m.todos[srcIdx], contentW))
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

// renderHeader lays out the top bar: the ASCII art logo on the left and the
// open-task count on the right, with empty space between them.
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

	sortLabel := "file"
	switch m.sort {
	case sortPriority:
		sortLabel = "priority"
	case sortName:
		sortLabel = "name"
	}
	openText := fmt.Sprintf("%d open · %s", open, sortLabel)

	// logoW is the max line width of the ASCII art; +2 adds the horizontal padding.
	const logoW = 24
	leftW := logoW + 2
	rightW := lipgloss.Width(openText) + 2

	left := m.styles.HeaderLogo.Width(leftW).Render(m.renderLogo())
	right := m.styles.HeaderCount.Width(rightW).Render(openText)

	// The filter boxes match the header height: content height = total − border.
	contentH := lipgloss.Height(left) - 2
	if contentH < 1 {
		contentH = 1
	}
	projBox := m.renderFilterBox("projects", m.projectKeys(), m.filter.projectCursor,
		m.filter.offProjects, m.filter.focus == focusProjects, contentH)
	ctxBox := m.renderFilterBox("contexts", m.contextKeys(), m.filter.contextCursor,
		m.filter.offContexts, m.filter.focus == focusContexts, contentH)

	// Fill the gap between the filter boxes and the count so the count lands at
	// the top-right corner.
	used := lipgloss.Width(left) + lipgloss.Width(projBox) + lipgloss.Width(ctxBox) + lipgloss.Width(right)
	gapW := width - used
	if gapW < 0 {
		gapW = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, projBox, ctxBox, strings.Repeat(" ", gapW), right)
}

// renderFilterBox renders one header filter box: a dim title line followed by a
// checkbox row per key ("[x]" selected, "[ ]" deselected). When focused, the box
// gets an accent border and its cursor row is highlighted. Rows scroll to keep
// the cursor visible when they exceed contentH.
func (m Model) renderFilterBox(title string, keys []string, cursor int, off map[string]bool, focused bool, contentH int) string {
	innerW := lipgloss.Width(title)
	for _, k := range keys {
		if w := lipgloss.Width(filterMarkOff + " " + filterRowLabel(k)); w > innerW {
			innerW = w
		}
	}
	if innerW > 16 {
		innerW = 16
	}
	if innerW < 6 {
		innerW = 6
	}

	visible := contentH - 1 // the title takes one line
	if visible < 1 {
		visible = 1
	}
	start := 0
	if cursor >= visible {
		start = cursor - visible + 1
	}
	end := start + visible
	if end > len(keys) {
		end = len(keys)
	}

	var b strings.Builder
	b.WriteString(m.styles.FilterTitle.Render(truncateRight(title, innerW)))
	for i := start; i < end; i++ {
		b.WriteByte('\n')
		selected := !off[keys[i]]
		label := filterRowLabel(keys[i])
		if focused && i == cursor {
			// Highlighted row: draw the whole line in the inverse style so the
			// mark and label share the selection bar.
			mark := filterMarkOff
			if selected {
				mark = filterMarkOn
			}
			b.WriteString(m.styles.Selected.Width(innerW).Render(truncateRight(mark+" "+label, innerW)))
		} else {
			mark := m.styles.FilterMarkOff.Render(filterMarkOff)
			if selected {
				mark = m.styles.FilterMarkOn.Render(filterMarkOn)
			}
			content := mark + " " + truncateRight(label, innerW-2)
			b.WriteString(lipgloss.NewStyle().Width(innerW).Render(content))
		}
	}

	style := m.styles.FilterBox
	if focused {
		style = m.styles.FilterBoxActive
	}
	// Rows are already padded to innerW, so let the box size to its content;
	// setting an explicit Width here would fight the border/padding and wrap rows.
	return style.Height(contentH).Render(b.String())
}

// Filter checkbox marks: a filled circle for a selected row, a hollow one for a
// deselected row (deliberately not the ✓ used for done todos).
const (
	filterMarkOn  = "●"
	filterMarkOff = "○"
)

// filterRowLabel is the display text for a filter key: the "(none)" row reuses
// the "—" placeholder shown for empty detail fields; otherwise the tag name.
func filterRowLabel(key string) string {
	if key == noneKey {
		return "—"
	}
	return key
}

// renderLogo returns the ASCII art "heute" with a left-to-right gradient
// using the theme's own rainbow stops.
func (m Model) renderLogo() string {
	lines := [4]string{
		`   __            __     `,
		`  / /  ___ __ __/ /____ `,
		` / _ \/ -_) // / __/ -_)`,
		`/_//_/\__/\_,_/\__/\__/`,
	}
	stops := m.styles.LogoRainbow
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		for x, ch := range line {
			color := interpolateRainbow(stops, x, 23)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render(string(ch)))
		}
	}
	return b.String()
}

// interpolateRainbow returns a color from a gradient of stops at position x/maxX.
func interpolateRainbow(stops []lipgloss.Color, x, maxX int) lipgloss.Color {
	if len(stops) == 0 {
		return lipgloss.Color("#FFFFFF")
	}
	if maxX <= 0 || len(stops) == 1 {
		return stops[0]
	}
	t := float64(x) / float64(maxX)
	scaled := t * float64(len(stops)-1)
	lo := int(scaled)
	hi := lo + 1
	if hi >= len(stops) {
		return stops[len(stops)-1]
	}
	return blendColors(stops[lo], stops[hi], scaled-float64(lo))
}

// blendColors linearly interpolates between two hex colors by factor t (0..1).
func blendColors(a, b lipgloss.Color, t float64) lipgloss.Color {
	ra, ga, ba := parseHexColor(string(a))
	rb, gb, bb := parseHexColor(string(b))
	r := uint8(float64(ra)*(1-t) + float64(rb)*t + 0.5)
	g := uint8(float64(ga)*(1-t) + float64(gb)*t + 0.5)
	bv := uint8(float64(ba)*(1-t) + float64(bb)*t + 0.5)
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bv))
}

// parseHexColor parses a "#RRGGBB" string into its RGB components.
func parseHexColor(s string) (uint8, uint8, uint8) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 128, 128, 128
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 128, 128, 128
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v)
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

// truncateRight shortens s to at most max display columns, dropping characters
// from the end and appending an ellipsis.
func truncateRight(s string, max int) string {
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
	return string(r[:max-1]) + "…"
}

func (m Model) renderRow(i int, t todotxt.Todo, width int) string {
	title, meta := splitDescription(t.Description)
	if title == "" {
		title = t.Description
	}

	hasPriority := !t.Done && t.Priority >= 'A' && t.Priority <= 'Z'

	// 2 = priority/checkmark slot + space.
	available := width - 2
	titleW := lipgloss.Width(title)

	// Stage 1: truncate meta if the full row overflows.
	if meta != "" && titleW+1+lipgloss.Width(meta) > available {
		metaAvail := available - titleW - 1
		if metaAvail >= 2 {
			meta = truncateRight(meta, metaAvail)
		} else {
			meta = ""
		}
	}

	// Stage 2: truncate title if it still overflows (meta gone or title itself too wide).
	titleAvail := available
	if meta != "" {
		titleAvail -= 1 + lipgloss.Width(meta)
	}
	if titleW > titleAvail {
		title = truncateRight(title, titleAvail)
	}

	if i == m.normalState.cursorPosition {
		var parts []string
		if t.Done {
			parts = []string{"✓", title}
		} else {
			prioritySlot := " "
			if hasPriority {
				prioritySlot = strconv.Itoa(priorityToDigit(t.Priority))
			}
			parts = []string{prioritySlot, title}
		}
		if meta != "" {
			parts = append(parts, meta)
		}
		return m.styles.Selected.Render(strings.Join(parts, " "))
	}

	if t.Done {
		body := title
		if meta != "" {
			body += " " + meta
		}
		return m.styles.DoneIcon.Render("✓") + " " + m.styles.Done.Render(body)
	}

	var parts []string
	if hasPriority {
		digit := priorityToDigit(t.Priority)
		parts = []string{m.styles.Priority[digit].Render(strconv.Itoa(digit))}
	} else {
		parts = []string{" "}
	}
	parts = append(parts, title)
	if meta != "" {
		parts = append(parts, m.styles.RowMeta.Render(meta))
	}
	return strings.Join(parts, " ")
}

// helpItem renders a single shortcut entry: key in the dim HelpKey color,
// action text in the faint Help style.
func (m Model) helpItem(key, action string) string {
	return m.styles.HelpKey.Render(key) + m.styles.Help.Render(":"+action)
}

// priorityToDigit maps a priority byte ('A'..'Z') to a 0–9 display digit.
func priorityToDigit(p byte) int {
	return int(p-'A') * 10 / 26
}
