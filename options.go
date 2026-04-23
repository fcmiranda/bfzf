package bfzf

import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
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

// WithPrompt sets the text-input prompt string (default "❯ ").
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
// The viewport adjusts to fit within this height. Resizing the terminal does
// not change the height once set (use [WithHeightPercent] for adaptive sizing).
func WithHeight(h int) Option {
	return func(m *Model) {
		m.height = h
		m.heightFixed = true
		m.resize()
		m.ready = true
	}
}

// WithHeightPercent sets the component height as a percentage (1–100) of the
// terminal height. The height adapts when the terminal is resized, unlike
// [WithHeight] which fixes the height. Overrides a previous [WithHeight] call.
func WithHeightPercent(pct int) Option {
	return func(m *Model) {
		if pct < 1 {
			pct = 1
		}
		if pct > 100 {
			pct = 100
		}
		m.heightPercent = pct
		m.heightFixed = false
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

// WithPreview attaches a preview function to the model. On each cursor
// movement the function is called in a goroutine and its output is displayed
// in a split preview pane. Pass nil to disable preview.
func WithPreview(fn PreviewFunc) Option {
	return func(m *Model) {
		m.previewFunc = fn
		if fn != nil && m.previewVP.Width() == 0 {
			m.previewVP = initPreview()
		}
	}
}

// WithPreviewPosition sets the position of the preview pane.
// Use [PreviewRight] (default) or [PreviewBottom].
func WithPreviewPosition(pos PreviewPosition) Option {
	return func(m *Model) {
		m.previewPos = pos
	}
}

// WithPreviewSize sets the percentage of available space allocated to the
// preview pane. Valid range is 10–90 (default 40).
func WithPreviewSize(pct int) Option {
	return func(m *Model) {
		if pct < 10 {
			pct = 10
		}
		if pct > 90 {
			pct = 90
		}
		m.previewSize = pct
	}
}

// WithNoSort disables score-based sorting of fuzzy matches, preserving the
// original input order (equivalent to fzf's --no-sort).
func WithNoSort() Option {
	return func(m *Model) {
		m.sortResults = false
	}
}

// WithListTitle sets a title string displayed above the list.
// Pass an empty string to hide the title.
func WithListTitle(title string) Option {
	return func(m *Model) {
		m.listTitle = title
	}
}

// WithListBorder enables a lipgloss border around the list pane using the
// ListBorder style. Call without argument to use the default rounded border.
func WithListBorder() Option {
	return func(m *Model) {
		m.showListBorder = true
		if m.styles.ListBorder.GetBorderStyle() == (lipgloss.Border{}) {
			m.styles.ListBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)
		}
	}
}

// WithPreviewBorder enables a box border around the preview pane, replacing
// the plain separator bar. Uses the PreviewBorder style.
func WithPreviewBorder() Option {
	return func(m *Model) {
		m.showPreviewBorder = true
		if m.styles.PreviewBorder.GetBorderStyle() == (lipgloss.Border{}) {
			m.styles.PreviewBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
		}
	}
}

// WithNoInput hides the search text input. Useful when using bfzf purely as a
// navigable list without filtering.
func WithNoInput() Option {
	return func(m *Model) {
		m.hideInput = true
	}
}

// WithInputBorder enables a lipgloss border around the search text input using
// the InputBorder style. Call without argument to use the default rounded border.
func WithInputBorder() Option {
	return func(m *Model) {
		m.showInputBorder = true
		if m.styles.InputBorder.GetBorderStyle() == (lipgloss.Border{}) {
			m.styles.InputBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)
		}
	}
}

// WithStyleFunc allows granular style overrides by applying a callback to the
// current Styles value. This is preferred over WithStyles when only a few
// style fields need changing.
//
//	bfzf.WithStyleFunc(func(s *bfzf.Styles) {
//	    s.CursorText = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
//	})
func WithStyleFunc(fn func(*Styles)) Option {
	return func(m *Model) {
		fn(&m.styles)
	}
}

