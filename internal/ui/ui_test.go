package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flcp/heute-todo/internal/config"
	"github.com/flcp/heute-todo/internal/todotxt"
)

func testModel(n int) Model {
	todos := make([]todotxt.Todo, n)
	for i := range todos {
		todos[i] = todotxt.Todo{Description: string(rune('a' + i))}
	}
	return Model{todos: todos, editState: editState{input: newInput()}, search: searchState{input: newSearchInput()}}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func enter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

func esc() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEsc} }

func tab() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyTab} }

func slash() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")} }

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

func TestAutosaveWritesFileOnAdd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	m := testModel(0)
	m.path = path

	m = step(m, key("o"))
	m = step(m, key("write the report"))

	_, cmd := m.Update(enter())
	if cmd == nil {
		t.Fatal("expected a save command after commit")
	}
	msg := cmd()
	saved, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("expected savedMsg, got %T", msg)
	}
	if saved.err != nil {
		t.Fatalf("save failed: %v", saved.err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "write the report") {
		t.Fatalf("file = %q, want it to contain the new todo", string(data))
	}
}

func TestEditWithAAppendsAtTitleEnd(t *testing.T) {
	m := testModel(1) // "a"

	m = step(m, key("a")) // edit: cursor jumps to the end of the title
	if m.editState.field != fieldTitle {
		t.Fatalf("field = %d, want fieldTitle", m.editState.field)
	}
	m = step(m, key("X")) // append at title end -> "aX"
	m = step(m, enter())

	if m.todos[0].Description != "aX" {
		t.Fatalf("desc = %q, want \"aX\"", m.todos[0].Description)
	}
}

func TestEditWithAAppendsAfterMultiWordTitle(t *testing.T) {
	m := testModel(1)
	todo, _ := todotxt.Parse("(A) Buy milk +errands @home")
	m.todos[0] = todo

	m = step(m, key("a")) // cursor at the end of "Buy milk", before +errands
	if pos, _ := locateField(m.editState.input.Value(), fieldTitle); m.editState.input.Position() <= pos {
		t.Fatalf("cursor = %d, want past the title start %d", m.editState.input.Position(), pos)
	}
	m = step(m, key("!")) // -> "(A) Buy milk! +errands @home"
	m = step(m, enter())

	if m.todos[0].Description != "Buy milk! +errands @home" {
		t.Fatalf("desc = %q, want \"Buy milk! +errands @home\"", m.todos[0].Description)
	}
}

func TestEditWithIStartsAtTitle(t *testing.T) {
	m := testModel(1) // "a"

	m = step(m, key("i")) // edit: cursor jumps to the title field
	m = step(m, key("X")) // insert at title start -> "Xa"
	m = step(m, enter())

	if m.todos[0].Description != "Xa" {
		t.Fatalf("desc = %q, want \"Xa\"", m.todos[0].Description)
	}
}

func TestToggleDoneMarksAndUnmarks(t *testing.T) {
	m := testModel(1) // "a", not done

	next, cmd := m.Update(key(" "))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("toggling should trigger an autosave")
	}
	if !m.todos[0].Done || m.todos[0].CompletedAt == nil {
		t.Fatalf("space should mark done with a completion date: %+v", m.todos[0])
	}

	m = step(m, key(" "))
	if m.todos[0].Done || m.todos[0].CompletedAt != nil {
		t.Fatalf("space again should clear done and date: %+v", m.todos[0])
	}
}

func TestDeleteWithDDRemovesSelected(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("d"))
	if !m.normalState.pendingDelete {
		t.Fatal("first d should arm pending delete")
	}
	if len(m.todos) != 2 {
		t.Fatal("first d must not delete yet")
	}

	next, cmd := m.Update(key("d"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("confirming delete should trigger an autosave")
	}
	if m.normalState.pendingDelete {
		t.Fatal("pending delete should clear after confirm")
	}
	if len(m.todos) != 1 || m.todos[0].Description != "b" {
		t.Fatalf("after dd, todos = %+v, want [b]", m.todos)
	}
}

func TestDeletePendingCancelledByOtherKey(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("d"))
	m = step(m, key("j")) // any other key cancels

	if m.normalState.pendingDelete {
		t.Fatal("another key should cancel pending delete")
	}
	if len(m.todos) != 2 {
		t.Fatalf("nothing should be deleted, got len %d", len(m.todos))
	}
}

