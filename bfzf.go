// Package bfzf provides an fzf-inspired fuzzy picker built on Bubble Tea,
// Bubbles, and Lip Gloss.
//
// Features:
//   - Real-time fuzzy search with match highlighting
//   - Grouped options: non-selectable headers visually separate groups
//   - Per-option animated Bubble spinner support
//   - Multi-select with configurable limit
//   - Fully customisable styles and key bindings
//
// Basic usage:
//
//	items := []bfzf.Item{
//	    bfzf.NewHeader("Fruits"),
//	    bfzf.NewItem("Apple"),
//	    bfzf.NewItem("Banana"),
//	}
//	m := bfzf.New(items, bfzf.WithHeight(12))
//	p := tea.NewProgram(m, tea.WithAltScreen())
//	final, _ := p.Run()
//	if fm, ok := final.(bfzf.Model); ok && fm.Submitted() {
//	    fmt.Println(fm.Selected()[0].Label())
//	}
package bfzf

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

// ────────────────────────────────────────────────────────────────────────────
// Public interfaces
// ────────────────────────────────────────────────────────────────────────────

// Item is anything that can appear in the picker list.
type Item interface {
	// Label is the text displayed for this entry.
	Label() string
	// FilterValue is the string used for fuzzy matching.
	// Return an empty string to make the item un-searchable (e.g. group headers).
	FilterValue() string
	// IsHeader reports whether this item is a non-selectable group header.
	IsHeader() bool
}

// SpinnerItem is an optional extension of [Item].
// Items implementing SpinnerItem will have an animated spinner rendered
// to the left of their label.
type SpinnerItem interface {
	Item
	// Spinner returns the initial spinner.Model configuration for this item.
	// The caller only provides the initial state; bfzf drives the animation.
	Spinner() spinner.Model
}

// ────────────────────────────────────────────────────────────────────────────
// Convenience concrete types
// ────────────────────────────────────────────────────────────────────────────

// SimpleItem is a basic selectable option.
type SimpleItem struct{ Text string }

func (s SimpleItem) Label() string       { return s.Text }
func (s SimpleItem) FilterValue() string { return s.Text }
func (s SimpleItem) IsHeader() bool      { return false }

// HeaderItem is a non-selectable group divider shown between groups of items.
// Headers are always visible when the search query is empty and disappear (are
// hidden) only when no items in their group match the query.
type HeaderItem struct{ Text string }

func (h HeaderItem) Label() string       { return h.Text }
func (h HeaderItem) FilterValue() string { return "" }
func (h HeaderItem) IsHeader() bool      { return true }

// NewItem creates a [SimpleItem].
func NewItem(text string) SimpleItem { return SimpleItem{Text: text} }

// NewHeader creates a [HeaderItem].
func NewHeader(text string) HeaderItem { return HeaderItem{Text: text} }

// ────────────────────────────────────────────────────────────────────────────
// Internal types
// ────────────────────────────────────────────────────────────────────────────

// visibleEntry is a single row currently shown in the viewport.
type visibleEntry struct {
	itemIdx     int   // index into Model.items
	matchedIdxs []int // fuzzy match positions for highlighting (nil = no highlights)
	isHeader    bool
}

// ────────────────────────────────────────────────────────────────────────────
// Model
// ────────────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea component.  Implement tea.Model.
type Model struct {
	// ── Public options (readable/writable before first Update) ──────────────

	// Placeholder is shown in the search box when empty.
	Placeholder string
	// Prompt is the search box prompt string (default "❯ ").
	Prompt string
	// Limit is the selection limit. 0 = unlimited multi-select, 1 = single-select.
	Limit int

	// ── Sub-components ───────────────────────────────────────────────────────

	input textinput.Model
	vp    viewport.Model

	// ── Source data ──────────────────────────────────────────────────────────

	items   []Item
	// spinners stores live spinner state keyed by item index.
	// Populated at construction time from items implementing SpinnerItem.
	spinners map[int]spinner.Model

	// ── Computed on each filter operation ───────────────────────────────────

	visible        []visibleEntry // all rows to render (headers + selectables in order)
	selectableIdxs []int          // item indices of visible non-header entries

	// ── Navigation ───────────────────────────────────────────────────────────

	// cursorPos is the index into selectableIdxs pointing to the highlighted item.
	cursorPos int
	// selected is the set of selected item indices (multi-select).
	selected map[int]struct{}

	// ── Terminal dimensions ──────────────────────────────────────────────────

	width  int
	height int
	ready  bool // true once we know the terminal size

	// ── Output ───────────────────────────────────────────────────────────────

	quitting  bool
	submitted bool

	// ── Customisation ────────────────────────────────────────────────────────

	styles Styles
	keymap KeyMap
}

