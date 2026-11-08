# heute

A fast, vim-style terminal todo list that reads and writes the plain-text
[todo.txt](https://github.com/todotxt/todo.txt) format. Built with Go and
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Install

```sh
ln -sf "$HOME/go/bin/heute" "$HOME/.local/bin/heute" && \
go install github.com/flcp/heute-todo/cmd/heute@latest
```

The first command symlinks the binary into `~/.local/bin` (on your `PATH`);
the second installs `heute` into `~/go/bin`, which the symlink points at. Run
it again any time to upgrade.

Or build from source:

```sh
go install ./cmd/heute
```

## Usage

```sh
heute                   # open your todo file (the saved path, or todo.txt)
heute --set-path path.txt   # set and remember the todo file to open
```

Autosaves changes on every edit. The file path, sort mode,
visibility, and theme persist to `<os-config-dir>/heute/config.json`.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | move down / up |
| `g` / `G` | jump to top / bottom |
| `o` / `O` | add a task below / above |
| `a` / `i` / `enter` | edit the selected task |
| `space` | toggle done |
| `0`–`9` | set priority (`0` clears) |
| `J` / `K` | reorder the selected task (free sort), or move it down / up in priority — `J` lowers priority, `K` raises it (priority sort) |
| `d` `d` | delete (press `d` twice to confirm) |
| `u` / `ctrl+r` | undo / redo |
| `s` | cycle sort mode |
| `z` | show / hide completed tasks |
| `tab` / `shift+tab` | focus the project / context filter |
| `/` | search tasks (see below) |
| `q` | quit |

In a filter box: `j`/`k` to move, `space` to toggle, `!` to isolate a single
tag, `esc` to return to the list.

## Search

The search query is split
on spaces and each word is matched individually.
So `write boss` matches `write email to boss`.

# Developer Information
## Run as dev
```sh
go run ./cmd/heute
```