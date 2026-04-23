# bfzf

`bfzf` is an `fzf`-inspired fuzzy picker built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Real-time fuzzy search** with match highlighting
- **fzf-compatible query operators** — `'exact`, `^prefix`, `suffix$`, `!negate`, multi-term AND
- **Regex search** — prefix query with `/` to activate regexp mode (e.g. `/\.go$`)
- **`--reverse`** — render list bottom-to-top
- **`--query STRING`** — pre-filled initial filter
- **`--exact`** — switch to substring/exact match mode
- **`--header-lines N`** — pinned non-scrolling column header above the list
- **`--preview-window hidden`** — start with preview hidden; `Ctrl+/` to toggle
- **`--preview-window wrap-word`** — word-level soft-wrap in preview pane
- **`--preview-wrap-sign STR`** — glyph shown on soft-wrapped continuation lines in preview
- **`--bind key:action`** — runtime bindings: `toggle-preview`, `toggle-wrap`, `toggle-wrap-word`, `toggle-preview-wrap-word`, `clear-query`, `reload(cmd)`, `accept`, `abort`
- **`--print0`** — NUL-separated output for filenames with spaces
- **`--input-width N`** — constrain the search input width
- **`--marker-selected` / `--marker-unselected`** — custom glyph strings for multi-select markers
- **`--wrap-word`** — word-level wrapping of long item labels in the list
- **`--wrap-sign STR`** — glyph prepended to continuation lines when `--wrap-word` is active
- **Info line** — match count (`42/1000`) shown above the list; `--no-info` to hide
- **Cursor row highlight** — full-width background highlight on the focused item (fzf-style)
- **`--border TYPE`** — wrap entire picker in a border (`rounded`, `sharp`, `bold`, `block`, `double`)
- **`--no-color`** — disable all ANSI colour output
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
# No pipe — works like fzf: uses $FZF_DEFAULT_COMMAND or find . -type f
./bfzf
./bfzf --preview 'cat {}'

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

# Reverse list order + pre-filled query
ls | ./bfzf --reverse --query go

# Exact match mode
ls | ./bfzf --exact --query '.go'

# Hidden preview pane, toggle with ctrl+/
ls | ./bfzf --preview 'cat {}' --preview-window hidden

# Runtime key binding: toggle preview with alt+p
ls | ./bfzf --preview 'cat {}' --bind 'alt+p:toggle-preview'

# Reload items with a bind action
ls | ./bfzf --bind 'ctrl+r:reload(find . -name "*.go")'

# Pin first line as a non-scrolling header
ls -l | ./bfzf --header-lines 1 --preview 'stat {-1}'

# NUL-separated output (pipeline-safe)
ls | ./bfzf --print0 | xargs -0 cat

# Constrain search input width
ls | ./bfzf --input-width 40

# Custom multi-select glyphs
ls | ./bfzf -m --marker-selected '▶ ' --marker-unselected '  '

# Regex search — prefix query with / to activate regexp mode
ls | ./bfzf --query '/\.go$'

# Word-wrap long labels in the list (alt+w to toggle at runtime)
cat long_paths.txt | ./bfzf --wrap-word --wrap-sign '↩ '

# Word-wrap in preview pane with a wrap indicator
ls | ./bfzf --preview 'cat {}' --preview-window wrap-word --preview-wrap-sign '↩'

# Outer border around the entire picker
ls | ./bfzf --border rounded

# No colour output (useful for logging/CI)
ls | ./bfzf --no-color