// New creates a new Model with the provided items and options.
//
// Items are displayed in the order given.  [HeaderItem]s act as non-selectable
// section dividers.  Any item implementing [SpinnerItem] will have its spinner
// animated during the session.
//
// Call [WithHeight] / [WithWidth] if you embed the component inside a larger
// program; omit them for full-screen usage (the model resizes on
// tea.WindowSizeMsg).
func New(items []Item, opts ...Option) Model {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Prompt = "❯ "

	m := Model{
		Prompt:         "❯ ",
		Placeholder:    "Filter...",
		Limit:          1,
		items:          items,
		spinners:       make(map[int]spinner.Model),
		selected:       make(map[int]struct{}),
		selectableIdxs: make([]int, 0, len(items)),
		cursorPos:      0,
		input:          ti,
		vp:             viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
		styles:         DefaultStyles(),
		keymap:         DefaultKeyMap(),
	}

	// Extract spinner configs from items implementing SpinnerItem.
	for i, item := range items {
		if si, ok := item.(SpinnerItem); ok {
			s := si.Spinner()
			// Ensure the spinner has a valid preset; fall back to Dot.
			if s.Spinner.FPS == 0 {
				s.Spinner = spinner.Dot
			}
			m.spinners[i] = s
		}
	}

	// Apply functional options.
	for _, opt := range opts {
		opt(&m)
	}

	// Sync input fields after options may have changed them.
	m.input.Placeholder = m.Placeholder
	m.input.Prompt = m.Prompt

	// Compute the initial (unfiltered) visible set.
	m.buildVisibleAll()

	return m
}

// ────────────────────────────────────────────────────────────────────────────
// tea.Model implementation
// ────────────────────────────────────────────────────────────────────────────

// Init implements [tea.Model].
func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 1+len(m.spinners))
	cmds = append(cmds, m.input.Focus())
	for _, s := range m.spinners {
		cmds = append(cmds, s.Tick)
	}
	return tea.Batch(cmds...)
}

// Update implements [tea.Model].
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.ready = true

	case tea.KeyPressMsg:
		switch {
		case matchesAny(msg, m.keymap.Abort):
			m.quitting = true
			return m, tea.Interrupt

		case matchesAny(msg, m.keymap.Quit):
			m.quitting = true
			return m, tea.Quit

		case matchesAny(msg, m.keymap.Submit):
			if len(m.selectableIdxs) > 0 {
				// In single-select mode auto-mark the cursor item.
				if m.Limit == 1 && m.cursorPos >= 0 {
					m.selected[m.selectableIdxs[m.cursorPos]] = struct{}{}
				}
				m.submitted = true
				m.quitting = true
				return m, tea.Quit
			}

		case matchesAny(msg, m.keymap.Down):
			m.moveCursorDown()

		case matchesAny(msg, m.keymap.Up):
			m.moveCursorUp()

		case matchesAny(msg, m.keymap.Home):
			m.moveCursorToStart()

		case matchesAny(msg, m.keymap.End):
			m.moveCursorToEnd()

		case matchesAny(msg, m.keymap.Toggle):
			if m.Limit != 1 {
				m.toggleSelection()
			}

		case matchesAny(msg, m.keymap.ToggleAndNext):
			if m.Limit != 1 {
				m.toggleSelection()
				m.moveCursorDown()
			}

		case matchesAny(msg, m.keymap.ToggleAndPrev):
			if m.Limit != 1 {
				m.toggleSelection()
				m.moveCursorUp()
			}

		case matchesAny(msg, m.keymap.SelectAll):
			if m.Limit == 0 {
				m.selectAll()
			}

		default:
			// Pass unhandled keys to the text input.
			prev := m.input.Value()
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			if inputCmd != nil {
				cmds = append(cmds, inputCmd)
			}
			if m.input.Value() != prev {
				m.updateFilter()
			}
		}
	}

	// Update text input for non-key messages (cursor blink, etc.).
	if _, isKey := msg.(tea.KeyPressMsg); !isKey {
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		if inputCmd != nil {
			cmds = append(cmds, inputCmd)
		}
	}

	// Update all spinners on every message.
	for i, s := range m.spinners {
		updated, cmd := s.Update(msg)
		m.spinners[i] = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Sync viewport content after any state change.
	m.vp.SetContent(m.renderList())
	m.ensureCursorVisible()

	return m, tea.Batch(cmds...)
}

