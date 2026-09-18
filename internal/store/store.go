package store
// Package store loads and saves the todo.txt file from disk.
package store

import (
	"os"
	"path/filepath"

	"github.com/flcp/heute-todo/internal/todotxt"
)

// Load reads todos from the file at path. A missing file is treated as an empty
// list rather than an error, so the first run starts cleanly.
func Load(path string) ([]todotxt.Todo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return todotxt.ToTodoList(string(data)), nil
}

// Save writes todos to path atomically: it writes a temp file in the same
// directory and renames it into place, so a crash mid-write cannot corrupt the
// existing file.
func Save(path string, todos []todotxt.Todo) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".todo-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(todotxt.ToStringList(todos)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
