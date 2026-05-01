# bfzf Preview Pane — Bug Fixes & Root Cause Analysis

> Reference document for the Go implementation and future Rust port.

---

## Fix 1 — Preview pane height shifts when cycling items

### Symptom

Navigating between items caused the entire split layout to jump vertically by
one row. The list pane and preview pane would briefly misalign on every cursor
movement.

### Root cause

`renderPreviewPane()` (`bfzf.go`) builds the no-border preview by concatenating
a dynamic title row above the viewport string:

```go
// BEFORE (buggy)
titleRow := titleStr + lineCount
return titleRow + "\n" + vpView
```

`titleStr` is the selected item's label; `lineCount` is a `"N/M"` scroll
indicator that disappears when the viewport has no content (`TotalLineCount() == 0`).

Because `lineCount` toggled between `""` and a non-empty badge as different
items were selected, the **rendered column width of `titleRow` varied
per-item**. When `lipgloss.JoinHorizontal` assembled the list pane and preview
pane side by side, lipgloss padded the shorter column to match the taller one.
A width change in `titleRow` caused the whole split block to be recalculated,
producing a one-row height shift.

### Fix

Wrap `titleRow` in a `lipgloss.NewStyle().Width(n).Render()` call so that it is
always padded/clipped to exactly the preview viewport's column width, regardless
of whether `lineCount` is empty or populated:

```go
// AFTER (fixed)
titleRow := lipgloss.NewStyle().Width(m.previewVP.Width()).Render(titleStr + lineCount)
return titleRow + "\n" + vpView
```

**File changed:** `bfzf.go` — `renderPreviewPane()`

### Key insight for the Rust port

Any fixed-width split layout must ensure every sub-pane string has a
**constant column width on every render frame**, regardless of content. When
using a layout library that joins panes by padding to the tallest/widest, a
single variable-width row anywhere in a pane will cause the whole layout to
reflow. The safe pattern is: always render header/footer rows through a
fixed-width formatter before joining.

---

## Fix 2 — Preview content word-wraps inside the viewport

### Symptom

Long lines in the preview pane were broken at word boundaries mid-line, e.g.:

```
...trims each line to exactly the preview pan
e layout.
```

The word `pane` was split across two lines. fzf showed the same content on a
single unbroken line.

### Root cause

The Bubble Tea `viewport.Model.View()` function rendered its visible lines by
joining them with `\n` and passing the entire block to:

```go
// BEFORE (buggy) — inside bubbles/viewport/viewport.go
contents := lipgloss.NewStyle().
    Width(contentWidth).   // pad to width.
    Height(contentHeight). // pad to height.
    Render(strings.Join(m.visibleLines(), "\n"))
```

The `Width(n)` property in lipgloss **does two things**: it pads short content
to `n` columns AND it **word-wraps** content longer than `n` columns. Internally
it calls:

```go
// lipgloss/style.go
wrapAt := width - leftPadding - rightPadding
str = Wrap(str, wrapAt, "")          // ansi.Wrap → word-wrap pass
```

`ansi.Wrap` implements a **word-boundary wrap algorithm**: when a word would
cause the current line to exceed `limit` columns, it inserts a newline _before_
that word. This can split a word across lines when the word falls exactly at the
column boundary (e.g. `"pan"` fits, but `"pane"` does not → newline inserted
before `"e"`).

`bfzf`'s `truncatePreviewLines()` correctly truncates each line to
`m.previewVP.Width()` before calling `SetContent`, so no individual line should
be wider than the viewport. However, because `visibleLines()` returns lines that
have already been cut, and those lines are then **re-processed as a single joined
string** by lipgloss, the wrap algorithm re-evaluates word boundaries across the
entire block. An ANSI-decorated line that is visually exactly `contentWidth`
columns can still trigger a wrap if the word-wrap algorithm's internal cursor
lands at a boundary.

### Why fzf does not have this problem

fzf allocates the preview area as a **low-level terminal window** (a fixed cell
grid). All rendering writes characters directly into that grid, cell by cell.
If content is shorter than the window width, the remaining cells are cleared.
If content is wider, the extra characters are simply not written. No
string-joining, no lipgloss layout pass, no wrap algorithm is involved.

### Fix

Replace the single `lipgloss.NewStyle().Width(n).Render(joinedLines)` call with
per-line processing that hard-truncates first and then pads — keeping the outer
`Height` render only for vertical padding:

```go
// AFTER (fixed) — bubbles/viewport/viewport.go
contentWidth := w - m.Style.GetHorizontalFrameSize()
contentHeight := h - m.Style.GetVerticalFrameSize()

// Process each line individually: truncate lines that exceed contentWidth
// (preventing lipgloss word-wrap) and pad short lines to contentWidth for
// consistent background coverage.
visLines := m.visibleLines()
linePad := lipgloss.NewStyle().Width(contentWidth)
for i, line := range visLines {
    if ansi.StringWidth(line) > contentWidth {
        line = ansi.Truncate(line, contentWidth, "")
    }
    visLines[i] = linePad.Render(line)
}
contents := lipgloss.NewStyle().
    Height(contentHeight). // pad to height only — no wrap.
    Render(strings.Join(visLines, "\n"))
```