// View implements [tea.Model].
func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}
	return tea.NewView(m.render())
}

// ────────────────────────────────────────────────────────────────────────────
// Public accessors (valid after the program exits)
// ────────────────────────────────────────────────────────────────────────────

// Selected returns the items chosen by the user.
// In single-select mode it contains one item (the cursor item on submit).
// In multi-select mode it contains all toggled items.
// Only meaningful when [Submitted] returns true.
func (m Model) Selected() []Item {
	result := make([]Item, 0, len(m.selected))
	// Preserve original order.
	for i, item := range m.items {
		if _, ok := m.selected[i]; ok {
			result = append(result, item)
		}
	}
	return result
}

// Submitted reports whether the user confirmed the selection.
func (m Model) Submitted() bool { return m.submitted }

// Quitting reports whether the model is done (submitted or quit/aborted).
func (m Model) Quitting() bool { return m.quitting }

// ────────────────────────────────────────────────────────────────────────────
// Filter helpers
// ────────────────────────────────────────────────────────────────────────────

func (m *Model) updateFilter() {
	if m.input.Value() == "" {
		m.buildVisibleAll()
	} else {
		m.buildVisibleFiltered(m.input.Value())
	}
	// Clamp cursor within the new visible set.
	if len(m.selectableIdxs) == 0 {
		m.cursorPos = -1
		return
	}
	if m.cursorPos < 0 || m.cursorPos >= len(m.selectableIdxs) {
		m.cursorPos = 0
	}
}

// buildVisibleAll populates visible with every item (no filter, no highlights).
func (m *Model) buildVisibleAll() {
	m.visible = m.visible[:0]
	m.selectableIdxs = m.selectableIdxs[:0]
	for i, item := range m.items {
		m.visible = append(m.visible, visibleEntry{
			itemIdx:  i,
			isHeader: item.IsHeader(),
		})
		if !item.IsHeader() {
			m.selectableIdxs = append(m.selectableIdxs, i)
		}
	}
}

