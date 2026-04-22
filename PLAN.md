# bfzf — CLI Wrapper & Preview Feature: Implementation Plan

## Overview

bfzf is a fuzzy-finder TUI library built on Bubble Tea + Bubbles + Lip Gloss.
This plan extends it with:

1. **Preview pane** — like fzf's `--preview 'cmd {}'`, rendered in a
   scrollable split panel (right or bottom) that updates as the cursor moves.
2. **CLI binary** (`cmd/bfzf`) — lets you pipe standard input or pass positional
   arguments, exactly like fzf:

   ```sh
   ls | bfzf
   bfzf item1 item2 item3
   ls | bfzf --preview 'cat {}'
   printf '#Fruits\nApple\nBanana\n#Veg\nCarrot' | bfzf --group-prefix '#'
   printf '@Loading…\nReady item' | bfzf --spinner-prefix '@'
   ```

---

## Architecture

```
bfzf/
├── bfzf.go          ← core Model (MODIFIED: preview fields, split layout)
├── preview.go       ← NEW: PreviewFunc, PreviewPosition, previewResultMsg
├── styles.go        ← MODIFIED: add PreviewBorder, PreviewTitle styles
├── options.go       ← MODIFIED: WithPreview, WithPreviewPosition,
│                       WithPreviewSize, WithNoSort
├── keymap.go        (unchanged)
├── cmd/
│   └── bfzf/
│       └── main.go  ← NEW: CLI binary
└── example/
    └── main.go      (unchanged)
```

---

## Phase 1 — Preview in the Library

### 1.1 `preview.go` (new file)

- **`PreviewFunc`** — `type PreviewFunc func(Item) string`  
  Called in a goroutine by `tea.Cmd`; may safely block (e.g., exec).
- **`PreviewPosition`** — `right` (default) or `bottom`.
- **`previewResultMsg`** — internal message carrying rendered content + item
  index for stale-result rejection.

### 1.2 Model changes (`bfzf.go`)

New fields on `Model`:

| Field | Type | Purpose |
|---|---|---|
| `previewFunc` | `PreviewFunc` | nil = no preview |
| `previewVP` | `viewport.Model` | scrollable preview pane |
| `previewPos` | `PreviewPosition` | right or bottom |
| `previewSize` | `int` | percentage of space for preview (default 40) |
| `lastPreviewIdx` | `int` | last triggered item index (stale-check) |
| `sortResults` | `bool` | controls fuzzy sort (default true) |

**`triggerPreview()`** — emits a `tea.Cmd` whenever the focused item changes.
Returns nil if same item as last time (idempotent). Discards stale responses
via `msg.itemIdx == m.lastPreviewIdx` check.

**`resize()`** — three cases:
- No preview → original behaviour.
- `PreviewRight` → list gets `width * (100-pct)/100` cols; preview gets the rest
  minus 1 for the border character.
- `PreviewBottom` → height is split between list and preview viewports.

**`render()`** — branches on `m.previewFunc`:
- No preview → unchanged output.
- `PreviewRight` → `lipgloss.JoinHorizontal(Top, listPanel, borderChar, previewPanel)`.
- `PreviewBottom` → border separator line between list and preview.

Preview pane always shows a title bar (item label) and the scrollable
`previewVP.View()` content.

### 1.3 Options

| Option | Description |
|---|---|
| `WithPreview(fn PreviewFunc)` | Attach a preview function |
| `WithPreviewPosition(pos)` | `PreviewRight` or `PreviewBottom` |
| `WithPreviewSize(pct int)` | Percentage width/height for preview (10–90) |
| `WithNoSort()` | Use `fuzzy.FindNoSort` instead of `fuzzy.Find` |

---

## Phase 2 — CLI Wrapper (`cmd/bfzf/main.go`)

### 2.1 Input sources (priority order)

1. Positional arguments (`bfzf item1 item2 …`).
2. Standard input pipe (`ls | bfzf`).
3. Error with usage hint if both are empty.

When stdin is a pipe, bfzf reads all lines before starting the TUI, then
redirects keyboard input to `/dev/tty` and renders the TUI on stderr so that
stdout remains clean for piped output.

### 2.2 CLI flags

```
Usage: bfzf [flags] [item ...]

  -m, -multi              Enable unlimited multi-select
  -limit n                Max selections (default 1; overridden by -multi)
  -prompt str             Search prompt (default "❯ ")
  -placeholder str        Placeholder text (default "Filter…")
  -height n               Terminal lines (0 = full screen, default 0)
  -group-prefix str       Lines starting with this prefix become group headers
                          (prefix is stripped from the displayed label)
  -spinner-prefix str     Lines starting with this prefix get an animated spinner
                          (prefix is stripped from the displayed label)
  -preview cmd            Shell command; {} is replaced with focused item label
  -preview-position side  right (default) or bottom
  -preview-size n         Preview pane size in percent (default 40)
  -no-sort                Disable fuzzy-match score sorting (preserve input order)
  -delimiter str          Field delimiter for reading input (default newline)
  -0                      Use NUL (\x00) as delimiter (like fzf -0)
```

### 2.3 Item annotation (group-prefix / spinner-prefix)

Lines are classified in order:

```
─ group-prefix match  →  HeaderItem   (non-selectable, skipped by cursor)
─ spinner-prefix match →  SpinnerItem  (animated Dot spinner, default gold color)
─ otherwise           →  SimpleItem
```

Prefix is stripped from the label in all cases.

### 2.4 Preview execution

```
sh -c "<cmd with {} replaced by shell-quoted item label>"
```

The label is wrapped in single quotes with internal single quotes escaped
(`'→'\\''`), preventing shell injection from item values.
stdout is captured and shown in the preview pane; stderr from the preview
command appears as error text in the pane.

### 2.5 Output

- Selected labels are printed to **stdout**, one per line.
- TUI renders to **stderr** (so `bfzf | wc -l` works).
- Exit codes: `0` = selection made, `1` = aborted/no selection.

---

## Phase 3 — Validation

1. `go build -o /tmp/bfzf ./cmd/bfzf/` builds without errors.
2. `go vet ./...` passes.
3. Manual smoke tests:
   - `ls | ./bfzf` — full screen list from stdin.
   - `./bfzf a b c` — items from args.
   - `ls | ./bfzf --preview 'file {}'` — preview pane on right.
   - `ls | ./bfzf --preview 'cat {}' --preview-position bottom` — bottom split.
   - `printf '#G1\nfoo\nbar\n#G2\nbaz' | ./bfzf --group-prefix '#'` — headers.
   - `printf '@loading\nnormal' | ./bfzf --spinner-prefix '@'` — spinners.
   - `ls | ./bfzf -m` — multi-select.

---

## Implementation Status

- [x] Phase 0: Core library (bfzf.go, styles, keymap, options)
- [x] Phase 1: Preview support in library
- [x] Phase 2: CLI wrapper
- [x] Phase 3: Validation
