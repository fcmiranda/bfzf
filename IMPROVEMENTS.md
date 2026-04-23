# bfzf — Improvement Ideas (fzf feature parity)

This document tracks fzf features not yet in bfzf, ordered roughly by value-to-effort ratio.

---

## High priority

### 1. `--reverse` — list order ✅ DONE
fzf can render the list top-to-bottom with the cursor starting at the top, or
bottom-to-top (default in many setups) with the input at the bottom.
bfzf always renders input-then-list top-to-bottom.

**Scope**: Add `WithReverse() Option` that flips the viewport content rendering order
and positions the cursor at the top. Change scroll direction in `ensureCursorVisible`.

---

### 2. `--bind` — dynamic, runtime key binding ✅ DONE
fzf's `--bind key:action` is one of its most powerful features:
reload items, execute shell commands, toggle UI panels, change query, etc.

**Scope**: Design a `BindAction` interface with implementations such as
`Execute(cmd)`, `ReloadItems(fn)`, `ChangeQuery(s)`, `TogglePreview()`,
`ToggleSort()`. Expose `WithBind(key, action)` and `--bind key:action` CLI flag.

---

### 3. `--query STRING` — initial pre-filled filter ✅ DONE
fzf starts with `--query` already typed in the search box, enabling scripts to
pre-filter to a likely match.

**Scope**: `WithQuery(s string) Option` that calls `m.input.SetValue(s)` and runs
`m.updateFilter()` in `New()`.  CLI: `--query`.

---

### 4. `--exact` / exact-match tokens ✅ DONE
fzf supports `'token` (exact), `!token` (negate), `^prefix`, `suffix$` as special
search operators within the query string.

**Scope**: Pre-process the query string in `buildVisibleFiltered` to extract and
apply these operators before/after the fuzzy pass.

---

### 5. `--read0` / `--print0` — NUL-delimited I/O ✅ DONE
fzf supports NUL (`\0`) as the record separator on both input and output, making
it safe for filenames containing newlines. bfzf already has `--0` for input but
does not expose a `--print0` flag for output.

**Scope**: Add `--print0` flag in `cmd/bfzf/main.go`; write `\0` between selected
items instead of `\n`.

---

### 6. `--header-lines N` — treat first N input lines as a non-scrolling header ✅ DONE
fzf can pin the first N input lines as column headers above the list.

**Scope**: Parse the first N items as `HeaderItem`s (or a dedicated `PinnedHeader`
type), render them outside the scrollable viewport, and exclude them from
fuzzy matching.

---

### 7. `--preview-window hidden` + toggle ✅ DONE
In fzf, `--preview-window hidden` starts with the preview hidden and `ctrl+/`
toggles it.  bfzf always shows the preview when `WithPreview` is set.

**Scope**: Add `hidePreview bool` field + `KeyMap.TogglePreview` keybind.
Options: `WithPreviewHidden()`, `--preview-window hidden`, `--toggle-preview-key`.

---

### 8. `--pointer STR` / `--marker STR` as single-char flags ✅ DONE
fzf exposes `--pointer` (cursor row indicator) and `--marker` (selected-item glyph)
as direct string flags separate from a style preset.  bfzf already has `--cursor`
and `--marker` but `--marker` takes a style name rather than the raw glyphs.

**Scope**: Accept custom glyph strings:
`--marker-selected '▶'` + `--marker-unselected ' '`.

---

## Medium priority

### 9. `--info [default|right|hidden|inline]` — match count / position info
fzf can show the match info line in different positions: below the input (default),
inline at the right edge of the input, or hidden.  bfzf always shows the help line
at the bottom.

**Scope**: Add `WithInfoStyle(s InfoStyle) Option` with three modes; move the info
string render site accordingly.

---