func TestDeleteLastItemClampsCursor(t *testing.T) {
	m := testModel(2)     // a, b
	m = step(m, key("j")) // select b (index 1)
	m = step(m, key("d"))
	m = step(m, key("d"))

	if len(m.todos) != 1 || m.todos[0].Description != "a" {
		t.Fatalf("after dd, todos = %+v, want [a]", m.todos)
	}
	if m.normalState.cursorPosition != 0 {
		t.Fatalf("cursor = %d, want clamped to 0", m.normalState.cursorPosition)
	}
}

func TestLocateFieldValueOffsets(t *testing.T) {
	// (A) Buy milk +errands @home due:2026-10-10
	//  ^1 ^4       ^13/14   ^22/23 ^28    ^32
	const line = "(A) Buy milk +errands @home due:2026-10-10"
	cases := []struct {
		f   editField
		pos int
	}{
		{fieldPriority, 1}, // the letter
		{fieldTitle, 4},    // first title word
		{fieldProject, 14}, // after '+'
		{fieldContext, 23}, // after '@'
		{fieldDue, 32},     // after "due:"
	}
	for _, c := range cases {
		pos, ok := locateField(line, c.f)
		if !ok || pos != c.pos {
			t.Errorf("locateField(%d) = (%d,%v), want (%d,true)", c.f, pos, ok, c.pos)
		}
	}
}

func TestEnsureFieldScaffolds(t *testing.T) {
	if v, pos := ensureField("Buy milk", fieldDue); v != "Buy milk due:" || pos != len([]rune(v)) {
		t.Errorf("due scaffold = %q pos %d, want \"Buy milk due:\" pos %d", v, pos, len([]rune(v)))
	}
	if v, pos := ensureField("Buy milk", fieldPriority); v != "() Buy milk" || pos != 1 {
		t.Errorf("priority scaffold = %q pos %d, want \"() Buy milk\" pos 1", v, pos)
	}
	if v, pos := ensureField("", fieldProject); v != "+" || pos != 1 {
		t.Errorf("project scaffold on empty = %q pos %d, want \"+\" pos 1", v, pos)
	}
	// A present field is left untouched.
	if v, _ := ensureField("Buy milk due:2026-10-10", fieldDue); v != "Buy milk due:2026-10-10" {
		t.Errorf("present due mutated = %q", v)
	}
}

func TestStripEmptyField(t *testing.T) {
	if got := stripEmptyField("Buy milk due:", fieldDue); got != "Buy milk" {
		t.Errorf("strip empty due = %q, want \"Buy milk\"", got)
	}
	if got := stripEmptyField("Buy milk +", fieldProject); got != "Buy milk" {
		t.Errorf("strip empty project = %q, want \"Buy milk\"", got)
	}
	// A filled field is preserved.
	if got := stripEmptyField("Buy milk due:2026-10-10", fieldDue); got != "Buy milk due:2026-10-10" {
		t.Errorf("strip filled due = %q, want unchanged", got)
	}
}

func TestQuotedDetailsField(t *testing.T) {
	const line = `Buy milk details:"lorem ipsum"`

	pos, ok := locateField(line, fieldDetails)
	if !ok {
		t.Fatal("details should be present")
	}
	if want := strings.Index(line, `details:"`) + len(`details:"`); pos != want {
		t.Fatalf("details pos = %d, want %d (inside the quotes)", pos, want)
	}

	// A missing details field scaffolds an empty quoted tag with the cursor
	// between the quotes.
	v, cpos := ensureField("Buy milk", fieldDetails)
	if v != `Buy milk details:""` {
		t.Fatalf("details scaffold = %q, want %q", v, `Buy milk details:""`)
	}
	if cpos != len([]rune(v))-1 {
		t.Fatalf("cursor = %d, want %d (between the quotes)", cpos, len([]rune(v))-1)
	}

	// A filled quoted value with spaces survives an empty-scaffold strip.
	if got := stripEmptyField(line, fieldDetails); got != line {
		t.Errorf("strip filled details = %q, want unchanged", got)
	}
	// An empty quoted scaffold is dropped.
	if got := stripEmptyField(`Buy milk details:""`, fieldDetails); got != "Buy milk" {
		t.Errorf("strip empty details = %q, want \"Buy milk\"", got)
	}
}

