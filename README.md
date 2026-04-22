# bfzf

`bfzf` is an `fzf`-inspired fuzzy picker built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Real-time fuzzy search** with match highlighting
- **CLI Tool** — pipe stdin or pass positional arguments, just like `fzf`
- **Live Preview** — run any shell command on the focused item; split pane (right or bottom)
- **Preview field selectors** — `{-1}` extracts the last field (filename from `ls -l`), `{n}` for nth field
- **Grouped options** — non-selectable headers visually separate groups
- **Animated Spinners** — per-item Bubble spinners (e.g. indicating background tasks)
- **JSON input** — accept `--json` arrays of strings or rich objects
- **Multi-select** — `Tab` to toggle, configurable limit
- **Fully customisable** styles and key bindings

---

## Quick start

```bash
# Build binary into project root
make build

# Or install system-wide
make install          # puts `bfzf` on your PATH via go install
```

---

## CLI Usage

```bash
# Basic — pipe any command's output
ls | ./bfzf

# With live preview — {} = full label
ls | ./bfzf --preview 'cat {}'

# Preview field selectors
# {-1} = last whitespace field = filename when ls outputs columns (eza, ls -l, etc.)
ls -l | ./bfzf --preview 'stat {-1}'
eza -l | ./bfzf --preview 'file {-1}'

# Preview split at the bottom (default is right)
ls | ./bfzf --preview 'cat {}' --preview-position bottom --preview-size 50

# Group headers (lines starting with prefix become non-selectable headers)
printf '#Fruits\nApple\nBanana\n#Veg\nCarrot' | ./bfzf --group-prefix '#'

# Animated spinners (lines starting with prefix get a Bubble spinner)
printf '@Building…\nReady item' | ./bfzf --spinner-prefix '@'

# Multi-select (Tab to toggle, Enter to confirm)
ls | ./bfzf -m

# Positional arguments instead of stdin
./bfzf item1 item2 item3

# JSON input — array of strings
echo '["Apple","Banana","Cherry"]' | ./bfzf --json

# JSON input — rich objects (headers + spinners)
cat <<'EOF' | ./bfzf --json
[
  {"label": "── Fruits", "header": true},
  {"label": "Apple"},
  {"label": "Banana"},
  {"label": "── Building", "header": true},
  {"label": "Carbon", "spinner": true},
  {"label": "Mojo",   "spinner": true}
]
EOF
```

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-m`, `-multi` | false | Enable unlimited multi-select |
| `-limit n` | 1 | Max selections (overridden by `-multi`) |
| `-prompt str` | `❯ ` | Search prompt |
| `-placeholder str` | `Filter…` | Search box placeholder |
| `-height n` | 0 (full screen) | Terminal lines to use |
| `-group-prefix str` | — | Lines with this prefix become non-selectable headers (prefix stripped) |
| `-spinner-prefix str` | — | Lines with this prefix get an animated spinner (prefix stripped) |
| `-preview cmd` | — | Shell command for preview; supports `{}`, `{-1}`, `{n}` |
| `-preview-position` | `right` | `right` or `bottom` |
| `-preview-size n` | 40 | Preview pane size in percent (10–90) |
| `-no-sort` | false | Preserve input order (disable score sorting) |
| `-delimiter str` | `\n` | Line delimiter for plain-text input |
| `-0` | false | Use NUL (`\x00`) as delimiter |
| `-json` | false | Parse stdin as JSON array |

### Preview field selectors

| Selector | Expands to |
|---|---|
| `{}` | Full item label (shell-quoted) |
| `{-1}` | Last whitespace-split field — e.g. the **filename** from `eza -l` or `ls -l` output |
| `{1}` | First field |
| `{n}` | nth field (1-based) |
| `{-n}` | nth-from-last field |

```bash
# eza / ls -l outputs:  .rw-r--r-- 12k user date  filename
# {-1} extracts "filename"
eza -l | ./bfzf --preview 'bat --color=always {-1}'
```

### JSON input format

**Simple** — plain string array:
```json
["Apple", "Banana", "Cherry"]
```

**Rich** — object array with optional `header` and `spinner` fields:
```json
[
  {"label": "── Fruits",  "header": true},
  {"label": "Apple"},
  {"label": "Banana"},
  {"label": "── In Progress", "header": true},
  {"label": "Carbon",  "spinner": true},
  {"label": "Mojo",    "spinner": true}
]
```

---

## Building

```bash
make build     # → ./bfzf
make install   # → go install (system-wide)
make clean     # remove ./bfzf
```

Or build manually:
```bash
go build -o bfzf ./cmd/bfzf/
```

---

## Library Usage (Go)

### Basic example

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/fecavmi/bfzf"
)

func main() {
	items := []bfzf.Item{
		bfzf.NewHeader("Fruits"),
		bfzf.NewItem("Apple"),
		bfzf.NewItem("Banana"),
	}

	m := bfzf.New(items,
		bfzf.WithPreview(func(i bfzf.Item) string {
			return "Selected: " + i.Label()
		}),
	)

	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if fm, ok := final.(bfzf.Model); ok && fm.Submitted() {
		fmt.Println(fm.Selected()[0].Label())
	}
}
```

### With spinners

```go
import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/fecavmi/bfzf"
)

type myItem struct {
	name string
	s    spinner.Model
}

func (m myItem) Label() string          { return m.name }
func (m myItem) FilterValue() string    { return m.name }
func (m myItem) IsHeader() bool         { return false }
func (m myItem) Spinner() spinner.Model { return m.s }

func newSpinnerItem(name string) myItem {
	return myItem{
		name: name,
		s: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("214"))),
		),
	}
}

// then:
items := []bfzf.Item{
	bfzf.NewHeader("In Progress"),
	newSpinnerItem("Building Carbon…"),
	newSpinnerItem("Downloading Mojo…"),
}
m := bfzf.New(items)
```

### Key bindings

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate |
| `Enter` | Confirm selection |
| `Tab` / `Shift+Tab` | Toggle item (multi-select) |
| `Ctrl+A` | Select all visible (unlimited multi-select) |
| `Esc` | Quit without selecting |
| `Ctrl+C` | Abort |
| `Home` / `End` | Jump to start / end |