**File changed:** `bubbles/viewport/viewport.go` — `View()`

### Key insight for the Rust port

`ratatui` (the likely Rust TUI library) uses a `Paragraph` widget with a
`Wrap` option. **Do not enable `Wrap` on the preview paragraph.** Instead,
hard-truncate each line to the pane width before passing it to the widget. The
safe rendering contract for a no-wrap preview pane is:

1. Truncate each line to `pane_width` columns (ANSI-aware, e.g. `strip_ansi` +
   `unicode_width`, or the `ansi_str` crate).
2. Pad each line to exactly `pane_width` columns with spaces (so background
   colour fills the row).
3. Assemble lines with `\n` and render with height-only clamping.
4. Never pass the joined block through any word-wrap pass.

---

## Fix 3 — Bordered preview still word-wraps (lipgloss v2 `Width` is outer, not inner)

### Symptom

After Fix 2, no-border preview was correct, but the bordered layout (`--border="rounded"`)
still word-wrapped long lines at the right edge of the preview pane. e.g.:

```
A practical reference demonstrating the breadth of `bfzf` in shell pipelines, scripts, and
Go
```

The word `Go` (end of the line) was pushed to a new line.

### Root cause

`titledBorder()` renders the preview viewport string inside the border by calling:

```go
rendered := style.Render(content)  // style = m.styles.PreviewBorder
```

`m.styles.PreviewBorder` has an explicit `Width` set in `resize()`:

```go
// BEFORE (buggy)
m.styles.PreviewBorder = m.styles.PreviewBorder.Width(m.previewVP.Width())
```

The intent (documented in a comment) was to set the "inner width" so the border
always spans the full area. However, **in lipgloss v2 `Width(n)` is the total
outer width** (content + borders). Internally, before word-wrapping, lipgloss
subtracts the horizontal border frame:

```go
// lipgloss/style.go
width -= horizontalBorderSize   // 2 for a single-char left+right border
wrapAt := width - leftPadding - rightPadding
str = Wrap(str, wrapAt, "")     // ansi.Wrap — word-boundary algorithm
```

So with `Width(previewVP.Width())`:
- `wrapAt = previewVP.Width() - 2`
- content lines are `previewVP.Width()` columns wide
- `previewVP.Width() > wrapAt` → every line that fills the viewport triggers a wrap

The off-by-`horizontalBorderSize` (2) was enough to break the last 1–2 words
onto a new line for any line that reached the right edge of the pane.

### Fix

Set `PreviewBorder.Width` to the **outer area width** (`prevAreaW` for
`PreviewRight`, `effW` for `PreviewBottom`). After lipgloss subtracts the border
frame, `wrapAt` equals `previewVP.Width()` — matching exactly the width the
viewport produces:

```go
// AFTER (fixed) — bfzf.go resize(), PreviewRight case
if m.showPreviewBorder {
    m.styles.PreviewBorder = m.styles.PreviewBorder.Width(prevAreaW)
}

// AFTER (fixed) — bfzf.go resize(), PreviewBottom case
if m.showPreviewBorder {
    m.styles.PreviewBorder = m.styles.PreviewBorder.Width(effW)
}
```

**File changed:** `bfzf.go` — `resize()`, both `PreviewRight` and `PreviewBottom` branches.

### Key insight for the Rust port

In **ratatui**, `Block` (the border widget) does not wrap its inner content — it
simply draws the border frame and the inner area is sized by the layout. There is
no equivalent "outer vs. inner width" ambiguity. However, if you use a `Paragraph`
inside a `Block` and set a `Wrap` option, wrapping will still occur. The safe
pattern is: size the `Paragraph` to `inner_area.width` (the area returned by
`block.inner(area)`) and do **not** enable `Wrap`. Pass pre-truncated lines
(see Fix 2) of exactly `inner_area.width` columns.

---

## Summary table

| # | File | Location | Problem | Fix |
|---|------|----------|---------|-----|
| 1 | `bfzf.go` | `renderPreviewPane()` | Title row width varies per item → layout reflows | Wrap title row with `lipgloss.NewStyle().Width(previewVP.Width()).Render(...)` |
| 2 | `bubbles/viewport/viewport.go` | `View()` | `lipgloss.Width(n).Render(block)` word-wraps content | Process lines individually: truncate then pad per-line; outer render uses `Height` only |
| 3 | `bfzf.go` | `resize()` (both preview positions) | `PreviewBorder.Width` set to inner width; lipgloss v2 treats `Width` as outer → `wrapAt = inner - borderSize` | Set `PreviewBorder.Width` to the outer area width (`prevAreaW` / `effW`) |