func TestDetailsFieldRoundTrips(t *testing.T) {
	m := testModel(0)
	m = step(m, key("o"))
	m = step(m, key("Buy milk"))
	// Title -> Priority -> Project -> Context -> Due -> Details.
	for i := 0; i < 5; i++ {
		m = step(m, tab())
	}
	if m.editState.field != fieldDetails {
		t.Fatalf("field = %d, want fieldDetails", m.editState.field)
	}
	m = step(m, key("lorem ipsum")) // multi-word value, typed between the quotes
	m = step(m, enter())

	if len(m.todos) != 1 {
		t.Fatalf("len = %d, want 1", len(m.todos))
	}
	if got := m.todos[0].Tags["details"]; got != "lorem ipsum" {
		t.Fatalf("details tag = %q, want \"lorem ipsum\"", got)
	}
	if !strings.Contains(m.todos[0].Description, `details:"lorem ipsum"`) {
		t.Fatalf("description = %q, want it to contain the quoted details tag", m.todos[0].Description)
	}
}

func TestTabCyclesFieldsAndJumpsCursor(t *testing.T) {
	m := testModel(1)
	todo, _ := todotxt.Parse("(A) Buy milk +errands @home due:2026-10-10")
	m.todos[0] = todo

	m = step(m, key("a"))
	if m.editState.field != fieldTitle {
		t.Fatalf("edit should start on fieldTitle, got %d", m.editState.field)
	}
	for _, want := range []editField{fieldPriority, fieldProject, fieldContext, fieldDue} {
		m = step(m, tab())
		if m.editState.field != want {
			t.Fatalf("field = %d, want %d", m.editState.field, want)
		}
	}
	if pos, _ := locateField(m.editState.input.Value(), fieldDue); m.editState.input.Position() != pos {
		t.Fatalf("cursor = %d, want due value at %d", m.editState.input.Position(), pos)
	}
	m = step(m, tab()) // due -> details
	if m.editState.field != fieldDetails {
		t.Fatalf("field = %d, want fieldDetails", m.editState.field)
	}
	m = step(m, tab()) // details -> wraps back to the title
	if m.editState.field != fieldTitle {
		t.Fatalf("field = %d, want wrap to fieldTitle", m.editState.field)
	}
}

func TestTabDropsEmptyScaffoldOnLeave(t *testing.T) {
	m := testModel(1)
	todo, _ := todotxt.Parse("Buy milk")
	m.todos[0] = todo

	m = step(m, key("a"))
	m = step(m, tab()) // priority ()
	m = step(m, tab()) // project +
	m = step(m, tab()) // context @
	m = step(m, tab()) // due due:
	if !strings.Contains(m.editState.input.Value(), "due:") {
		t.Fatalf("expected due scaffold, got %q", m.editState.input.Value())
	}
	m = step(m, tab()) // leaving the empty due removes it
	if strings.Contains(m.editState.input.Value(), "due:") {
		t.Fatalf("empty due should be removed, got %q", m.editState.input.Value())
	}
}

func TestCommitDropsEmptyScaffold(t *testing.T) {
	m := testModel(0)
	m = step(m, key("o"))
	m = step(m, key("Buy milk"))
	m = step(m, tab()) // priority
	m = step(m, tab()) // project
	m = step(m, tab()) // context
	m = step(m, tab()) // due (empty scaffold)
	m = step(m, enter())

	if len(m.todos) != 1 {
		t.Fatalf("len = %d, want 1", len(m.todos))
	}
	if m.todos[0].Description != "Buy milk" || m.todos[0].Priority != 0 {
		t.Fatalf("todo = %+v, want \"Buy milk\" with no priority and no empty tags", m.todos[0])
	}
}

func filterModel(lines ...string) Model {
	todos := make([]todotxt.Todo, len(lines))
	for i, l := range lines {
		todos[i], _ = todotxt.Parse(l)
	}
	return Model{todos: todos, editState: editState{input: newInput()}, search: searchState{input: newSearchInput()}}
}

