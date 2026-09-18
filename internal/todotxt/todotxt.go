// Package todotxt parses and serializes the todo.txt format.
//
// Format reference: https://github.com/todotxt/todo.txt
//
//	x (A) 2026-09-17 2026-09-01 Task text +project @context due:2026-09-20
//
// Ordering of a line:
//   - optional completion marker "x " (completed)
//   - optional priority "(A) " (only meaningful on incomplete tasks per spec,
//     but we preserve it wherever it appears at the start)
//   - optional dates: for completed tasks the first date is the completion date
//     and an optional second date is the creation date; for incomplete tasks a
//     single leading date is the creation date
//   - the description, which may contain +projects, @contexts and key:value tags
package todotxt

import (
	"sort"
	"strings"
	"time"
)

// DateLayout is the date format used by the todo.txt spec.
const DateLayout = "2006-01-02"

// Todo represents a single task line.
type Todo struct {
	Done        bool
	Priority    byte // 'A'..'Z', or 0 if none
	CompletedAt *time.Time
	CreatedAt   *time.Time
	Description string
	Projects    []string          // tokens starting with '+' (without the '+')
	Contexts    []string          // tokens starting with '@' (without the '@')
	Tags        map[string]string // key:value tags
}

// Parse parses a single todo.txt line into a Todo.
// It returns ok=false for empty/blank lines.
func Parse(line string) (Todo, bool) {
	trimmedLine := strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(trimmedLine) == "" {
		return Todo{}, false
	}
	trimmedLine = strings.TrimLeft(trimmedLine, " ")

	t := Todo{Tags: map[string]string{}}

	// Completion marker.
	if strings.HasPrefix(trimmedLine, "x ") {
		t.Done = true
		trimmedLine = strings.TrimLeft(trimmedLine[2:], " ")
	}

	// Priority.
	if len(trimmedLine) >= 3 && trimmedLine[0] == '(' && trimmedLine[2] == ')' &&
		trimmedLine[1] >= 'A' && trimmedLine[1] <= 'Z' &&
		(len(trimmedLine) == 3 || trimmedLine[3] == ' ') {
		t.Priority = trimmedLine[1]
		trimmedLine = strings.TrimLeft(trimmedLine[3:], " ")
	}

	// Dates.
	first, after, ok := leadingDate(trimmedLine)
	if ok {
		if t.Done {
			// First date is completion date; optional second is creation date.
			t.CompletedAt = first
			second, after2, ok2 := leadingDate(after)
			if ok2 {
				t.CreatedAt = second
				after = after2
			}
		} else {
			t.CreatedAt = first
		}
		trimmedLine = after
	}

	t.Description = strings.TrimSpace(trimmedLine)
	t.Projects, t.Contexts, t.Tags = extractTokens(t.Description)
	return t, true
}

// leadingDate parses a leading YYYY-MM-DD date from s.
// It returns the parsed date and the remainder (with leading spaces trimmed).
func leadingDate(s string) (*time.Time, string, bool) {
	if len(s) < 10 {
		return nil, s, false
	}
	candidate := s[:10]
	d, err := time.Parse(DateLayout, candidate)
	if err != nil {
		return nil, s, false
	}
	if len(s) > 10 && s[10] != ' ' {
		return nil, s, false
	}
	return &d, strings.TrimLeft(s[10:], " "), true
}

// extractTokens pulls projects, contexts and key:value tags from a description.
func extractTokens(desc string) (projects, contexts []string, tags map[string]string) {
	tags = map[string]string{}
	for _, tok := range strings.Fields(desc) {
		switch {
		case len(tok) > 1 && tok[0] == '+':
			projects = append(projects, tok[1:])
		case len(tok) > 1 && tok[0] == '@':
			contexts = append(contexts, tok[1:])
		default:
			if k, v, ok := splitTag(tok); ok {
				tags[k] = v
			}
		}
	}
	return projects, contexts, tags
}

// splitTag splits a "key:value" token. Both key and value must be non-empty and
// contain no additional colon-delimited emptiness.
func splitTag(tok string) (key, value string, ok bool) {
	i := strings.IndexByte(tok, ':')
	if i <= 0 || i == len(tok)-1 {
		return "", "", false
	}
	key, value = tok[:i], tok[i+1:]
	if strings.ContainsRune(value, ' ') {
		return "", "", false
	}
	return key, value, true
}

// String serializes the Todo back to a single todo.txt line.
func (t Todo) String() string {
	var b strings.Builder
	if t.Done {
		b.WriteString("x ")
	}
	if t.Priority >= 'A' && t.Priority <= 'Z' {
		b.WriteByte('(')
		b.WriteByte(t.Priority)
		b.WriteString(") ")
	}
	if t.Done {
		if t.CompletedAt != nil {
			b.WriteString(t.CompletedAt.Format(DateLayout))
			b.WriteByte(' ')
			if t.CreatedAt != nil {
				b.WriteString(t.CreatedAt.Format(DateLayout))
				b.WriteByte(' ')
			}
		}
	} else if t.CreatedAt != nil {
		b.WriteString(t.CreatedAt.Format(DateLayout))
		b.WriteByte(' ')
	}
	b.WriteString(t.Description)
	return b.String()
}

// ToTodoList parses a full todo.txt file body into a slice of Todos,
// skipping blank lines.
func ToTodoList(text string) []Todo {
	var todos []Todo
	for _, line := range strings.Split(text, "\n") {
		if t, ok := Parse(line); ok {
			todos = append(todos, t)
		}
	}
	return todos
}

// ToStringList serializes a slice of Todos into a todo.txt file body,
// terminated by a trailing newline when non-empty.
func ToStringList(todos []Todo) string {
	if len(todos) == 0 {
		return ""
	}
	lines := make([]string, len(todos))
	for i, t := range todos {
		lines[i] = t.String()
	}
	return strings.Join(lines, "\n") + "\n"
}

// SortByPriority returns a stable ordering: incomplete tasks first (by priority,
// A highest, unprioritized last), then completed tasks. It is a helper for the
// UI and does not mutate the input.
func SortByPriority(todos []Todo) []Todo {
	out := make([]Todo, len(todos))
	copy(out, todos)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Done != b.Done {
			return !a.Done
		}
		return priorityRank(a.Priority) < priorityRank(b.Priority)
	})
	return out
}

func priorityRank(p byte) int {
	if p >= 'A' && p <= 'Z' {
		return int(p - 'A')
	}
	return 1 << 30
}