// WithKeyMapFunc allows granular key binding overrides by applying a callback
// to the current KeyMap value. This is preferred over WithKeyMap when only a
// few bindings need changing.
//
//	bfzf.WithKeyMapFunc(func(km *bfzf.KeyMap) {
//	    km.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
//	})
func WithKeyMapFunc(fn func(*KeyMap)) Option {
	return func(m *Model) {
		fn(&m.keymap)
	}
}

// WithPreset applies a named layout+style combination.
//
//   - [PresetDefault] — no borders, plain separator (current default)
//   - [PresetFull]    — list border, input border, and preview border all enabled;
//     titles are embedded in the top border line fzf-style
//   - [PresetMinimal] — no borders, no help line
func WithPreset(p Preset) Option {
	return func(m *Model) {
		switch p {
		case PresetFull:
			WithListBorder()(m)
			WithInputBorder()(m)
			WithPreviewBorder()(m)
		case PresetMinimal:
			m.showListBorder = false
			m.showInputBorder = false
			m.showPreviewBorder = false
			m.styles.Help = lipgloss.Style{} // hide help line
		}
		// PresetDefault is the zero value — nothing to do.
	}
}

// WithPreviewWidth sets an absolute column count for the preview pane.
// When > 0 it overrides the [WithPreviewSize] percentage for the horizontal
// dimension. Only meaningful for [PreviewRight] layout.
func WithPreviewWidth(cols int) Option {
	return func(m *Model) {
		if cols > 0 {
			m.previewWidth = cols
		}
	}
}

// WithPreviewHeight sets an absolute row count for the preview pane.
// When > 0 it overrides the [WithPreviewSize] percentage for the vertical
// dimension. Only meaningful for [PreviewBottom] layout.
func WithPreviewHeight(lines int) Option {
	return func(m *Model) {
		if lines > 0 {
			m.previewHeight = lines
		}
	}
}

// WithColor parses a comma-separated key:value color specification (similar to
// fzf's --color flag) and applies it to the model's styles.
//
//	bfzf.WithColor("fg+:212,hl:220,border:99,preview-border:135")
//
// See [ApplyColorSpec] for the full list of supported keys and color formats.
func WithColor(spec string) Option {
	return func(m *Model) {
		_ = ApplyColorSpec(spec, &m.styles)
	}
}

// WithCursorPrefix overrides the cursor-line prefix glyph (default "❯ ").
// The string is rendered using the [Styles.CursorIndicator] style.
func WithCursorPrefix(s string) Option {
	return func(m *Model) {
		m.keymap.CursorPrefix = s
	}
}

// MarkerStyle holds the glyphs printed before selected and unselected items in
// multi-select mode. Both strings should have equal display widths to keep the
// list columns aligned.
type MarkerStyle struct {
	Selected   string
	Unselected string
}

// Predefined [MarkerStyle] sets.
var (
	MarkerCircles    = MarkerStyle{Selected: "◉ ", Unselected: "○ "}  // default
	MarkerSquares    = MarkerStyle{Selected: "▪ ", Unselected: "▫ "}
	MarkerFilled     = MarkerStyle{Selected: "◼ ", Unselected: "◻ "}
	MarkerArrows     = MarkerStyle{Selected: "▶ ", Unselected: "  "}
	MarkerCheckmarks = MarkerStyle{Selected: "✓ ", Unselected: "  "}
	MarkerStars      = MarkerStyle{Selected: "★ ", Unselected: "☆ "}
	MarkerDiamonds   = MarkerStyle{Selected: "◆ ", Unselected: "◇ "}
)

// WithMarkerStyle overrides the selected/unselected item glyphs for multi-select.
func WithMarkerStyle(ms MarkerStyle) Option {
	return func(m *Model) {
		m.keymap.SelectedPrefix = ms.Selected
		m.keymap.UnselectedPrefix = ms.Unselected
	}
}