func TestFilterDeselectProjectHidesMatching(t *testing.T) {
	m := filterModel("call bob +work", "buy milk +home", "just a task")
	if m.visibleCount() != 3 {
		t.Fatalf("initial visible = %d, want 3 (all selected)", m.visibleCount())
	}
	m.filter.offProjects = map[string]bool{"work": true}
	if m.visibleCount() != 2 {
		t.Fatalf("visible = %d, want 2 after hiding +work", m.visibleCount())
	}
	for _, i := range m.displayIndices() {
		if strings.Contains(m.todos[i].Description, "+work") {
			t.Fatalf("+work todo should be hidden, got %q", m.todos[i].Description)
		}
	}
}

func TestFilterNoneRowHidesUntagged(t *testing.T) {
	m := filterModel("call bob +work", "just a task")
	m.filter.offProjects = map[string]bool{noneKey: true}
	if m.visibleCount() != 1 {
		t.Fatalf("visible = %d, want 1 after hiding untagged", m.visibleCount())
	}
	if got := m.todos[m.displayIndices()[0]].Description; got != "call bob +work" {
		t.Fatalf("remaining = %q, want the +work todo", got)
	}
}

func TestFilterAndAcrossDimensions(t *testing.T) {
	m := filterModel("a +work @home", "b +work @office")
	m.filter.offContexts = map[string]bool{"home": true}
	if m.visibleCount() != 1 {
		t.Fatalf("visible = %d, want 1 (context filter ANDs)", m.visibleCount())
	}
	if !strings.Contains(m.todos[m.displayIndices()[0]].Description, "@office") {
		t.Fatal("only the @office todo should survive")
	}
}

func TestFilterFocusCyclesWithTab(t *testing.T) {
	m := filterModel("a +work @home")
	for _, want := range []filterFocus{focusProjects, focusContexts, focusList} {
		m = step(m, tab())
		if m.filter.focus != want {
			t.Fatalf("focus = %d, want %d", m.filter.focus, want)
		}
	}
	m = step(m, tab()) // list -> projects
	m = step(m, esc())
	if m.filter.focus != focusList {
		t.Fatalf("esc should return to the list, got focus %d", m.filter.focus)
	}
}

func TestFilterToggleViaSpace(t *testing.T) {
	m := filterModel("a +work", "b +home", "c")
	m = step(m, tab()) // focus projects; keys = ["", "home", "work"]
	m = step(m, key("j"))
	m = step(m, key("j")) // cursor on "work"
	m = step(m, key(" ")) // deselect it
	if !m.filter.offProjects["work"] {
		t.Fatal("space should deselect the work row")
	}
	if m.visibleCount() != 2 {
		t.Fatalf("visible = %d, want 2 after hiding +work", m.visibleCount())
	}
	m = step(m, key(" ")) // re-select
	if m.filter.offProjects["work"] {
		t.Fatal("space should re-select the work row")
	}
	if m.visibleCount() != 3 {
		t.Fatalf("visible = %d, want 3 after re-selecting", m.visibleCount())
	}
}

func TestFilterToggleClampsListCursor(t *testing.T) {
	m := filterModel("a +work", "b +work", "c +work")
	m.normalState.cursorPosition = 2 // last row
	m = step(m, tab())               // focus projects; keys = ["", "work"]
	m = step(m, key("j"))            // cursor on "work"
	m = step(m, key(" "))            // hide all +work todos
	if m.visibleCount() != 0 {
		t.Fatalf("visible = %d, want 0", m.visibleCount())
	}
	if m.normalState.cursorPosition != 0 {
		t.Fatalf("list cursor = %d, want clamped to 0", m.normalState.cursorPosition)
	}
}

func TestFilterSoloSelectsOnlyCursorRow(t *testing.T) {
	m := filterModel("a +work", "b +home", "c +errands")
	m = step(m, tab())    // focus projects; keys = ["", "errands", "home", "work"]
	m = step(m, key("j")) // cursor on "errands"
	m = step(m, key("!")) // only errands
	for _, k := range []string{noneKey, "home", "work"} {
		if !m.filter.offProjects[k] {
			t.Fatalf("%q should be deselected after solo", k)
		}
	}
	if m.filter.offProjects["errands"] {
		t.Fatal("errands should stay selected after solo")
	}
	if m.visibleCount() != 1 {
		t.Fatalf("visible = %d, want 1 (only +errands)", m.visibleCount())
	}
	if !strings.Contains(m.todos[m.displayIndices()[0]].Description, "+errands") {
		t.Fatal("the only visible todo should be +errands")
	}
}