### 10. `--layout [default|reverse|reverse-list]` — cursor-at-top variants
Complement of `--reverse`.  `reverse-list` keeps input at top but renders list
with newest matches at top (fzf's default behaviour when used with `--reverse`).

---

### 11. Regex search mode (`//` prefix)
fzf activates regex mode when the query starts with `/`.
Implementing regex alongside the existing fuzzy pass would unblock power users.

---

### 12. `--no-clear` / `--keep-right`
fzf's `--no-clear` preserves a line above the picker for existing terminal content.
`--keep-right` keeps the right end of long lines visible in the input box.

---

### 13. Horizontal scrolling in list rows
Long item labels are currently truncated at the viewport edge.  fzf scrolls long
rows horizontally.

**Scope**: Implement per-row horizontal clipping with `ansi.Truncate` and a
`KeyMap.ScrollRight` / `ScrollLeft` binding when a row overflows.

---

### 14. `--with-nth FIELD` — display only a subset of fields
fzf can show `{2..}` of an item in the list while using the full label for fuzzy
matching.  `WithDisplayField(template)` would enable e.g. showing only the
filename while matching the full path.

---

### 15. `--tiebreak` — configurable sort tie-breaking
fzf can break fuzzy-score ties by `length`, `begin`, `end`, `chunk`, or `index`.
Currently bfzf uses `sahilm/fuzzy` score order.

---

### 16. `--delimiter REGEXP` — regex field delimiter
fzf's `--delimiter` accepts a regexp.  bfzf only accepts a literal string.
Switch `readLines` to `bufio.Scanner` with a regexp-based split function.

---

### 17. Colors: `--no-color` and 256/24-bit auto-detection
fzf disables colors automatically when the terminal reports < 8 colors.  bfzf
always emits ANSI codes.  Add `WithAutoColor(bool)` and `--no-color` flag.

---

### 18. `--black` — force black background
Convenience pre-set that sets `bg` and `bg+` to the default terminal background
without specifying a color explicitly.

---

### 19. `--border [rounded|sharp|bold|block|double|horizontal|vertical|none]`
fzf's `--border` wraps the entire picker in a single outer border (not just the
list or preview panes individually).  bfzf currently only borders individual panes.

**Scope**: Wrap `render()` output in a top-level lipgloss border style.
Option: `WithOuterBorder(b lipgloss.Border)` / `--border TYPE`.

---

### 20. `--prompt` / `--pointer` width auto-detection
Use `lipgloss.Width()` to measure custom prompts so item indentation stays aligned
regardless of whether the prompt is ASCII (`> `) or multi-byte (`❯ `).

---

## Low priority / nice to have

### 21. `--filepath-word`
When enabled, `ctrl+w` in the textinput deletes one path segment (`/word`) instead
of a whole word.

---

### 22. `--listen PORT` — HTTP API
fzf can be controlled remotely over HTTP.  A lightweight HTTP handler could
accept `{"action":"reload","items":[...]}` payloads via `tea.Cmd` channels.

---

### 23. `--sync` — synchronous initial rendering
fzf's `--sync` holds rendering until all items are loaded.  bfzf already blocks
in `New()`, but streaming / lazy item loading is not yet supported.

---

### 24. Streaming item source (`WithItemSource(ch <-chan Item)`)
Allowing the caller to push items incrementally (e.g. from a slow command) while
the picker is already interactive, matching fzf's `--listen` / reload behaviour.

---

### 25. Mouse support beyond scroll: click to select
Currently mouse events only trigger preview scrolling.  Clicking a list row should
move the cursor; double-click should submit.

---

### 26. `--cycle` — wrap cursor at ends
fzf's `--cycle` wraps the cursor from the last item back to the first (and vice
versa).  bfzf already wraps — document this as already implemented.

---

### 27. `--hscroll` / `--hscroll-off` — fine-tune horizontal scroll
When horizontal scrolling (#13) is implemented, `--hscroll-off N` controls how
many characters to keep visible to the left of the matched region.

---

### 28. `--tabstop N` — tab character width
fzf replaces `\t` in labels with spaces; the tab-stop column width is
configurable via `--tabstop`.  Useful for tabular data.

---

### 29. `--preview-label` / `--border-label`
fzf can show a custom label in the border of the preview pane or the outer border,
distinct from the item title.  bfzf uses the item label; a separate
`WithPreviewLabel(s)` would allow a static string.

---

### 30. `--input-label`
fzf can show a label inside the input area border (e.g. `SEARCH`).  Expose as
`WithInputLabel(s string)` displayed before the prompt character.

---

### 31. Ansi-hyperlink passthrough
fzf passes OSC 8 hyperlink sequences through to the terminal.  bfzf strips them
with `ansi.Strip`.  Add a `WithHyperlinks()` option that skips stripping.

---

### 32. `--disabled` — show all items, highlight matches, do not filter
In disabled mode the list is never narrowed; fuzzy matches are merely highlighted.
Useful when bfzf is used as a navigator, not a filter.

---

### 33. `--jump` / `--jump-labels` — EasyMotion-style cursor jump
After pressing the jump key, each visible row is annotated with a single letter;
pressing the letter jumps the cursor directly to that row.

---

### 34. Undo / redo in query
Standard `ctrl+z` / `ctrl+y` undo/redo for the search query, in addition to the
single-character undo already provided by the `textinput` component.

---

### 35. `--ghost` — placeholder text reflecting first match
fzf can ghost-complete the top match into the input field in dim text, like
browser URL-bar completion.

---

_Last updated: 2026-04-23_

---

## Completed extras

### Input filter width parameter ✅ DONE
Allow callers to constrain the search input to a specific column count,
enabling parent layouts where other UI elements share the same row.

`WithInputWidth(n int) Option` and `--input-width N` CLI flag.
