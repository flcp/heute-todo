package config

import (
	"os"
	"testing"
)

// isolateConfigDir points os.UserConfigDir at a temp directory for the test.
func isolateConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS (~/Library/Application Support)
}

func TestLoadReturnsDefaultsWhenMissing(t *testing.T) {
	isolateConfigDir(t)
	cfg, existed, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if existed {
		t.Fatal("existed = true, want false for a missing file")
	}
	if cfg != Default() {
		t.Fatalf("cfg = %+v, want defaults %+v", cfg, Default())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	isolateConfigDir(t)
	want := Config{Path: "/tmp/todo.txt", Sort: "priority", ShowDone: false, Theme: "default"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, existed, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !existed {
		t.Fatal("existed = false, want true after Save")
	}
	if got != want {
		t.Fatalf("round-trip = %+v, want %+v", got, want)
	}
}

func TestLoadPartialKeepsDefaults(t *testing.T) {
	isolateConfigDir(t)
	path, err := FilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// Only "sort" is present; the rest must fall back to defaults.
	if err := os.WriteFile(path, []byte(`{"sort":"name"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sort != "name" {
		t.Fatalf("sort = %q, want name", cfg.Sort)
	}
	if cfg.ShowDone != Default().ShowDone || cfg.Theme != Default().Theme {
		t.Fatalf("missing fields not defaulted: %+v", cfg)
	}
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