func TestFilterSoloTogglesBackToAll(t *testing.T) {
	m := filterModel("a +work", "b +home", "c +errands")
	m = step(m, tab())    // focus projects; keys = ["", "errands", "home", "work"]
	m = step(m, key("j")) // cursor on "errands"
	m = step(m, key("!")) // solo: only errands
	if m.visibleCount() != 1 {
		t.Fatalf("visible = %d, want 1 after solo", m.visibleCount())
	}
	m = step(m, key("!")) // errands is sole selection, so reselect all
	if len(m.filter.offProjects) != 0 {
		t.Fatalf("offProjects = %v, want empty after toggling back", m.filter.offProjects)
	}
	if m.visibleCount() != 3 {
		t.Fatalf("visible = %d, want 3 after toggling back to all", m.visibleCount())
	}
}

func TestDigitToPriorityMiddleOfWindow(t *testing.T) {
	// Digit 0's window is A–C, so it maps to the middle letter B (not A).
	if got := digitToPriority(0); got != 'B' {
		t.Fatalf("digitToPriority(0) = %q, want 'B'", got)
	}
	// Every digit must round-trip back through priorityToDigit.
	for d := 0; d <= 9; d++ {
		if got := priorityToDigit(digitToPriority(d)); got != d {
			t.Errorf("priorityToDigit(digitToPriority(%d)) = %d, want %d", d, got, d)
		}
	}
}

