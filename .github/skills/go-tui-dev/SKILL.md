---
name: go-tui-dev
description: 'Go TUI Developer Specialist using Bubble Tea, Bubbles, and Lip Gloss. Use when: building a new TUI component, adding preview panes, split layouts, spinners, fuzzy filtering, key bindings, custom styles, golden-file tests for View() output, or updating README with install/quick-start/keybindings/theming/CLI flags sections. Triggers: bubbletea, bubbles, lipgloss, tea.Model, viewport, textinput, spinner, fuzzy picker, TUI, terminal UI, bfzf.'
argument-hint: 'Describe the TUI feature or component to build (e.g. "add a preview pane", "write golden tests for Filter component")'
---

# Go TUI Developer Specialist

Deep-expertise workflow for building, extending, and testing terminal UIs with
[Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles), and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## When to Use

- Building or extending a `tea.Model` component
- Adding split-layout panes (right/bottom preview, list+detail)
- Implementing spinners, fuzzy filtering, viewports, or text inputs
- Writing golden-file tests for `View()` output
- Adding a `cmd/` CLI binary that wraps the library
- Updating the README after any public-API change

---

## Procedure

### Step 1 — Understand the Request

Before writing code, ask:

1. **Scope**: New component, new feature in existing model, or bug fix?
2. **Layout**: Does it require a split pane? Which direction (right / bottom)?
3. **Input model**: Will there be a text-input search box? Hidden in cmd mode?
4. **Limit / selection**: Single-select, multi-select, or unlimited?
5. **Tests**: Are golden-file tests needed? (always say yes unless they exist)
6. **README**: Does any public API change need to be reflected?

> If the answer to any question is unclear, ask. Good questions prevent
> layout bugs and API breaks.

---

### Step 2 — Plan the Phase

Sketch a phased delivery before writing code:

```
Phase 1 — Core model (tea.Model, fields, Init/Update/View)
Phase 2 — Sub-types (headers, spinners, special items)
Phase 3 — Styling + KeyMap
Phase 4 — Preview / split layout (if needed)
Phase 5 — CLI binary (cmd/ package, if needed)
Phase 6 — Tests
Phase 7 — README update
```

Capture the plan in `PLAN.md` only if the feature is complex (≥3 phases).

---

### Step 3 — File Layout

Follow this canonical structure:

```
<package>/
├── <pkg>.go        ← Model, Item interface, concrete types, Init/Update/View
├── styles.go       ← Styles struct, DefaultStyles()
├── keymap.go       ← KeyMap struct, DefaultKeyMap(), matchesAny()
├── options.go      ← Option type, With*() functional options
├── preview.go      ← PreviewFunc, PreviewPosition, triggerPreview() (if split layout)
├── cmd/<pkg>/
│   └── main.go     ← CLI binary (flags, stdin, tty redirection)
└── example/
    └── main.go     ← Runnable showcase
```

See [./references/file-layout.md](./references/file-layout.md) for full templates.

---

### Step 4 — Implement the Model

Follow the rules in [./references/bubbletea-rules.md](./references/bubbletea-rules.md).

Key non-obvious rules (violations cause hard-to-debug layout bugs):

| Rule | Detail |
|------|--------|
| **Never put a full-width widget inside a sub-pane** | `input.View()` is `m.width` wide. If it's inside `renderListPane()`, and that pane is `JoinHorizontal`-ed with a preview, total width = `m.width + previewW`. Extract the input above the split in `render()`. |
| **Always subtract border frames in `resize()`** | When a border is enabled, `vpW = areaW - style.GetHorizontalFrameSize()`, `vpH = areaH - style.GetVerticalFrameSize()`. Skipping this causes overflow. |
| **Stale-check async preview results** | In the `previewResultMsg` handler, only apply content when `msg.itemIdx == m.lastPreviewIdx`. |
| **`View()` returns `tea.View`** | Use `return tea.NewView(m.render())`, not a plain string. |
| **Key events are `tea.KeyPressMsg`** | Use `key.Matches(kp, binding)` from `charm.land/bubbles/v2/key`. |
| **Separate global keys from local keys** | Handle Quit/Abort/scroll at the top of the `switch`; list navigation in a nested `default`. |

