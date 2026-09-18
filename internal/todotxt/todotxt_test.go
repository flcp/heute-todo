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