func TestDigitKeySetsPriority(t *testing.T) {
	m := filterModel("write the report")
	next, cmd := m.Update(key("0"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("setting priority should trigger an autosave")
	}
	if m.todos[0].Priority != 'B' {
		t.Fatalf("priority = %q, want 'B' (middle of digit 0's window)", m.todos[0].Priority)
	}
	// A higher digit maps to a later letter.
	m = step(m, key("9"))
	if m.todos[0].Priority != digitToPriority(9) {
		t.Fatalf("priority = %q, want %q", m.todos[0].Priority, digitToPriority(9))
	}
}

func TestShiftJKAdjustsPriorityInPrioritySort(t *testing.T) {
	m := filterModel("(G) write the report") // starts at digit 2
	m.sort = sortPriority
	if got := priorityToDigit(m.todos[0].Priority); got != 2 {
		t.Fatalf("initial digit = %d, want 2 (letter G)", got)
	}

	// J moves the item down the list -> lower priority -> digit goes up: 2 -> 3.
	next, cmd := m.Update(key("J"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("adjusting priority should trigger an autosave")
	}
	if got := priorityToDigit(m.todos[0].Priority); got != 3 {
		t.Fatalf("after J digit = %d, want 3", got)
	}

	// K moves the item up the list -> higher priority -> digit goes down: 3 -> 2 (G).
	m = step(m, key("K"))
	if m.todos[0].Priority != 'G' {
		t.Fatalf("after K priority = %q, want 'G' (digit 2)", m.todos[0].Priority)
	}
}

func TestShiftJKPriorityClampsAtEnds(t *testing.T) {
	m := filterModel("(G) write the report")
	m.sort = sortPriority

	// K raises priority repeatedly; the digit cannot drop below 0.
	for i := 0; i < 5; i++ {
		m = step(m, key("K"))
	}
	if got := priorityToDigit(m.todos[0].Priority); got != 0 {
		t.Fatalf("after repeated K digit = %d, want clamped 0", got)
	}

	// J lowers priority repeatedly; the digit cannot rise above 9.
	for i := 0; i < 15; i++ {
		m = step(m, key("J"))
	}
	if got := priorityToDigit(m.todos[0].Priority); got != 9 {
		t.Fatalf("after repeated J digit = %d, want clamped 9", got)
	}
}

func TestShiftJKPriorityIgnoredInFreeSort(t *testing.T) {
	// In free sort, J/K reorder rather than change priority.
	m := filterModel("(G) first", "second")
	m.sort = sortFree
	m = step(m, key("J")) // move the selected task down
	if m.todos[0].Description != "second" || m.todos[1].Description != "first" {
		t.Fatalf("free-sort J should reorder, got %q then %q",
			m.todos[0].Description, m.todos[1].Description)
	}
}

func TestToggleHideDone(t *testing.T) {
	m := filterModel("open one", "x done two", "open three")
	if m.visibleCount() != 3 {
		t.Fatalf("initial visible = %d, want 3", m.visibleCount())
	}
	m = step(m, key("z")) // hide done
	if !m.filter.hideDone {
		t.Fatal("z should enable hideDone")
	}
	if m.visibleCount() != 2 {
		t.Fatalf("visible = %d, want 2 with done hidden", m.visibleCount())
	}
	for _, i := range m.displayIndices() {
		if m.todos[i].Done {
			t.Fatal("no done task should be visible")
		}
	}
	m = step(m, key("z")) // show again
	if m.filter.hideDone || m.visibleCount() != 3 {
		t.Fatalf("z again should show done: hideDone=%v visible=%d", m.filter.hideDone, m.visibleCount())
	}
}

func TestMoveInFileModeSkipsHiddenTasks(t *testing.T) {
	// File-order (free) sort with a done task hidden between two open tasks:
	// pressing J on the first open task must move it below the other open task,
	// not swap it with the hidden done task sitting next in file order.
	m := filterModel("open one", "x done two", "open three")
	m = step(m, key("z")) // hide done
	if got := m.visibleCount(); got != 2 {
		t.Fatalf("visible = %d, want 2", got)
	}

	m = step(m, key("J")) // move "open one" down past "open three"

	got := []string{}
	for _, i := range m.displayIndices() {
		got = append(got, m.todos[i].Description)
	}
	want := []string{"open three", "open one"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("visible order = %v, want %v", got, want)
	}
	if src := m.cursorSourceIndex(); m.todos[src].Description != "open one" {
		t.Fatalf("cursor on %q, want cursor to follow \"open one\"", m.todos[src].Description)
	}
}

func TestListScrollsToFitHeight(t *testing.T) {
	todos := make([]todotxt.Todo, 50)
	for i := range todos {
		todos[i], _ = todotxt.Parse(fmt.Sprintf("task-%02d", i))
	}
	m := Model{todos: todos, editState: editState{input: newInput()}, search: searchState{input: newSearchInput()}}
	m.width = 80
	m.height = 20

	// With the cursor at the top the whole view must stay within the terminal
	// height rather than overflowing off screen.
	out := m.View()
	if got := lipgloss.Height(out); got > m.height {
		t.Fatalf("view height = %d, want <= %d", got, m.height)
	}
	if strings.Contains(out, "task-49") {
		t.Fatal("task-49 should be scrolled out of view with cursor at top")
	}

	// Move the cursor to the last task; the list scrolls so it stays visible and
	// the view still fits.
	m.normalState.cursorPosition = len(todos) - 1
	out = m.View()
	if got := lipgloss.Height(out); got > m.height {
		t.Fatalf("view height with cursor at bottom = %d, want <= %d", got, m.height)
	}
	if !strings.Contains(out, "task-49") {
		t.Fatal("selected task-49 should be visible after scrolling to the bottom")
	}
	if strings.Contains(out, "task-00") {
		t.Fatal("task-00 should be scrolled out of view with cursor at bottom")
	}
}

func TestDetailClampsToFitHeight(t *testing.T) {
	// A single task whose Details value wraps into many lines makes the detail
	// panel the tallest half. It must clamp instead of pushing the header off.
	long := strings.Repeat("word ", 60)
	todo, _ := todotxt.Parse(`Buy milk +errands @home due:2026-10-10 details:"` + long + `"`)
	m := Model{todos: []todotxt.Todo{todo}, editState: editState{input: newInput()}, search: searchState{input: newSearchInput()}}
	m.width = 80
	m.height = 16

	out := m.View()
	if got := lipgloss.Height(out); got > m.height {
		t.Fatalf("view height = %d, want <= %d", got, m.height)
	}
	if !strings.Contains(out, "Buy milk") {
		t.Fatal("title should stay visible when the detail is clamped")
	}

	// In the extreme (a single row of space) only the title survives, dropping
	// every field line below it.
	body := m.renderBody(1)
	if !strings.Contains(body, "Buy milk") {
		t.Fatal("title should survive even at one row")
	}
	if strings.Contains(body, "Priority") {
		t.Fatal("field lines should be dropped from the bottom at one row")
	}
}

func TestClampLines(t *testing.T) {
	s := "a\nb\nc\nd"
	if got := clampLines(s, 2); got != "a\nb" {
		t.Fatalf("clampLines(_, 2) = %q, want \"a\\nb\"", got)
	}
	if got := clampLines(s, 10); got != s {
		t.Fatalf("clampLines(_, 10) = %q, want unchanged", got)
	}
	if got := clampLines(s, 0); got != s {
		t.Fatalf("clampLines(_, 0) = %q, want unchanged", got)
	}
}

func TestWithConfigAppliesPreferences(t *testing.T) {
	m := testModel(0).WithConfig(config.Config{
		Path: "/tmp/x.txt", Sort: "priority", ShowDone: false, Theme: "default",
	})
	if m.sort != sortPriority {
		t.Errorf("sort = %d, want sortPriority", m.sort)
	}
	if !m.filter.hideDone {
		t.Error("ShowDone:false should set hideDone")
	}
	if m.theme != "default" {
		t.Errorf("theme = %q, want default", m.theme)
	}
	// toConfig should round-trip the applied preferences.
	got := m.toConfig()
	if got.Sort != "priority" || got.ShowDone != false || got.Theme != "default" || got.Path != "/tmp/x.txt" {
		t.Fatalf("toConfig = %+v", got)
	}
}

func ctrlR() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlR} }