# Hide the match-count info line
ls | ./bfzf --no-info

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
| `-height val` | full screen | Component height: absolute lines (`20`) or percentage (`40%`); adapts on resize when given as `%` |
| `-group-prefix str` | — | Lines with this prefix become non-selectable headers (prefix stripped) |
| `-spinner-prefix str` | — | Lines with this prefix get an animated spinner (prefix stripped) |
| `-preview cmd` | — | Shell command for preview; supports `{}`, `{-1}`, `{n}` |
| `-preview-position` | `right` | `right` or `bottom` |
| `-preview-size n` | 40 | Preview pane size in percent (10–90) |
| `-preview-border` | false | Draw a rounded border around the preview pane; title + `n/total` counter are embedded in the top border line |
| `-no-sort` | false | Preserve input order (disable score sorting) |
| `-delimiter str` | `\n` | Line delimiter for plain-text input |
| `-0` | false | Use NUL (`\x00`) as delimiter |
| `-json` | false | Parse stdin as JSON array |
| `-header str` | — | Title shown in the top border line of the list pane |
| `-list-border` | false | Draw a rounded border around the list pane; `-header` is embedded in the top border line |
| `-no-input` | false | Hide search input (navigation only; `ctrl+f` toggles at runtime) |
| `-input-border` | false | Draw a rounded border around the search input |
| `-cursor str` | `❯ ` | Cursor-row prefix glyph |
| `-marker style` | `circles` | Multi-select marker style: `circles` `squares` `filled` `arrows` `checkmarks` `stars` `diamonds` |
| `-popup spec` | — | Start in tmux/Zellij popup; spec: `[center\|top\|bottom\|left\|right][,W%][,H%]` (e.g. `center`, `left,40%,90%`) |
| `-style` | `default` | Style preset: `default`, `full` (all borders), or `minimal` (no borders, no help) |
| `-preview-width n` | 0 (use %) | Preview pane width in columns (overrides `-preview-size` when > 0; right layout) |
| `-preview-height n` | 0 (use %) | Preview pane height in lines (overrides `-preview-size` when > 0; bottom layout) |
| `-color spec` | — | Comma-separated `key:value` color overrides — see [Color spec](#color-spec) |
| `-reverse` | false | Render list in reverse order (last item at top) |
| `-exact` | false | Exact match mode: disable fuzzy, use substring matching |
| `-query STRING` | — | Initial query string for pre-filtering |
| `-print0` | false | Output NUL-separated results instead of newline-separated |
| `-header-lines N` | 0 | Treat first N input lines as a pinned non-scrolling header (excluded from matching) |
| `-preview-window opts` | — | Preview window options; `hidden` starts with preview hidden (`ctrl+/` to toggle); `wrap-word` enables word-level wrapping in preview |
| `-marker-selected str` | — | Raw glyph for selected items in multi-select mode (e.g. `▶ `) |
| `-marker-unselected str` | — | Raw glyph for unselected items in multi-select mode |
| `-input-width N` | 0 | Constrain search input to N columns (0 = full width) |
| `-bind key:action` | — | Runtime key binding (repeatable); actions: `toggle-preview`, `toggle-wrap`, `toggle-wrap-word`, `toggle-preview-wrap-word`, `clear-query`, `abort`, `accept`, `reload(cmd)` |
| `-wrap-word` | false | Enable word-level wrapping of long item labels in the list |
| `-wrap-sign str` | — | Glyph prepended to continuation lines when `--wrap-word` is active (e.g. `↩ `) |
| `-preview-wrap-sign str` | — | Glyph on soft-wrapped continuation lines in the preview pane (e.g. `↩`) |
| `-no-info` | false | Hide the match-count info line |
| `-border type` | — | Wrap entire picker in a border: `rounded` (default), `sharp`, `bold`, `block`, `double`, `none` |
| `-no-color` | false | Disable all ANSI colour output |

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

| Key | Action | Overridable via `WithKeyMapFunc` |
|---|---|---|
| `↑` / `↓` | Navigate | `km.Up` / `km.Down` |
| `Enter` | Confirm selection | `km.Submit` |
| `Tab` / `Shift+Tab` | Toggle item (multi-select) | `km.ToggleAndNext` / `km.ToggleAndPrev` |
| `Ctrl+A` | Select all visible (unlimited multi-select) | `km.SelectAll` |
| `Esc` | Quit without selecting | `km.Quit` |
| `Ctrl+C` | Abort | `km.Abort` |
| `Home` / `End` | Jump to start / end | `km.Home` / `km.End` |
| `Shift+↑` / `Shift+↓` | Scroll preview up / down | `km.PreviewUp` / `km.PreviewDown` |
| `Shift+PgUp` / `Shift+PgDn` | Scroll preview by page | `km.PreviewPageUp` / `km.PreviewPageDown` |
| `Shift+Home` / `Shift+End` | Jump to preview top / bottom | `km.PreviewTop` / `km.PreviewBottom` |
| `Ctrl+F` | Toggle search input on/off | `km.ToggleInput` |
| `Ctrl+/` | Toggle preview pane on/off | `km.TogglePreview` |
| `Alt+/` | Toggle character-level wrap in list | `km.ToggleWrap` |
| `Alt+W` | Toggle word-level wrap in list | `km.ToggleWrapWord` |
| `Alt+Shift+W` | Toggle word-level wrap in preview | `km.TogglePreviewWrapWord` |

### Theming / Style overrides

```go
m := bfzf.New(items,
    bfzf.WithStyleFunc(func(s *bfzf.Styles) {
        s.CursorText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
        s.ListBorder = lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("99"))
    }),
)
```

### Key binding overrides

```go
import "charm.land/bubbles/v2/key"

m := bfzf.New(items,
    bfzf.WithKeyMapFunc(func(km *bfzf.KeyMap) {
        // Change quit key from Esc to q
        km.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
        // Change toggle-input from ctrl+f to ctrl+h
        km.ToggleInput = key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "toggle input"))
    }),
)
```

### Bordered input + list pane with title

```go
m := bfzf.New(items,
    bfzf.WithListTitle("Files"),
    bfzf.WithListBorder(),   // title is embedded in the top border line
    bfzf.WithInputBorder(),
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewBorder(), // item label + scroll counter in top border line
)
```

CLI equivalent:
```bash
ls | ./bfzf --header Files --list-border --input-border \
            --preview 'cat {}' --preview-border
```

### Style presets (`WithPreset`)

Presets mirror fzf's `--style` option — one option turns on a full visual mode:

| Preset | Description |
|---|---|
| `bfzf.PresetDefault` | No borders, plain separator bar (default) |
| `bfzf.PresetFull` | Rounded border around list, input, and preview pane; titles embedded in top border line |
| `bfzf.PresetMinimal` | No borders, help line hidden |

```go
m := bfzf.New(items,
    bfzf.WithPreset(bfzf.PresetFull),
    bfzf.WithListTitle("Files"),
    bfzf.WithPreview(previewFn),
)
```

CLI equivalent:
```bash
ls | ./bfzf --style full --header Files --preview 'cat {}'
```

### Explicit preview pane dimensions

```go
// Right-side preview with fixed 60-column width
m := bfzf.New(items,
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewWidth(60),
)

// Bottom preview with fixed 20-line height
m := bfzf.New(items,
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewPosition(bfzf.PreviewBottom),
    bfzf.WithPreviewHeight(20),
)
```

CLI equivalent:
```bash
ls | ./bfzf --preview 'cat {}' --preview-width 60
ls | ./bfzf --preview 'cat {}' --preview-position bottom --preview-height 20
```

### Color spec

`WithColor` / `--color` accepts a comma-separated list of `key:value` pairs.
Color values can be ANSI 256 numbers (`"212"`), hex strings (`"#ff87af"`), or
named 4-bit ANSI colors (`"red"`, `"bright-blue"`, etc.).

| Key | Affects |
|---|---|
| `fg` | Item text foreground |
| `fg+` | Cursor item foreground |
| `bg` | Item text background |
| `bg+` | Cursor item background |
| `hl` | Fuzzy match highlight foreground |
| `header` | Group header foreground |
| `prompt` / `pointer` | Cursor indicator foreground |
| `info` | Help text and preview line-count foreground |
| `border` | All three border foregrounds at once |
| `list-border` | List pane border foreground |
| `preview-border` | Preview pane border foreground |
| `input-border` | Search-input border foreground |
| `scrollbar` | Preview scrollbar track foreground |
| `scrollbar-thumb` | Preview scrollbar thumb foreground |

```go
m := bfzf.New(items,
    bfzf.WithPreset(bfzf.PresetFull),
    bfzf.WithColor("fg+:212,hl:220,border:99,preview-border:135"),
)
```

CLI equivalent:
```bash
ls | ./bfzf --style full --color 'fg+:212,hl:220,border:99,preview-border:135' --preview 'cat {}'
```

---

## fzf-compatible features

### `--reverse` — render list in reverse order

```bash
ls | ./bfzf --reverse
ls | ./bfzf --reverse --query go   # pre-filtered
```

Library: `bfzf.WithReverse()`

---

### `--query STRING` — initial pre-filled filter

```bash
ls | ./bfzf --query go            # start with "go" already typed
```

Library: `bfzf.WithQuery("go")`

---

### `--exact` — exact / substring match mode

Disables fuzzy matching; all query tokens become case-insensitive substring searches. Special operators still work in this mode.

```bash
ls | ./bfzf --exact
ls | ./bfzf --exact --query '*.go'
```

Library: `bfzf.WithExact()`

---

### Exact-match operators (fzf-compatible query syntax)

Even without `--exact`, special token prefixes change match semantics on a per-term basis:

| Token | Meaning |
|---|---|
| `'word` | Exact substring match |
| `^prefix` | Must start with `prefix` |
| `suffix$` | Must end with `suffix` |
| `!word` | Exclude fuzzy matches for `word` |
| `!'word` | Must NOT contain `word` (exact negate) |
| `!^word` | Must NOT start with `word` |
| `!word$` | Must NOT end with `word` |

Multiple tokens are ANDed: all must match independently.

```bash
# Files that contain "go" and don't end with ".sum"
ls | ./bfzf --query "'go !.sum$"
# Files starting with "cmd" that are not "main.go"
ls | ./bfzf --query "^cmd !'main"
```

---

### `--print0` — NUL-separated output

Safe for filenames containing newlines or spaces.

```bash
ls | ./bfzf --print0 | xargs -0 cat
```

---

### `--header-lines N` — pinned column headers

The first N input lines are rendered as a non-scrolling header above the list and excluded from fuzzy matching.

```bash
# ls -l output: first line is "total …"; pin it as header
ls -l | ./bfzf --header-lines 1 --preview 'stat {-1}'
```

Library: `bfzf.WithHeaderLines(1)`

---

### `--preview-window hidden` + `ctrl+/` toggle

Start with the preview pane hidden; toggle it at any time with `Ctrl+/`.

```bash
ls | ./bfzf --preview 'cat {}' --preview-window hidden
# custom toggle key:
ls | ./bfzf --preview 'cat {}' --preview-window hidden --bind 'alt+p:toggle-preview'
```

Library:
```go
m := bfzf.New(items,
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewHidden(),
)
// Override toggle key:
m := bfzf.New(items,
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewHidden(),
    bfzf.WithKeyMapFunc(func(km *bfzf.KeyMap) {
        km.TogglePreview = key.NewBinding(key.WithKeys("alt+p"), key.WithHelp("alt+p", "toggle preview"))
    }),
)
```

---

### `--marker-selected` / `--marker-unselected` — raw glyph strings

```bash
ls | ./bfzf -m --marker-selected '▶ ' --marker-unselected '  '
```

Library: `bfzf.WithMarkerGlyphs("▶ ", "  ")`

---

### `--input-width N` — constrain search input width

```bash
ls | ./bfzf --input-width 40
```

Library: `bfzf.WithInputWidth(40)`

---

### `--bind key:action` — runtime key bindings

| Action | Description |
|---|---|
| `toggle-preview` | Show/hide preview pane |
| `toggle-wrap` | Toggle character-level soft-wrap in the list |
| `toggle-wrap-word` | Toggle word-level wrap in the list |
| `toggle-preview-wrap-word` | Toggle word-level wrap in the preview pane |
| `clear-query` | Clear the search input |
| `abort` | Quit without selecting |
| `accept` | Confirm current selection |
| `reload(cmd)` | Re-run shell command and replace item list |

```bash
ls | ./bfzf --preview 'cat {}' --bind 'ctrl+/:toggle-preview'
ls | ./bfzf --bind 'ctrl+r:reload(find . -type f)'
ls | ./bfzf --bind 'alt+w:toggle-wrap-word'
ls | ./bfzf --bind 'ctrl+x:abort' --bind 'alt+enter:accept'
```

Library:
```go
m := bfzf.New(items,
    bfzf.WithBind("ctrl+/", bfzf.BindTogglePreview()),
    bfzf.WithBind("alt+w", bfzf.BindToggleWrapWord()),
    bfzf.WithBind("ctrl+r", bfzf.BindReloadItems(func() []bfzf.Item {
        out, _ := exec.Command("find", ".", "-type", "f").Output()
        var items []bfzf.Item
        for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
            items = append(items, bfzf.NewItem(line))
        }
        return items
    })),
    bfzf.WithBind("ctrl+space", bfzf.BindChangeQuery("myprefix")),
)
```

---

### Regex search (query starts with `/`)

Prefix your query with `/` to activate regexp matching (case-insensitive):

```bash
ls | ./bfzf --query '/\.go$'       # files ending in .go
ls | ./bfzf --query '/^cmd'        # files starting with cmd
ls | ./bfzf                        # type /test in the search box at runtime
```

---

### Word-wrap long labels (`--wrap-word`)

Enable word-level wrapping of long item labels in the list. Long entries are
split at word boundaries and continuation lines are indented to align with the
label start. Toggle at runtime with `Alt+W`.

```bash
cat long_paths.txt | ./bfzf --wrap-word
cat long_paths.txt | ./bfzf --wrap-word --wrap-sign '↩ '
```

Library:
```go
m := bfzf.New(items,
    bfzf.WithWrapWord(),
    bfzf.WithWrapSign("↩ "),
)
```

---

### Word-wrap in preview (`--preview-window wrap-word`)

Enable word-level soft-wrapping in the preview pane. Toggle at runtime with `Alt+Shift+W`.

```bash
ls | ./bfzf --preview 'cat {}' --preview-window wrap-word --preview-wrap-sign '↩'
```

Library:
```go
m := bfzf.New(items,
    bfzf.WithPreview(previewFn),
    bfzf.WithPreviewWrapWord(),
    bfzf.WithPreviewWrapSign("↩"),
)
```

---

### Cursor row highlight

The focused item row is highlighted with a full-width background bar (fzf-style).
Customise via `Styles.CursorRowBg`:

```go
m := bfzf.New(items,
    bfzf.WithStyleFunc(func(s *bfzf.Styles) {
        s.CursorRowBg = lipgloss.NewStyle().Background(lipgloss.Color("236"))
    }),
)
// Disable: set CursorRowBg to a zero-value lipgloss.Style
```

---

### `--info` / `--no-info` — match count display

A match count line (`42/1000`) is shown by default between the input and the list.
Hide it with `--no-info`:

```bash
ls | ./bfzf --no-info
```

Library: `bfzf.WithInfoStyle(bfzf.InfoHidden)`

---

### `--border` — outer picker border

Wrap the entire picker in a border:

```bash
ls | ./bfzf --border rounded
ls | ./bfzf --border sharp
ls | ./bfzf --border bold
```

Library:
```go
m := bfzf.New(items,
    bfzf.WithOuterBorder(lipgloss.RoundedBorder()),
)
```

---

### `--no-color` — disable ANSI colours

```bash
ls | ./bfzf --no-color
```

Library: `bfzf.WithNoColor()`

---

### Updated Key Bindings

| Key | Action | Overridable via `WithKeyMapFunc` |
|---|---|---|
| `↑` / `↓` | Navigate | `km.Up` / `km.Down` |
| `Enter` | Confirm selection | `km.Submit` |
| `Tab` / `Shift+Tab` | Toggle item (multi-select) | `km.ToggleAndNext` / `km.ToggleAndPrev` |
| `Ctrl+A` | Select all visible (unlimited multi-select) | `km.SelectAll` |
| `Esc` | Quit without selecting | `km.Quit` |
| `Ctrl+C` | Abort | `km.Abort` |
| `Home` / `End` | Jump to start / end | `km.Home` / `km.End` |
| `Shift+↑` / `Shift+↓` | Scroll preview up / down | `km.PreviewUp` / `km.PreviewDown` |
| `Shift+PgUp` / `Shift+PgDn` | Scroll preview by page | `km.PreviewPageUp` / `km.PreviewPageDown` |
| `Shift+Home` / `Shift+End` | Jump to preview top / bottom | `km.PreviewTop` / `km.PreviewBottom` |
| `Ctrl+F` | Toggle search input on/off (clears filter when hidden) | `km.ToggleInput` |
| `Ctrl+/` | Toggle preview pane on/off | `km.TogglePreview` |
| `Alt+/` | Toggle character-level wrap in list | `km.ToggleWrap` |
| `Alt+W` | Toggle word-level wrap in list | `km.ToggleWrapWord` |
| `Alt+Shift+W` | Toggle word-level wrap in preview | `km.TogglePreviewWrapWord` |

