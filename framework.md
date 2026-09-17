# Framework Evaluation — HEUTE TODO

## Candidates

### 1. Go + Bubble Tea (+ Lip Gloss / Bubbles)
- Elm-style architecture (Model/Update/View), great fit for a keyboard-driven TUI.
- Mature ecosystem: Bubbles (list, textinput), Lip Gloss (styling).
- Compiles to a single static binary, no runtime deps.
- Excellent key handling for vim-style bindings and modal input.
- Large community, well documented.

### 2. Rust + Ratatui
- High performance, single binary.
- Immediate-mode rendering; more manual state/event wiring.
- Steeper learning curve, slower iteration than Go.

### 3. TypeScript + OpenTUI / Ink (what lazytodo uses)
- Fast to prototype, React-like.
- Requires a JS runtime (Bun/Node) unless bundled; heavier distribution.
- Weaker fit for a small, single-binary CLI tool.

### 4. Python + Textual
- Very ergonomic, rich widgets.
- Requires Python runtime; packaging a standalone binary is awkward.

## Decision: Go + Bubble Tea

Reasons:
- Single self-contained binary — easy to install and run.
- Modal, keyboard-first design maps naturally onto the Elm update loop.
- Bubbles `textinput` covers add/edit/search/command inputs out of the box.
- Best balance of dev speed, performance, and distribution for a TUI todo app.

Stack:
- `github.com/charmbracelet/bubbletea` — core loop
- `github.com/charmbracelet/bubbles` — list, textinput
- `github.com/charmbracelet/lipgloss` — styling