---

### Step 5 — Styling

```go
// styles.go pattern
type Styles struct {
    // one field per visual element
}

func DefaultStyles() Styles { ... }
```

- Use `lipgloss.Color("ansi256code")` for portable 256-colour values.
- Expose `WithStyleFunc(fn func(*Styles))` so callers can patch individual fields.
- For borders: `lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(...)`.
- Query frame sizes via `style.GetHorizontalFrameSize()` / `style.GetVerticalFrameSize()`.

---

### Step 6 — KeyMap

```go
// keymap.go pattern
type KeyMap struct {
    // one key.Binding per action
    CursorPrefix string // rendered glyph, not a real binding
}

func DefaultKeyMap() KeyMap { ... }

func matchesAny(msg tea.Msg, bindings ...key.Binding) bool {
    if kp, ok := msg.(tea.KeyPressMsg); ok {
        return key.Matches(kp, bindings...)
    }
    return false
}
```

---

### Step 7 — Split Layout (Preview Pane)

See [./references/split-layout.md](./references/split-layout.md).

Quick checklist:

- [ ] `resize()` subtracts `listBorderH`, `prevBorderH`, `prevBorderV` from viewport dimensions
- [ ] `render()` renders `input.View()` **above** `JoinHorizontal()`/`JoinVertical()`
- [ ] `renderListPane()` contains only title + `vp.View()`; **no input**
- [ ] `renderPreviewPane()` contains title bar + `previewVP.View()`; wraps with border if enabled
- [ ] `triggerPreview()` is no-op when the cursor item hasn't changed
- [ ] Preview subprocess runs with `TERM=xterm-256color`, `COLORTERM=truecolor`, `CLICOLOR_FORCE=1`

---

### Step 8 — Tests (Golden Files)

See [./references/testing.md](./references/testing.md).

Pattern:

```go
// <pkg>_test.go
func TestView_Filter(t *testing.T) {
    m := New(items, WithHeight(10), WithWidth(60))
    // send tea messages via teatest or direct Update calls
    got := string(m.View())
    golden.RequireEqual(t, []byte(got))
}
```

- Store snapshots in `testdata/<TestName>.golden`.
- Use `UPDATE_GOLDEN=1 go test ./...` to regenerate.
- Strip ANSI before comparing when the test environment has no colour profile.
- Always test: empty list, single item, cursor at bottom, filtered state.

---

### Step 9 — README Update

After any public API change, update the README with these sections **in this order**:

1. **Install** — `go get` command with the module path
2. **Quick Start** — minimal working Go snippet using `New()` + `tea.NewProgram`
3. **Key Bindings** — markdown table (Action | Default Key | Overridable?)
4. **Styles / Theming** — short snippet using `WithStyleFunc`
5. **CLI Flags** — table (Flag | Default | Description) if a `cmd/` binary exists

See [./references/readme-template.md](./references/readme-template.md) for section boilerplate.

---

## go.mod Notes

- For local multi-repo development use `replace` directives:
  ```
  replace charm.land/bubbletea/v2 => ../bubbletea
  replace charm.land/bubbles/v2   => ../bubbles
  replace charm.land/lipgloss/v2  => ../lipgloss
  ```
- Always run `go mod tidy` after adding imports.
- Run `go vet ./...` before declaring a phase complete.

---

## Quality Gates (Definition of Done)

Before closing any phase:

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` leaves no diff
- [ ] Golden tests pass (or are created for new `View()` paths)
- [ ] README updated if public API changed
- [ ] No full-width widget inside a split sub-pane
- [ ] Border frame sizes subtracted in `resize()` wherever borders are enabled
