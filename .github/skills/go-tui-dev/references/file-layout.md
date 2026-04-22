# File Layout Templates

Canonical file structure and boilerplate for a bfzf-style Go TUI library.

---

## Core Files

### `<pkg>.go` — Model

```go
package <pkg>

import (
    "strings"

    "charm.land/bubbles/v2/spinner"
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
)

// Item is the minimal interface every list entry must satisfy.
type Item interface {
    Label()       string
    FilterValue() string
    IsHeader()    bool
}

// SpinnerItem is an optional extension for items with an animated spinner.
type SpinnerItem interface {
    Item
    Spinner() spinner.Model
}

// SimpleItem is a basic selectable option.
type SimpleItem struct{ Text string }
func (s SimpleItem) Label() string       { return s.Text }
func (s SimpleItem) FilterValue() string { return s.Text }
func (s SimpleItem) IsHeader() bool      { return false }

// HeaderItem is a non-selectable group divider.
type HeaderItem struct{ Text string }
func (h HeaderItem) Label() string       { return h.Text }
func (h HeaderItem) FilterValue() string { return "" }
func (h HeaderItem) IsHeader() bool      { return true }

func NewItem(text string) SimpleItem   { return SimpleItem{Text: text} }
func NewHeader(text string) HeaderItem { return HeaderItem{Text: text} }

// Model is the Bubble Tea component.
type Model struct {
    Placeholder string
    Prompt      string
    Limit       int   // 0=unlimited, 1=single-select

    input    textinput.Model
    vp       viewport.Model

    items    []Item
    spinners map[int]spinner.Model

    visible        []visibleEntry
    selectableIdxs []int
    cursorPos      int
    selected       map[int]struct{}

    width, height int
    ready         bool
    quitting      bool
    submitted     bool

    styles  Styles
    keymap  KeyMap

    sortResults bool
}

type visibleEntry struct {
    itemIdx     int
    matchedIdxs []int
    isHeader    bool
}

const (
    inputLines = 1
    helpLines  = 1
    minVPLines = 1
)

func New(items []Item, opts ...Option) Model {
    ti := textinput.New()
    ti.Placeholder = "Filter..."
    ti.Prompt = "❯ "

    m := Model{
        Prompt:      "❯ ",
        Placeholder: "Filter...",
        Limit:       1,
        items:       items,
        spinners:    make(map[int]spinner.Model),
        selected:    make(map[int]struct{}),
        input:       ti,
        vp:          viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
        styles:      DefaultStyles(),
        keymap:      DefaultKeyMap(),
        sortResults: true,
    }
    for i, item := range items {
        if si, ok := item.(SpinnerItem); ok {
            s := si.Spinner()
            if s.Spinner.FPS == 0 { s.Spinner = spinner.Dot }
            m.spinners[i] = s
        }
    }
    for _, opt := range opts { opt(&m) }
    m.input.Placeholder = m.Placeholder
    m.input.Prompt = m.Prompt
    m.buildVisibleAll()
    return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
    cmds := []tea.Cmd{m.input.Focus()}
    for _, s := range m.spinners { cmds = append(cmds, s.Tick) }
    return tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() tea.View {
    if !m.ready { return tea.NewView("") }
    return tea.NewView(m.render())
}

// Selected returns the chosen items in original order.
func (m Model) Selected() []Item {
    var result []Item
    for i, item := range m.items {
        if _, ok := m.selected[i]; ok { result = append(result, item) }
    }
    return result
}

func (m Model) Submitted() bool { return m.submitted }
func (m Model) Quitting() bool  { return m.quitting }
```

---

### `styles.go`

```go
package <pkg>

import "charm.land/lipgloss/v2"

type Styles struct {
    Header           lipgloss.Style
    CursorText       lipgloss.Style
    ItemText         lipgloss.Style
    MatchHighlight   lipgloss.Style
    CursorIndicator  lipgloss.Style
    SelectedPrefix   lipgloss.Style
    UnselectedPrefix lipgloss.Style
    Help             lipgloss.Style
    NoMatches        lipgloss.Style
    // Add per-feature styles here.
}

func DefaultStyles() Styles {
    return Styles{
        Header:           lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).PaddingLeft(1),
        CursorText:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
        ItemText:         lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
        MatchHighlight:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")),
        CursorIndicator:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
        SelectedPrefix:   lipgloss.NewStyle().Foreground(lipgloss.Color("78")),
        UnselectedPrefix: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
        Help:             lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
        NoMatches:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true),
    }
}
```

---

### `keymap.go`

```go
package <pkg>

import (
    "charm.land/bubbles/v2/key"
    tea "charm.land/bubbletea/v2"
)

type KeyMap struct {
    Down, Up, Home, End     key.Binding
    Toggle, ToggleAndNext   key.Binding
    ToggleAndPrev, SelectAll key.Binding
    Submit, Quit, Abort     key.Binding
    CursorPrefix            string
}

func DefaultKeyMap() KeyMap {
    return KeyMap{
        Down:          key.NewBinding(key.WithKeys("down", "ctrl+n"), key.WithHelp("↓", "down")),
        Up:            key.NewBinding(key.WithKeys("up", "ctrl+k"),   key.WithHelp("↑", "up")),
        Home:          key.NewBinding(key.WithKeys("home"),            key.WithHelp("home", "start")),
        End:           key.NewBinding(key.WithKeys("end"),             key.WithHelp("end", "end")),
        Toggle:        key.NewBinding(key.WithKeys("ctrl+@"),          key.WithHelp("ctrl+@", "toggle")),
        ToggleAndNext: key.NewBinding(key.WithKeys("tab"),             key.WithHelp("tab", "toggle+next")),
        ToggleAndPrev: key.NewBinding(key.WithKeys("shift+tab"),       key.WithHelp("shift+tab", "toggle+prev")),
        SelectAll:     key.NewBinding(key.WithKeys("ctrl+a"),          key.WithHelp("ctrl+a", "select all")),
        Submit:        key.NewBinding(key.WithKeys("enter"),           key.WithHelp("enter", "select")),
        Quit:          key.NewBinding(key.WithKeys("esc"),             key.WithHelp("esc", "quit")),
        Abort:         key.NewBinding(key.WithKeys("ctrl+c"),          key.WithHelp("ctrl+c", "abort")),
        CursorPrefix:  "❯ ",
    }
}

func matchesAny(msg tea.Msg, bindings ...key.Binding) bool {
    if kp, ok := msg.(tea.KeyPressMsg); ok {
        return key.Matches(kp, bindings...)
    }
    return false
}
```

---

### `options.go`

```go
package <pkg>

// Option is a functional option for New.
type Option func(*Model)

func WithLimit(n int) Option         { return func(m *Model) { m.Limit = n } }
func WithPrompt(p string) Option     { return func(m *Model) { m.Prompt = p; m.input.Prompt = p } }
func WithPlaceholder(p string) Option{ return func(m *Model) { m.Placeholder = p; m.input.Placeholder = p } }
func WithStyles(s Styles) Option     { return func(m *Model) { m.styles = s } }
func WithKeyMap(km KeyMap) Option    { return func(m *Model) { m.keymap = km } }
func WithNoSort() Option             { return func(m *Model) { m.sortResults = false } }

func WithHeight(h int) Option {
    return func(m *Model) { m.height = h; m.resize(); m.ready = true }
}
func WithWidth(w int) Option {
    return func(m *Model) { m.width = w; m.vp.SetWidth(w); m.input.SetWidth(w) }
}

// WithStyleFunc patches individual style fields.
func WithStyleFunc(fn func(*Styles)) Option {
    return func(m *Model) { fn(&m.styles) }
}
```