func descriptions(todos []todotxt.Todo) []string {
	out := make([]string, len(todos))
	for i, t := range todos {
		out[i] = t.Description
	}
	return out
}

func TestUndoRestoresStateBeforeDelete(t *testing.T) {
	m := testModel(3) // a, b, c

	m = step(m, key("d"))
	m = step(m, key("d")) // delete a
	if got := descriptions(m.todos); len(got) != 2 || got[0] != "b" {
		t.Fatalf("after delete, todos = %v, want [b c]", got)
	}

	next, cmd := m.Update(key("u"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("undo should trigger an autosave")
	}
	if got := descriptions(m.todos); len(got) != 3 || got[0] != "a" {
		t.Fatalf("after undo, todos = %v, want [a b c]", got)
	}
}

func TestUndoMultipleTraversesHistory(t *testing.T) {
	m := testModel(1) // a

	m = step(m, key(" ")) // toggle done on a
	m = step(m, key("d")) // delete
	m = step(m, key("d"))
	if len(m.todos) != 0 {
		t.Fatalf("after delete, todos = %v, want empty", descriptions(m.todos))
	}

	m = step(m, key("u")) // undo delete -> [a done]
	if len(m.todos) != 1 || !m.todos[0].Done {
		t.Fatalf("after first undo, todos = %+v, want [a done]", m.todos)
	}

	m = step(m, key("u")) // undo toggle -> [a not done]
	if len(m.todos) != 1 || m.todos[0].Done {
		t.Fatalf("after second undo, todos = %+v, want [a not done]", m.todos)
	}
}

func TestRedoReappliesUndoneChange(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("d"))
	m = step(m, key("d")) // delete a -> [b]
	m = step(m, key("u")) // undo -> [a b]
	if got := descriptions(m.todos); len(got) != 2 {
		t.Fatalf("after undo, todos = %v, want [a b]", got)
	}

	next, cmd := m.Update(ctrlR()) // redo -> [b]
	m = next.(Model)
	if cmd == nil {
		t.Fatal("redo should trigger an autosave")
	}
	if got := descriptions(m.todos); len(got) != 1 || got[0] != "b" {
		t.Fatalf("after redo, todos = %v, want [b]", got)
	}
}

func TestChangeAfterUndoClearsRedo(t *testing.T) {
	m := testModel(2) // a, b

	m = step(m, key("d"))
	m = step(m, key("d")) // delete a -> [b]
	m = step(m, key("u")) // undo -> [a b], redo now holds [b]
	m = step(m, key(" ")) // fresh change: toggle a -> clears redo
	m = step(m, ctrlR())  // redo should be a no-op now
	if got := descriptions(m.todos); len(got) != 2 {
		t.Fatalf("redo after a fresh change should do nothing, todos = %v", got)
	}
	if !m.todos[0].Done {
		t.Fatal("the fresh toggle change should still stand after a no-op redo")
	}
}

