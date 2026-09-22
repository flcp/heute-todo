// Package config loads and saves the user's persistent preferences (todo file
// path, sort mode, done visibility and theme) as JSON under the OS config dir.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the persisted user preferences.
type Config struct {
	Path     string `json:"path"`      // default todo.txt to open
	Sort     string `json:"sort"`      // "file" | "priority" | "name"
	ShowDone bool   `json:"show_done"` // whether completed tasks are shown
	Theme    string `json:"theme"`     // palette name, e.g. "nord" | "default"
}

// Default returns the built-in defaults used when no config file exists yet.
func Default() Config {
	return Config{Path: "todo.txt", Sort: "file", ShowDone: true, Theme: "nord"}
}

// FilePath returns the path to the config file: <os-config-dir>/heute/config.json.
func FilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "heute", "config.json"), nil
}

// Load reads the config file, returning it with existed=true. When the file is
// absent it returns Default() with existed=false and no error. Fields missing
// from the file keep their default values.
func Load() (cfg Config, existed bool, err error) {
	cfg = Default()
	path, err := FilePath()
	if err != nil {
		return cfg, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil
		}
		return cfg, false, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, true, err
	}
	return cfg, true, nil
}

// Save writes cfg to the config file, creating the directory if needed.
func Save(cfg Config) error {
	path, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
