package bfzf

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
)

// ────────────────────────────────────────────────────────────────────────────
// Preview public API
// ────────────────────────────────────────────────────────────────────────────

// PreviewFunc is a function that computes the preview content for a focused
// item. It runs inside a [tea.Cmd] goroutine, so it may safely block (e.g.
// reading a file or running a subprocess).
type PreviewFunc func(item Item) string

// PreviewPosition controls the placement of the preview pane.
type PreviewPosition int

const (
	// PreviewRight places the preview panel to the right of the list (default).
	PreviewRight PreviewPosition = iota
	// PreviewBottom places the preview panel below the list.
	PreviewBottom
)

// ────────────────────────────────────────────────────────────────────────────
// Internal preview message
// ────────────────────────────────────────────────────────────────────────────

// previewResultMsg carries preview content back to the model.
// itemIdx mirrors the originating item so stale responses can be discarded
// if the cursor moved while the preview was computing.
type previewResultMsg struct {
	content string
	itemIdx int
}

// ────────────────────────────────────────────────────────────────────────────
// Preview helpers used by Model
// ────────────────────────────────────────────────────────────────────────────

// triggerPreview emits a tea.Cmd that computes the preview for the currently
// focused item. Returns nil if:
//   - no preview function is set,
//   - there are no selectable items, or
//   - the focused item is the same as the last triggered item.
func (m *Model) triggerPreview() tea.Cmd {
	if m.previewFunc == nil || len(m.selectableIdxs) == 0 || m.cursorPos < 0 {
		m.lastPreviewIdx = -1
		return nil
	}
	idx := m.selectableIdxs[m.cursorPos]
	if idx == m.lastPreviewIdx {
		return nil // same item — no need to re-run
	}
	m.lastPreviewIdx = idx
	item := m.items[idx]
	fn := m.previewFunc
	return func() tea.Msg {
		return previewResultMsg{content: fn(item), itemIdx: idx}
	}
}

// initPreview prepares the preview viewport; called from New() when a
// preview function is provided.
func initPreview() viewport.Model {
	return viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
}
