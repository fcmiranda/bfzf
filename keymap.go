package bfzf

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// KeyMap holds all key bindings used by the Model.
type KeyMap struct {
	Down        key.Binding
	Up          key.Binding
	Home        key.Binding
	End         key.Binding
	Toggle      key.Binding // toggle selection (multi-select)
	ToggleAndNext key.Binding // toggle + cursor down (tab)
	ToggleAndPrev key.Binding // toggle + cursor up (shift+tab)
	SelectAll   key.Binding
	Submit      key.Binding
	Quit        key.Binding
	Abort       key.Binding

	// CursorPrefix is the string rendered at the cursor line.
	CursorPrefix string
}

// ShortHelp implements help.KeyMap (optional).
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Down, k.Up, k.Submit, k.Quit}
}

// FullHelp implements help.KeyMap (optional).
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Down, k.Up, k.Home, k.End},
		{k.Toggle, k.ToggleAndNext, k.SelectAll},
		{k.Submit, k.Quit, k.Abort},
	}
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+j", "ctrl+n"),
			key.WithHelp("↓/ctrl+n", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "ctrl+k", "ctrl+p"),
			key.WithHelp("↑/ctrl+p", "up"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "go to end"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("ctrl+@"),
			key.WithHelp("ctrl+@", "toggle"),
		),
		ToggleAndNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle + next"),
		),
		ToggleAndPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "toggle + prev"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "abort"),
		),
		CursorPrefix: "❯ ",
	}
}

// matchesAny returns true if the message matches any of the given bindings
// (for key.Matches compatibility with both KeyPressMsg types).
func matchesAny(msg tea.Msg, bindings ...key.Binding) bool {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		return key.Matches(kp, bindings...)
	}
	return false
}