func TestUndoWithNoHistoryIsNoop(t *testing.T) {
	m := testModel(2)
	next, cmd := m.Update(key("u"))
	m = next.(Model)
	if cmd != nil {
		t.Fatal("undo with empty history should not autosave")
	}
	if len(m.todos) != 2 {
		t.Fatalf("todos should be unchanged, got %v", descriptions(m.todos))
	}
}

// typeStr feeds s to the model one rune at a time as key messages.
func typeStr(m Model, s string) Model {
	for _, r := range s {
		m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestMatchesSearchFragments(t *testing.T) {
	todo, _ := todotxt.Parse("test one two")
	cases := []struct {
		query string
		want  bool
	}{
		{"", true},
		{"test", true},
		{"te two", true},    // each fragment is searched independently
		{"two te", true},    // order does not matter
		{"TEST", true},      // case-insensitive
		{"te three", false}, // every fragment must match
		{"xyz", false},
	}
	for _, c := range cases {
		if got := matchesSearch(todo, c.query); got != c.want {
			t.Errorf("matchesSearch(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}

func TestSearchSpansEveryField(t *testing.T) {
	todo, _ := todotxt.Parse("(A) call bob +work @office due:2026-09-20")
	for _, frag := range []string{"call", "work", "office", "2026-09-20", "A"} {
		if !matchesSearch(todo, frag) {
			t.Errorf("fragment %q should match somewhere in the serialized line", frag)
		}
	}
}

func TestSearchLiveFiltersList(t *testing.T) {
	m := filterModel("buy milk", "call bob", "buy bread")

	m = step(m, slash())
	if !m.search.typing {
		t.Fatal("/ should start search typing")
	}
	m = typeStr(m, "buy")
	if got := m.visibleCount(); got != 2 {
		t.Fatalf("visible after 'buy' = %d, want 2", got)
	}
	m = typeStr(m, " bread")
	if got := m.visibleCount(); got != 1 {
		t.Fatalf("visible after 'buy bread' = %d, want 1", got)
	}
}

func TestSearchEnterKeepsQueryAndReturnsToList(t *testing.T) {
	m := filterModel("buy milk", "call bob", "buy bread")

	m = step(m, slash())
	m = typeStr(m, "call")
	m = step(m, enter())
	if m.search.typing {
		t.Fatal("enter should stop typing")
	}
	if !m.searchActive() {
		t.Fatal("query should stay active after enter")
	}
	if got := m.visibleCount(); got != 1 {
		t.Fatalf("visible after enter = %d, want 1", got)
	}
	// Normal navigation keys should now reach the list, not the search bar.
	m = step(m, key("j"))
}

func TestSearchEscClearsButLeavesFilters(t *testing.T) {
	m := filterModel("buy milk +home", "call bob +work")

	// Deselect the +work project so a filter is already active.
	m.filter.offProjects = map[string]bool{"work": true}
	if got := m.visibleCount(); got != 1 {
		t.Fatalf("visible with work filtered = %d, want 1", got)
	}

	m = step(m, slash())
	m = typeStr(m, "zzz")
	if got := m.visibleCount(); got != 0 {
		t.Fatalf("visible with non-matching search = %d, want 0", got)
	}
	m = step(m, esc())
	if m.searchActive() {
		t.Fatal("esc should clear the query")
	}
	// The project filter must survive the search being cleared.
	if !m.filter.offProjects["work"] {
		t.Fatal("esc on search must not touch project filters")
	}
	if got := m.visibleCount(); got != 1 {
		t.Fatalf("visible after clearing search = %d, want 1 (filter still active)", got)
	}
}

func TestSearchReopenPrefillsPreviousQuery(t *testing.T) {
	m := filterModel("buy milk", "call bob")

	m = step(m, slash())
	m = typeStr(m, "buy")
	m = step(m, enter()) // query frozen, cursor back to list

	m = step(m, slash()) // reopen
	if got := m.search.input.Value(); got != "buy" {
		t.Fatalf("reopened search value = %q, want \"buy\"", got)
	}
	if got := m.search.input.Position(); got != len("buy") {
		t.Fatalf("cursor position = %d, want %d (end of text)", got, len("buy"))
	}
}
