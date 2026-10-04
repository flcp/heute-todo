# heute

A fast, vim-style terminal todo list that reads and writes the plain-text
[todo.txt](https://github.com/todotxt/todo.txt) format. Built with Go and
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Install

```sh
go install github.com/flcp/heute-todo@latest
```

Or build from source:

```sh
go build -o heute .
```

## Usage

```sh
heute            # open the default todo.txt
heute path.txt   # open a specific file (and remember it as the default)
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
| `J` / `K` | reorder the selected task (free sort only) |
| `d` `d` | delete (press `d` twice to confirm) |
| `s` | cycle sort mode |
| `z` | show / hide completed tasks |
| `tab` / `shift+tab` | focus the project / context filter |
| `q` | quit |

In a filter box: `j`/`k` to move, `space` to toggle, `!` to isolate a single
tag, `esc` to return to the list.
