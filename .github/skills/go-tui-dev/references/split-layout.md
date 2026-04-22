# Split Layout Reference

Design guide and checklist for right-split and bottom-split preview panes.

---

## Mental Model

```
┌─ render() ──────────────────────────────────────────────── m.width ─┐
│  inputView          ← full width, rendered ABOVE the split           │
│  ┌─ listPane ──┐  │  ┌─ previewPane ──────────────────────────────┐  │
│  │ title       │  │  │ previewTitle          lineCount            │  │
│  │ vp.View()   │  │  │ previewVP.View()                           │  │
│  └─────────────┘  │  └────────────────────────────────────────────┘  │
│  helpStr            ← full width, rendered BELOW the split            │
└─────────────────────────────────────────────────────────────────────┘
```

**Critical invariant**: `inputView` must never be a child of `listPane`.
If it is, `JoinHorizontal(listPane, previewPane)` = `m.width + previewAreaW` → overflow.

---

## resize() Template

```go
func (m *Model) resize() {
    if m.width == 0 || m.height == 0 { return }

    // Fixed rows consumed outside the list viewport.
    fixedRows := helpLines                    // 1
    if !m.hideInput  { fixedRows += inputLines } // +1
    if m.listTitle != "" { fixedRows++ }         // +1 for title row

    // Border frame sizes — 0 when borders are off.
    listBorderH := 0
    if m.showListBorder {
        listBorderH = m.styles.ListBorder.GetHorizontalFrameSize()
    }
    prevBorderH, prevBorderV := 0, 0
    if m.showPreviewBorder {
        prevBorderH = m.styles.PreviewBorder.GetHorizontalFrameSize()
        prevBorderV = m.styles.PreviewBorder.GetVerticalFrameSize()
    }

    if !m.hideInput { m.input.SetWidth(m.width) }

    if m.previewFunc == nil {
        vpH := max(minVPLines, m.height-fixedRows)
        vpW := max(1, m.width-listBorderH)
        m.vp.SetHeight(vpH)
        m.vp.SetWidth(vpW)
        m.vp.SetContent(m.renderList())
        return
    }

    const previewTitleRows = 1

    switch m.previewPos {
    case PreviewRight:
        vpH := max(minVPLines, m.height-fixedRows)
        listAreaW := m.width * (100 - m.previewSize) / 100
        sepW := 1                          // bar character
        if m.showPreviewBorder { sepW = 0} // border replaces bar
        prevAreaW := m.width - listAreaW - sepW
        if prevAreaW < 8 { prevAreaW = 8; listAreaW = m.width - prevAreaW - sepW }

        m.vp.SetHeight(vpH)
        m.vp.SetWidth(max(1, listAreaW-listBorderH))           // ← subtract border
        m.previewVP.SetHeight(max(minVPLines, vpH-previewTitleRows-prevBorderV))
        m.previewVP.SetWidth(max(8, prevAreaW-prevBorderH))    // ← subtract border

    case PreviewBottom:
        totalH := max(2, m.height-fixedRows-1) // -1 for ─── separator row
        listAreaH := totalH * (100 - m.previewSize) / 100
        prevAreaH := totalH - listAreaH

        m.vp.SetHeight(max(minVPLines, listAreaH))
        m.vp.SetWidth(max(1, m.width-listBorderH))
        m.previewVP.SetHeight(max(minVPLines, prevAreaH-previewTitleRows-prevBorderV))
        m.previewVP.SetWidth(max(8, m.width-prevBorderH))
    }

    m.vp.SetContent(m.renderList())
}
```

---

## render() Template

```go
func (m *Model) render() string {
    // Compact join: skip empty strings.
    join := func(parts ...string) string {
        var out []string
        for _, p := range parts { if p != "" { out = append(out, p) } }
        return strings.Join(out, "\n")
    }

    var inputView string
    if !m.hideInput { inputView = m.input.View() }

    helpStr   := m.renderHelp()
    listPane  := m.renderListPane()

    if m.previewFunc == nil {
        return join(inputView, listPane, helpStr)
    }

    previewPane := m.renderPreviewPane()

    switch m.previewPos {
    case PreviewRight:
        var split string
        if m.showPreviewBorder {
            split = lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
        } else {
            h := m.vp.Height()
            bar := m.styles.PreviewBorder.Render(strings.Repeat("│\n", h-1) + "│")
            split = lipgloss.JoinHorizontal(lipgloss.Top, listPane, bar, previewPane)
        }
        return join(inputView, split, helpStr)

    case PreviewBottom:
        sep := m.styles.PreviewBorder.Render(strings.Repeat("─", m.width))
        return join(inputView, listPane, sep, previewPane, helpStr)
    }

    return join(inputView, listPane, helpStr)
}
```

---

## renderListPane() Template

```go
// renderListPane — NO input widget inside.
func (m *Model) renderListPane() string {
    var parts []string
    if m.listTitle != "" {
        parts = append(parts, m.styles.ListTitle.Render(m.listTitle))
    }
    parts = append(parts, m.vp.View())
    content := strings.Join(parts, "\n")
    if m.showListBorder {
        return m.styles.ListBorder.Render(content)
    }
    return content
}
```

---

## Async Preview Pattern

```go
// preview.go

type PreviewFunc func(item Item) string

type previewResultMsg struct {
    content string
    itemIdx int
}

func (m *Model) triggerPreview() tea.Cmd {
    if m.previewFunc == nil || len(m.selectableIdxs) == 0 { return nil }
    idx := m.selectableIdxs[m.cursorPos]
    if idx == m.lastPreviewIdx { return nil } // no-op: same item
    m.lastPreviewIdx = idx
    item := m.items[idx]
    fn := m.previewFunc
    return func() tea.Msg {
        return previewResultMsg{content: fn(item), itemIdx: idx}
    }
}

// In Update:
case previewResultMsg:
    if msg.itemIdx == m.lastPreviewIdx { // stale-check
        m.previewVP.SetContent(msg.content)
    }
```

---

## Preview Subprocess (CLI)

```go
// Pass colour env so tools like bat/ls output ANSI.
cmd := exec.Command("sh", "-c", expandedCmd)
cmd.Env = append(os.Environ(),
    "TERM=xterm-256color",
    "COLORTERM=truecolor",
    "CLICOLOR_FORCE=1",
    "FORCE_COLOR=3",
)
out, err := cmd.CombinedOutput()
```

---

## Common Bugs

| Symptom | Root Cause | Fix |
|---------|-----------|-----|
| Preview pane not visible / layout overflows terminal | `input.View()` inside `renderListPane()` — adds `m.width` to horizontal join | Move input above `JoinHorizontal` in `render()` |
| Preview pane clips content by 2 cols | `prevAreaW` not reduced by border frame | Subtract `GetHorizontalFrameSize()` when `showPreviewBorder` |
| Preview shows stale content after cursor jump | No stale-check on async result | Guard with `msg.itemIdx == m.lastPreviewIdx` |
| `{}` in preview cmd expands ANSI garbage | Item label contains ANSI from rendering | Strip ANSI with `regexp` before shell expansion |
