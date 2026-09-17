# HEUTE TODO — Plan

A vim-style TUI todo list app. Framework: **Go + Bubble Tea** (see [framework.md](framework.md)).

## Storage
- Persist to `todo.txt` following the [todo.txt format](https://github.com/todotxt/todo.txt).
- Format: `(A) 2026-09-17 Task text +project @context due:2026-09-20`, completed lines prefixed `x YYYY-MM-DD`.
- **Autosave** on every change (add/edit/toggle/delete).

## Modes
- **Normal** — navigation and actions.
- **Insert** — add/edit a todo (textinput).
- **Search** — `/` filters the list.
- **Command** — `:` command line (initially `:q` / `:quit` only).

## Keybindings

### Navigation (Normal)
| Key | Action |
|-----|--------|
| `j` / `↓` | move down |
| `k` / `↑` | move up |
| `g` | go to top |
| `G` | go to bottom |

### Actions (Normal)
| Key | Action |
|-----|--------|
| `o` | add new todo below (→ Insert) |
| `O` | add new todo above (→ Insert) |
| `i` | edit selected, cursor at start (→ Insert) |
| `I` | edit selected, cursor at first non-blank (→ Insert) |
| `space` | toggle done |
| `dd` | delete todo (press `d`, then `d` to confirm) |
| `u` | undo last change |
| `r` | redo |
| `/` | search (→ Search) |
| `:` | command mode (→ Command) |

### Search mode
- Type to filter; `Enter` apply, `Esc` cancel.

### Delete confirmation (`dd`)
- First `d` enters "pending delete" state and shows a hint in the footer, e.g. `delete? press d to confirm, esc to cancel` (vim operator-pending style).
- Second `d` confirms; any other key cancels.

### Command mode
- `:q` / `:quit` — quit.
- (extensible later: `:w`, `:wq`, etc.)

## State management
- No external state library needed. Bubble Tea's Elm-style `model` + centralized `Update` loop already provides single-source-of-truth state.
- Undo/redo implemented directly: keep `undoStack [][]Todo` (and optional `redoStack`). Before each mutation, push a copy of todos; `u` pops undo → redo. Snapshotting the small todo slice is cheap and simpler than diff-based approaches.

## Architecture
- `model` holds: todos slice, cursor, mode, textinput, search query, undo stack.
- `Update` handles keys per mode; mutations push previous state to undo stack, then autosave.
- `View` renders list with styling; header shows counts, footer shows mode/help.
- Packages:
  - `todotxt` — parse/serialize `todo.txt`.
  - `store` — load/save file.
  - `ui` — Bubble Tea model, update, view.

## Milestones
1. Project scaffold, todo.txt parse/serialize + tests.
2. List rendering + hjkl/gG navigation.
3. Add/edit/toggle/delete + autosave.
4. Undo (`u`).
5. Search (`/`).
6. Command mode (`:q`).
7. Styling polish (Lip Gloss).

## Out of scope (from lazytodo, maybe later)
Themes, braille priority charts, multi-panel projects/contexts view, yank/paste, CLI subcommands.
