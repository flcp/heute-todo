package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flcp/heute-todo/internal/config"
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

func tab() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyTab} }

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

func TestEditEntersAtTitleStart(t *testing.T) {
	m := testModel(1) // "a"

	m = step(m, key("a")) // edit: cursor jumps to the title field
	if m.editState.field != fieldTitle {
		t.Fatalf("field = %d, want fieldTitle", m.editState.field)
	}
	m = step(m, key("X")) // insert at title start -> "Xa"
	m = step(m, enter())

	if m.todos[0].Description != "Xa" {
		t.Fatalf("desc = %q, want \"Xa\"", m.todos[0].Description)
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
	return Model{todos: todos, editState: editState{input: newInput()}}
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
