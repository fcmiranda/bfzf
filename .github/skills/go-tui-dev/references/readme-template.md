# README Template for Go TUI Libraries

Use this template when updating the README after any public API change.
Sections appear in this order.

---

## Section 1 — Install

```markdown
## Install

```bash
go get github.com/<user>/<pkg>@latest
```
```

---

## Section 2 — Quick Start

Minimal working snippet. Must compile as-is.

```markdown
## Quick Start

```go
package main

import (
    "fmt"
    "os"

    "<module>" // replace with real module path
    tea "charm.land/bubbletea/v2"
)

func main() {
    items := []<pkg>.Item{
        <pkg>.NewHeader("Group"),
        <pkg>.NewItem("Apple"),
        <pkg>.NewItem("Banana"),
    }

    m := <pkg>.New(items, <pkg>.WithHeight(12))
    p := tea.NewProgram(m)
    result, err := p.Run()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    fm := result.(<pkg>.Model)
    if fm.Submitted() {
        fmt.Println(fm.Selected()[0].Label())
    }
}
```
```

---

## Section 3 — Key Bindings

Markdown table using `DefaultKeyMap()` values. Add an "Overridable?" column.

```markdown
## Key Bindings

| Action | Default Key | Overridable? |
|--------|-------------|-------------|
| Move down | `↓` / `ctrl+n` | ✓ |
| Move up | `↑` / `ctrl+k` | ✓ |
| Jump to start | `Home` | ✓ |
| Jump to end | `End` | ✓ |
| Toggle selection | `Tab` | ✓ |
| Select all | `ctrl+a` | ✓ |
| Confirm | `Enter` | ✓ |
| Quit | `Esc` | ✓ |
| Abort | `ctrl+c` | ✓ |
| Scroll preview ↓ | `Shift+↓` | ✓ |
| Scroll preview ↑ | `Shift+↑` | ✓ |

Override with `WithKeyMap(km)`:

```go
km := bfzf.DefaultKeyMap()
km.Submit = key.NewBinding(key.WithKeys("ctrl+s"))
m := bfzf.New(items, bfzf.WithKeyMap(km))
```
```

---

## Section 4 — Styles / Theming

```markdown
## Styles / Theming

Override individual style fields with `WithStyleFunc`:

```go
m := bfzf.New(items,
    bfzf.WithStyleFunc(func(s *bfzf.Styles) {
        s.CursorText    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
        s.MatchHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Underline(true)
        s.Header        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("87"))
    }),
)
```

Or replace all styles at once with `WithStyles(myStyles)`.
```

---

## Section 5 — CLI Flags (cmd/ packages only)

```markdown
## CLI Usage

```bash
ls | bfzf [flags]
bfzf [flags] item1 item2 ...
```

| Flag | Default | Description |
|------|---------|-------------|
| `--preview cmd` | — | Shell command; `{}` = focused item label |
| `--preview-position` | `right` | `right` or `bottom` |
| `--preview-size n` | `40` | Preview pane size in % |
| `--preview-border` | off | Box border around preview |
| `--header str` | — | Title shown above the list |
| `--list-border` | off | Box border around list pane |
| `--no-input` | off | Hide search input |
| `--no-sort` | off | Preserve input order (no score sort) |
| `-m`, `--multi` | off | Unlimited multi-select |
| `--limit n` | `1` | Max selections |
| `--prompt str` | `❯ ` | Search prompt text |
| `--json` | off | Parse input as JSON array of strings |
```

---

## Notes

- Keep code snippets short — prefer the library-use pattern, not the full program.
- Reference `example/main.go` for a richer walkthrough rather than duplicating it inline.
- After adding a new `With*()` option, add a row to whichever section it belongs to.
