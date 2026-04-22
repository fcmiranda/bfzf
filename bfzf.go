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
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	// ── Preview ──────────────────────────────────────────────────────────────

	// previewFunc is nil when preview is disabled.
	previewFunc PreviewFunc
	// previewVP is the scrollable preview viewport; only used when previewFunc != nil.
	previewVP viewport.Model
	// previewPos controls split direction (right or bottom).
	previewPos PreviewPosition
	// previewSize is the percentage of available space given to the preview pane.
	previewSize int
	// lastPreviewIdx is the item index that last triggered a preview;
	// used to discard stale results and avoid duplicate work.
	lastPreviewIdx int
	// previewFocused is true when keyboard focus is on the preview pane.
	// (reserved for future use — focus toggle removed)

	// showPreviewBorder toggles a full box border around the preview pane.
	showPreviewBorder bool

	// ── List decorations ─────────────────────────────────────────────────────

	// listTitle is an optional title drawn above the list.
	listTitle string
	// showListBorder toggles a box border around the list pane.
	showListBorder bool

	// ── Input ────────────────────────────────────────────────────────────────

	// hideInput hides the search text input (list-only mode).
	hideInput bool
	// showInputBorder draws a border around the search text input.
	showInputBorder bool

	// ── Sort ─────────────────────────────────────────────────────────────────

	// sortResults controls whether fuzzy matches are sorted by score (default: true).
	sortResults bool
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
		previewSize:    40,
		lastPreviewIdx: -1,
		sortResults:    true,
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

	case previewResultMsg:
		// Discard stale responses: only apply if the item is still focused.
		if msg.itemIdx == m.lastPreviewIdx {
			m.previewVP.SetContent(msg.content)
		}

	case tea.KeyPressMsg:
		// Global keys (always active).
		switch {
		case matchesAny(msg, m.keymap.Abort):
			m.quitting = true
			return m, tea.Interrupt

		case matchesAny(msg, m.keymap.Quit):
			m.quitting = true
			return m, tea.Quit

		// Preview scroll keys — always route to preview viewport.
		case matchesAny(msg, m.keymap.PreviewDown):
			m.previewVP.SetYOffset(m.previewVP.YOffset() + 1)
		case matchesAny(msg, m.keymap.PreviewUp):
			m.previewVP.SetYOffset(m.previewVP.YOffset() - 1)
		case matchesAny(msg, m.keymap.PreviewPageDown):
			m.previewVP.SetYOffset(m.previewVP.YOffset() + m.previewVP.Height())
		case matchesAny(msg, m.keymap.PreviewPageUp):
			m.previewVP.SetYOffset(m.previewVP.YOffset() - m.previewVP.Height())
		case matchesAny(msg, m.keymap.PreviewTop):
			m.previewVP.GotoTop()
		case matchesAny(msg, m.keymap.PreviewBottom):
			m.previewVP.GotoBottom()

		case matchesAny(msg, m.keymap.ToggleInput):
			m.hideInput = !m.hideInput
			if m.hideInput {
				m.input.SetValue("")
				m.updateFilter()
			}
			m.resize()

		// List navigation and selection.
		default:
			switch {
			case matchesAny(msg, m.keymap.Submit):
				if len(m.selectableIdxs) > 0 {
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
				if !m.hideInput {
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

	// Trigger preview refresh if the focused item changed.
	if cmd := m.triggerPreview(); cmd != nil {
		cmds = append(cmds, cmd)
	}

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
	var matches []fuzzy.Match
	if m.sortResults {
		matches = fuzzy.Find(query, labels)
	} else {
		matches = fuzzy.FindNoSort(query, labels)
	}

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

// render assembles the full TUI output.
// The search input always spans the full width above any split.
func (m *Model) render() string {
	helpStr := m.renderHelp()
	listPane := m.renderListPane()

	// Compact helper: join non-empty strings with newlines.
	join := func(parts ...string) string {
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return strings.Join(out, "\n")
	}

	var inputView string
	if !m.hideInput {
		if m.showInputBorder {
			inputView = m.styles.InputBorder.Render(m.input.View())
		} else {
			inputView = m.input.View()
		}
	}

	if m.previewFunc == nil {
		return join(inputView, listPane, helpStr)
	}

	previewPane := m.renderPreviewPane()

	switch m.previewPos {
	case PreviewRight:
		var splitPanel string
		if m.showPreviewBorder {
			splitPanel = lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
		} else {
			barHeight := m.vp.Height()
			bar := m.styles.PreviewBorder.Render(
				strings.Repeat("│\n", barHeight-1) + "│",
			)
			splitPanel = lipgloss.JoinHorizontal(lipgloss.Top, listPane, bar, previewPane)
		}
		return join(inputView, splitPanel, helpStr)

	case PreviewBottom:
		sep := m.styles.PreviewBorder.Render(strings.Repeat("─", m.width))
		return join(inputView, listPane, sep, previewPane, helpStr)
	}

	return join(inputView, listPane, helpStr)
}

// renderListPane builds the list side: optional titled border + viewport.
// When a border is enabled, the listTitle is embedded in the top border line
// (fzf-style). Without a border it appears as a separate row above the viewport.
func (m *Model) renderListPane() string {
	vpView := m.vp.View()
	if m.showListBorder {
		// Title goes in the top border line (fzf-style).
		return titledBorder(vpView, m.listTitle, "", m.styles.ListBorder)
	}
	content := vpView
	if m.listTitle != "" {
		content = m.styles.ListTitle.Render(m.listTitle) + "\n" + vpView
	}
	return content
}

// renderPreviewPane builds the preview panel.
// When a preview border is enabled, the item title and line count are
// embedded in the top border line (fzf-style). Without a border, a title
// row above the viewport is used.
func (m *Model) renderPreviewPane() string {
	var titleStr string
	if m.cursorPos >= 0 && m.cursorPos < len(m.selectableIdxs) {
		label := m.items[m.selectableIdxs[m.cursorPos]].Label()
		titleStr = m.styles.PreviewTitle.Render(label)
	}

	// Line count indicator: currentLine/totalLines
	lineCount := m.renderPreviewLineCount()

	vpView := m.previewVP.View()
	vpView = m.renderPreviewWithScrollbar(vpView)

	if m.showPreviewBorder {
		// Title + line count both go in the top border line.
		return titledBorder(vpView, titleStr, lineCount, m.styles.PreviewBorder)
	}

	// No border: title row + line count above content.
	var titleRow string
	if titleStr != "" || lineCount != "" {
		titleRow = titleStr + lineCount
	}
	if titleRow != "" {
		return titleRow + "\n" + vpView
	}
	return vpView
}

// renderPreviewLineCount returns the "n/total" scroll indicator.
func (m *Model) renderPreviewLineCount() string {
	total := m.previewVP.TotalLineCount()
	if total == 0 {
		return ""
	}
	current := m.previewVP.YOffset() + 1
	if current > total {
		current = total
	}
	return m.styles.PreviewLineCount.Render(fmt.Sprintf("%d/%d", current, total))
}

// renderPreviewWithScrollbar overlays a 1-char scrollbar on the rightmost
// column of the viewport view. Returns vpView unchanged when all content fits
// without scrolling.
func (m *Model) renderPreviewWithScrollbar(vpView string) string {
	total := m.previewVP.TotalLineCount()
	vpH := m.previewVP.Height()
	yOffset := m.previewVP.YOffset()
	if total <= vpH || vpH == 0 {
		return vpView
	}

	thumbSize := max(1, vpH*vpH/total)
	maxOffset := total - vpH
	thumbPos := 0
	if maxOffset > 0 {
		thumbPos = yOffset * (vpH - thumbSize) / maxOffset
	}

	vpLines := strings.Split(vpView, "\n")
	for i, line := range vpLines {
		var sc string
		if i >= thumbPos && i < thumbPos+thumbSize {
			sc = m.styles.PreviewScrollbarThumb.Render("┃")
		} else {
			sc = m.styles.PreviewScrollbar.Render("│")
		}
		lineW := lipgloss.Width(line)
		if lineW > 0 {
			vpLines[i] = ansi.Truncate(line, lineW-1, "") + sc
		} else {
			vpLines[i] = sc
		}
	}
	return strings.Join(vpLines, "\n")
}

// titledBorder renders content wrapped in the given lipgloss border, embedding
// leftTitle on the left side of the top border and rightTitle on the right
// side (fzf-style). Empty titles are skipped.
//
//	╭─ leftTitle ───────────── rightTitle ─╮
//	│  content                             │
//	╰──────────────────────────────────────╯
func titledBorder(content, leftTitle, rightTitle string, style lipgloss.Style) string {
	rendered := style.Render(content)
	if leftTitle == "" && rightTitle == "" {
		return rendered
	}
	lines := strings.SplitN(rendered, "\n", 2)
	if len(lines) < 2 {
		return rendered
	}
	outerW := lipgloss.Width(lines[0])
	b := style.GetBorderStyle()
	lineStyle := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())

	// Inner fill = outer width minus two corner characters.
	innerW := outerW - 2

	// Build left segment: "─ leftTitle ─"
	leftSeg := ""
	if leftTitle != "" {
		leftSeg = b.Top + " " + leftTitle + " " + b.Top
	}
	// Build right segment: "─ rightTitle ─"
	rightSeg := ""
	if rightTitle != "" {
		rightSeg = b.Top + " " + rightTitle + " " + b.Top
	}
	leftW := lipgloss.Width(leftSeg)
	rightW := lipgloss.Width(rightSeg)
	fillW := max(0, innerW-leftW-rightW)

	newTop := lineStyle.Render(b.TopLeft + leftSeg + strings.Repeat(b.Top, fillW) + rightSeg + b.TopRight)
	return newTop + "\n" + lines[1]
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
	var hints []string
	hints = append(hints, "↑/↓ navigate", "enter select")
	if m.Limit != 1 {
		hints = append(hints, "tab toggle")
		if m.Limit == 0 {
			hints = append(hints, "ctrl+a select all")
		}
	}
	if m.previewFunc != nil {
		hints = append(hints, "shift+↑/↓ preview scroll")
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

// resize updates all sub-component dimensions to fit m.width × m.height,
// accounting for the preview split and border frames when enabled.
func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Fixed rows consumed by the input, title, and help line.
	fixedRows := helpLines
	if !m.hideInput {
		fixedRows += inputLines
		if m.showInputBorder {
			fixedRows += m.styles.InputBorder.GetVerticalFrameSize()
		}
	}
	// List title only takes a row when there is no list border (with a border
	// the title is embedded in the top border line, not a separate row).
	if m.listTitle != "" && !m.showListBorder {
		fixedRows++
	}

	// Border frame sizes (0 when the respective border is disabled).
	listBorderH := 0
	if m.showListBorder {
		listBorderH = m.styles.ListBorder.GetHorizontalFrameSize()
	}
	prevBorderH, prevBorderV := 0, 0
	if m.showPreviewBorder {
		prevBorderH = m.styles.PreviewBorder.GetHorizontalFrameSize()
		prevBorderV = m.styles.PreviewBorder.GetVerticalFrameSize()
	}

	// Input width accounts for optional input border.
	if !m.hideInput {
		inputW := m.width
		if m.showInputBorder {
			inputW -= m.styles.InputBorder.GetHorizontalFrameSize()
		}
		m.input.SetWidth(inputW)
	}

	if m.previewFunc == nil {
		vpH := max(minVPLines, m.height-fixedRows)
		vpW := max(1, m.width-listBorderH)
		m.vp.SetHeight(vpH)
		m.vp.SetWidth(vpW)
		m.vp.SetContent(m.renderList())
		return
	}

	// When the preview border is enabled, the title+line-count row is embedded
	// in the border top line (not a content row), so we don't need to deduct it.
	// Without a preview border, we still need one row for the title row.
	previewTitleH := 1
	if m.showPreviewBorder {
		previewTitleH = 0
	}

	switch m.previewPos {
	case PreviewRight:
		vpH := max(minVPLines, m.height-fixedRows)

		// Horizontal allocation: list area | optional sep | preview area
		listAreaW := m.width * (100 - m.previewSize) / 100
		sepW := 1
		if m.showPreviewBorder {
			sepW = 0 // rounded border replaces the bar separator
		}
		prevAreaW := m.width - listAreaW - sepW
		if prevAreaW < 8 {
			prevAreaW = 8
			listAreaW = m.width - prevAreaW - sepW
		}

		// Viewport widths = area width minus their respective border frames.
		m.vp.SetHeight(vpH)
		m.vp.SetWidth(max(1, listAreaW-listBorderH))
		m.previewVP.SetHeight(max(minVPLines, vpH-previewTitleH-prevBorderV))
		m.previewVP.SetWidth(max(8, prevAreaW-prevBorderH))

	case PreviewBottom:
		totalH := max(2, m.height-fixedRows-1) // -1 for ─── separator row
		listAreaH := totalH * (100 - m.previewSize) / 100
		prevAreaH := totalH - listAreaH

		m.vp.SetHeight(max(minVPLines, listAreaH))
		m.vp.SetWidth(max(1, m.width-listBorderH))
		m.previewVP.SetHeight(max(minVPLines, prevAreaH-previewTitleH-prevBorderV))
		m.previewVP.SetWidth(max(8, m.width-prevBorderH))
	}

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
