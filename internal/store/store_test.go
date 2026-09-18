package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flcp/heute-todo/internal/todotxt"
)

func TestLoadMissingFile(t *testing.T) {
	todos, err := Load(filepath.Join(t.TempDir(), "nope.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected empty list, got %d", len(todos))
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	body := "(A) 2026-09-17 Write tests +heute @work due:2026-09-20\n" +
		"x 2026-09-18 Do the thing\n"
	todos := todotxt.ToTodoList(body)

	if err := Save(path, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != body {
		t.Errorf("file body = %q, want %q", string(data), body)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != len(todos) {
		t.Errorf("loaded %d todos, want %d", len(loaded), len(todos))
	}
}

func TestSaveCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "todo.txt")
	if err := Save(path, todotxt.ToTodoList("just a task\n")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}
