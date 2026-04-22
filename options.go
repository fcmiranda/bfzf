package bfzf

import (
	"charm.land/bubbles/v2/spinner"
)

// Option is a functional option for [New].
type Option func(*Model)

// WithLimit sets the maximum number of selectable items.
// 0 means unlimited multi-select; 1 means single-select (default).
func WithLimit(n int) Option {
	return func(m *Model) {
		m.Limit = n
	}
}

// WithPrompt sets the text-input prompt string (default "> ").
func WithPrompt(p string) Option {
	return func(m *Model) {
		m.Prompt = p
		m.input.Prompt = p
	}
}

// WithPlaceholder sets the placeholder shown when the search box is empty.
func WithPlaceholder(p string) Option {
	return func(m *Model) {
		m.Placeholder = p
		m.input.Placeholder = p
	}
}

// WithHeight sets the overall height of the component in terminal lines.
// The viewport adjusts to fit within this height.
func WithHeight(h int) Option {
	return func(m *Model) {
		m.height = h
		m.resize()
		m.ready = true
	}
}

// WithWidth sets the overall width of the component in terminal columns.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.width = w
		m.vp.SetWidth(w)
		m.input.SetWidth(w)
	}
}

// WithStyles replaces the default styling.
func WithStyles(s Styles) Option {
	return func(m *Model) {
		m.styles = s
	}
}

// WithKeyMap replaces the default key bindings.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) {
		m.keymap = km
	}
}

// WithDefaultSpinner sets the default spinner preset applied to all [SpinnerItem]s
// that return a zero-value spinner.Model from Spinner().
// This is a convenience wrapper; items may set their own spinner via [SpinnerItem].
func WithDefaultSpinner(s spinner.Spinner) Option {
	return func(m *Model) {
		for i, sp := range m.spinners {
			if sp.Spinner.FPS == 0 {
				sp.Spinner = s
				m.spinners[i] = sp
			}
		}
	}
}
