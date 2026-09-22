package todotxt

import (
	"testing"
	"time"
)

func date(t *testing.T, s string) *time.Time {
	t.Helper()
	d, err := time.Parse(DateLayout, s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return &d
}

func TestParseIncompleteTaskWithAllDetails(t *testing.T) {
	line := "(A) 2026-09-17 Write tests +heute @work due:2026-09-20"
	got, ok := Parse(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Done {
		t.Error("should not be done")
	}
	if got.Priority != 'A' {
		t.Errorf("priority = %q, want A", got.Priority)
	}
	if got.CreatedAt == nil || !got.CreatedAt.Equal(*date(t, "2026-09-17")) {
		t.Errorf("createdAt = %v", got.CreatedAt)
	}
	if got.Description != "Write tests +heute @work due:2026-09-20" {
		t.Errorf("desc = %q", got.Description)
	}
	if len(got.Projects) != 1 || got.Projects[0] != "heute" {
		t.Errorf("projects = %v", got.Projects)
	}
	if len(got.Contexts) != 1 || got.Contexts[0] != "work" {
		t.Errorf("contexts = %v", got.Contexts)
	}
	if got.Tags["due"] != "2026-09-20" {
		t.Errorf("tags = %v", got.Tags)
	}
}

func TestParseQuotedDetailsTag(t *testing.T) {
	line := `Buy milk details:"lorem ipsum dolor" @home`
	got, ok := Parse(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Tags["details"] != "lorem ipsum dolor" {
		t.Errorf("details = %q, want %q", got.Tags["details"], "lorem ipsum dolor")
	}
	if len(got.Contexts) != 1 || got.Contexts[0] != "home" {
		t.Errorf("contexts = %v, want [home]", got.Contexts)
	}
	if got.String() != line {
		t.Errorf("round-trip = %q, want %q", got.String(), line)
	}
}

func TestParseCompletedTaskWithBothDates(t *testing.T) {
	line := "x 2026-09-18 2026-09-17 Do the thing"
	got, ok := Parse(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if !got.Done {
		t.Error("should be done")
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(*date(t, "2026-09-18")) {
		t.Errorf("completedAt = %v", got.CompletedAt)
	}
	if got.CreatedAt == nil || !got.CreatedAt.Equal(*date(t, "2026-09-17")) {
		t.Errorf("createdAt = %v", got.CreatedAt)
	}
	if got.Description != "Do the thing" {
		t.Errorf("desc = %q", got.Description)
	}
}

func TestParsePlainText(t *testing.T) {
	got, ok := Parse("just a task")
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Done || got.Priority != 0 || got.CreatedAt != nil {
		t.Errorf("unexpected fields: %+v", got)
	}
	if got.Description != "just a task" {
		t.Errorf("desc = %q", got.Description)
	}
}

func TestParseBlank(t *testing.T) {
	if _, ok := Parse("   "); ok {
		t.Error("blank line should not parse")
	}
	if _, ok := Parse(""); ok {
		t.Error("empty line should not parse")
	}
}

func TestParseAndToString(t *testing.T) {
	lines := []string{
		"(A) 2026-09-17 Write tests +heute @work due:2026-09-20",
		"x 2026-09-18 2026-09-17 Do the thing",
		"x 2026-09-18 Simple done",
		"just a task",
		"(B) prioritized no date",
	}
	for _, line := range lines {
		got, ok := Parse(line)
		if !ok {
			t.Fatalf("failed to parse %q", line)
		}
		if s := got.String(); s != line {
			t.Errorf("round trip:\n got %q\nwant %q", s, line)
		}
	}
}

func TestParseList(t *testing.T) {
	body := "task one\n\n(A) task two\n"
	todos := ToTodoList(body)
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(todos))
	}
	out := ToStringList(todos)
	want := "task one\n(A) task two\n"
	if out != want {
		t.Errorf("serialize = %q, want %q", out, want)
	}
}

func TestSerializeEmpty(t *testing.T) {
	if s := ToStringList(nil); s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
}

func TestToggled(t *testing.T) {
	now := *date(t, "2026-09-18")
	td := Todo{Description: "task"}

	done := td.Toggled(now)
	if !done.Done || done.CompletedAt == nil || !done.CompletedAt.Equal(now) {
		t.Fatalf("toggled = %+v, want done with completion date %v", done, now)
	}
	if td.Done {
		t.Error("Toggled must not mutate the receiver")
	}

	back := done.Toggled(now)
	if back.Done || back.CompletedAt != nil {
		t.Fatalf("toggled back = %+v, want not done and no completion date", back)
	}
}

func TestSortByPriority(t *testing.T) {
	todos := []Todo{
		{Done: true, Description: "done"},
		{Priority: 'C', Description: "c"},
		{Description: "none"},
		{Priority: 'A', Description: "a"},
	}
	sorted := SortByPriority(todos)
	wantOrder := []string{"a", "c", "none", "done"}
	for i, w := range wantOrder {
		if sorted[i].Description != w {
			t.Errorf("pos %d = %q, want %q", i, sorted[i].Description, w)
		}
	}
}