// buildVisibleFiltered runs fuzzy matching and re-populates visible.
// Headers missing any matching descendant are hidden.
func (m *Model) buildVisibleFiltered(query string) {
	// Collect selectable items with their item indices.
	type sel struct {
		itemIdx int
		label   string
	}
	selectables := make([]sel, 0, len(m.items))
	for i, item := range m.items {
		if !item.IsHeader() {
			selectables = append(selectables, sel{i, item.FilterValue()})
		}
	}
	if len(selectables) == 0 {
		m.visible = m.visible[:0]
		m.selectableIdxs = m.selectableIdxs[:0]
		return
	}

	// Run fuzzy matching on FilterValues.
	labels := make([]string, len(selectables))
	for i, s := range selectables {
		labels[i] = s.label
	}
	matches := fuzzy.Find(query, labels)

	// Build match index map: item index -> matched character positions.
	type matchData struct{ idxs []int }
	matched := make(map[int]matchData, len(matches))
	for _, fm := range matches {
		itemIdx := selectables[fm.Index].itemIdx
		matched[itemIdx] = matchData{fm.MatchedIndexes}
	}

	// Build header-for-item map: item index -> preceding header index (-1 if none).
	headerFor := make(map[int]int, len(m.items))
	lastHeader := -1
	for i, item := range m.items {
		if item.IsHeader() {
			lastHeader = i
		} else {
			headerFor[i] = lastHeader
		}
	}

	// Collect headers that have at least one matched child.
	neededHeaders := make(map[int]struct{})
	for itemIdx := range matched {
		if h := headerFor[itemIdx]; h >= 0 {
			neededHeaders[h] = struct{}{}
		}
	}

	// Rebuild visible in original order.
	m.visible = m.visible[:0]
	m.selectableIdxs = m.selectableIdxs[:0]
	for i, item := range m.items {
		if item.IsHeader() {
			if _, ok := neededHeaders[i]; ok {
				m.visible = append(m.visible, visibleEntry{
					itemIdx:  i,
					isHeader: true,
				})
			}
		} else {
			if md, ok := matched[i]; ok {
				m.visible = append(m.visible, visibleEntry{
					itemIdx:     i,
					matchedIdxs: md.idxs,
					isHeader:    false,
				})
				m.selectableIdxs = append(m.selectableIdxs, i)
			}
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Navigation helpers
// ────────────────────────────────────────────────────────────────────────────

func (m *Model) moveCursorDown() {
	if len(m.selectableIdxs) == 0 {
		return
	}
	m.cursorPos = (m.cursorPos + 1) % len(m.selectableIdxs)
}

func (m *Model) moveCursorUp() {
	if len(m.selectableIdxs) == 0 {
		return
	}
	m.cursorPos = (m.cursorPos - 1 + len(m.selectableIdxs)) % len(m.selectableIdxs)
}

func (m *Model) moveCursorToStart() {
	m.cursorPos = 0
	m.vp.GotoTop()
}

func (m *Model) moveCursorToEnd() {
	if len(m.selectableIdxs) > 0 {
		m.cursorPos = len(m.selectableIdxs) - 1
	}
	m.vp.GotoBottom()
}

// ensureCursorVisible scrolls the viewport so the cursor row is visible.
func (m *Model) ensureCursorVisible() {
	if m.cursorPos < 0 || len(m.selectableIdxs) == 0 {
		return
	}
	cursorItemIdx := m.selectableIdxs[m.cursorPos]

	// Find the viewport row of the cursor item.
	viewportRow := -1
	for i, ve := range m.visible {
		if !ve.isHeader && ve.itemIdx == cursorItemIdx {
			viewportRow = i
			break
		}
	}
	if viewportRow < 0 {
		return
	}

	yOffset := m.vp.YOffset()
	vpHeight := m.vp.Height()
	switch {
	case viewportRow < yOffset:
		m.vp.SetYOffset(viewportRow)
	case viewportRow >= yOffset+vpHeight:
		m.vp.SetYOffset(viewportRow - vpHeight + 1)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Selection helpers
// ────────────────────────────────────────────────────────────────────────────

func (m *Model) toggleSelection() {
	if m.cursorPos < 0 || m.cursorPos >= len(m.selectableIdxs) {
		return
	}
	idx := m.selectableIdxs[m.cursorPos]
	if _, ok := m.selected[idx]; ok {
		delete(m.selected, idx)
	} else if m.Limit == 0 || len(m.selected) < m.Limit {
		m.selected[idx] = struct{}{}
	}
}

func (m *Model) selectAll() {
	for _, idx := range m.selectableIdxs {
		m.selected[idx] = struct{}{}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Rendering helpers
// ────────────────────────────────────────────────────────────────────────────

// render assembles the full TUI output: search input + list viewport + help.
func (m *Model) render() string {
	return strings.Join([]string{
		m.input.View(),
		m.vp.View(),
		m.renderHelp(),
	}, "\n")
}

// renderList builds the string content for the viewport.
func (m *Model) renderList() string {
	if len(m.visible) == 0 {
		return m.styles.NoMatches.Render("  No matches found")
	}

	cursorItemIdx := -1
	if m.cursorPos >= 0 && m.cursorPos < len(m.selectableIdxs) {
		cursorItemIdx = m.selectableIdxs[m.cursorPos]
	}

	prefixWidth := lipgloss.Width(m.keymap.CursorPrefix)
	var sb strings.Builder

	for i, ve := range m.visible {
		item := m.items[ve.itemIdx]

		if ve.isHeader {
			sb.WriteString(m.styles.Header.Render(item.Label()))
		} else {
			isCursor := ve.itemIdx == cursorItemIdx
			_, isSelected := m.selected[ve.itemIdx]

			var line strings.Builder

			// Cursor indicator column.
			if isCursor {
				line.WriteString(m.styles.CursorIndicator.Render(m.keymap.CursorPrefix))
			} else {
				line.WriteString(strings.Repeat(" ", prefixWidth))
			}

			// Selection prefix (multi-select only).
			if m.Limit != 1 {
				if isSelected {
					line.WriteString(m.styles.SelectedPrefix.Render("◉ "))
				} else {
					line.WriteString(m.styles.UnselectedPrefix.Render("○ "))
				}
			} else {
				line.WriteString(" ")
			}

			// Spinner (SpinnerItem only).
			if s, ok := m.spinners[ve.itemIdx]; ok {
				line.WriteString(s.View())
				line.WriteByte(' ')
			}

			// Label with optional fuzzy-match highlights.
			label := item.Label()
			if len(ve.matchedIdxs) > 0 {
				label = applyHighlights(label, ve.matchedIdxs, m.styles.MatchHighlight)
			}
			if isCursor {
				line.WriteString(m.styles.CursorText.Render(label))
			} else {
				line.WriteString(m.styles.ItemText.Render(label))
			}

			sb.WriteString(line.String())
		}

		if i < len(m.visible)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// renderHelp returns a compact help line.
func (m *Model) renderHelp() string {
	hints := []string{"↑/↓ navigate", "enter select"}
	if m.Limit != 1 {
		hints = append(hints, "tab toggle")
		if m.Limit == 0 {
			hints = append(hints, "ctrl+a select all")
		}
	}
	hints = append(hints, "esc quit", "ctrl+c abort")
	return m.styles.Help.Render(strings.Join(hints, "  ·  "))
}

// ────────────────────────────────────────────────────────────────────────────
// Layout helpers
// ────────────────────────────────────────────────────────────────────────────

const (
	inputLines = 1
	helpLines  = 1
	minVPLines = 1
)

// resize updates the viewport dimensions to fit within m.height/m.width.
func (m *Model) resize() {
	vpH := m.height - inputLines - helpLines
	if vpH < minVPLines {
		vpH = minVPLines
	}
	m.vp.SetHeight(vpH)
	m.vp.SetWidth(m.width)
	m.input.SetWidth(m.width)
	m.vp.SetContent(m.renderList())
}

// ────────────────────────────────────────────────────────────────────────────
// Fuzzy highlight helper
// ────────────────────────────────────────────────────────────────────────────

// applyHighlights applies highlight styling to matched character positions.
// Consecutive matched indices are collapsed into contiguous ranges.
func applyHighlights(label string, idxs []int, style lipgloss.Style) string {
	if len(idxs) == 0 {
		return label
	}

	var ranges []lipgloss.Range
	start := idxs[0]
	end := idxs[0] + 1
	for _, idx := range idxs[1:] {
		if idx == end {
			end++
		} else {
			ranges = append(ranges, lipgloss.NewRange(start, end, style))
			start = idx
			end = idx + 1
		}
	}
	ranges = append(ranges, lipgloss.NewRange(start, end, style))

	return lipgloss.StyleRanges(label, ranges...)
}
