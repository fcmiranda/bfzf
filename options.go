package bfzf

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
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
// A border is automatically enabled on the preview pane so that a visible
// right edge and title bar are shown. Call [WithNoPreviewBorder] to opt out.
func WithPreview(fn PreviewFunc) Option {
	return func(m *Model) {
		m.previewFunc = fn
		if fn != nil {
			if m.previewVP.Width() == 0 {
				m.previewVP = initPreview()
			}
			// Auto-enable preview border for a visible right edge that aligns
			// with the input filter's right edge.
			WithPreviewBorder()(m)
		}
	}
}

// WithNoPreviewBorder disables the preview pane border, reverting to the plain
// separator bar layout. This opts out of the default set by [WithPreview].
func WithNoPreviewBorder() Option {
	return func(m *Model) {
		m.showPreviewBorder = false
		// Restore plain-separator border style (foreground-only, no box).
		m.styles.PreviewBorder = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
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

// WithReverse renders the list in reverse order within the viewport
// (last item at the top, first at the bottom), equivalent to fzf's --reverse.
func WithReverse() Option {
	return func(m *Model) {
		m.reverse = true
	}
}

// WithExact disables fuzzy matching and treats all query tokens as exact
// substring matches, equivalent to fzf's --exact flag.
func WithExact() Option {
	return func(m *Model) {
		m.exact = true
	}
}

// WithQuery sets an initial pre-filled search query, equivalent to
// fzf's --query STRING. The filter is applied immediately in [New].
func WithQuery(s string) Option {
	return func(m *Model) {
		m.input.SetValue(s)
	}
}

// WithHeaderLines pins the first n items as a non-scrolling header above the
// list viewport, equivalent to fzf's --header-lines N.
// The pinned items are excluded from fuzzy matching and selection.
func WithHeaderLines(n int) Option {
	return func(m *Model) {
		if n < 0 {
			n = 0
		}
		m.headerLines = n
	}
}

// WithPreviewHidden starts with the preview pane hidden even when a preview
// function is set. The user can toggle it with [KeyMap.TogglePreview]
// (default Ctrl+/), equivalent to fzf's --preview-window hidden.
func WithPreviewHidden() Option {
	return func(m *Model) {
		m.hidePreview = true
	}
}

// WithBind registers a runtime key binding: when keyStr is pressed, fn is
// called with a pointer to the live Model.
// keyStr uses the same format as key.WithKeys (e.g. "ctrl+/", "alt+p").
// Use the BindTogglePreview, BindChangeQuery, BindClearQuery, and
// BindReloadItems helpers to build common actions.
func WithBind(keyStr string, fn BindFunc) Option {
	return func(m *Model) {
		m.bindActions = append(m.bindActions, bindEntry{
			binding: key.NewBinding(key.WithKeys(keyStr)),
			fn:      fn,
		})
	}
}

// WithInputWidth constrains the search text input to w columns.
// The remaining row width is left blank, allowing other UI elements to
// coexist on the same line in parent layouts.
func WithInputWidth(w int) Option {
	return func(m *Model) {
		if w > 0 {
			m.inputWidth = w
		}
	}
}

// WithMarkerGlyphs sets the raw glyph strings for selected/unselected items
// in multi-select mode (fzf's --marker STR). Both strings should have equal
// display widths to keep list columns aligned.
func WithMarkerGlyphs(selected, unselected string) Option {
	return func(m *Model) {
		m.keymap.SelectedPrefix = selected
		m.keymap.UnselectedPrefix = unselected
	}
}

// WithWrapWord enables word-level wrapping of long item labels in the list.
// Each label is split into logical lines at word boundaries, fitting within
// the list viewport width. Continuation lines are indented to align with the
// label start, with an optional wrap indicator set via [WithWrapSign].
// Equivalent to fzf's --wrap --wrap-sign with word-wrap mode.
func WithWrapWord() Option {
	return func(m *Model) {
		m.wrapWord = true
		m.wrapList = false
		m.vp.SoftWrap = false
	}
}

// WithWrap enables character-level soft-wrapping of long item labels in the
// list viewport (using the viewport's built-in SoftWrap mode).
// Word wrap (--wrap-word) takes precedence when both are set.
func WithWrap() Option {
	return func(m *Model) {
		if !m.wrapWord {
			m.wrapList = true
			m.vp.SoftWrap = true
		}
	}
}

// WithWrapSign sets the glyph prepended to continuation lines when word-wrap
// is enabled ([WithWrapWord]). It is styled with [Styles.WrapSign].
// Default: no sign (empty string).
func WithWrapSign(sign string) Option {
	return func(m *Model) {
		m.wrapSign = sign
	}
}

// WithPreviewWrapWord enables word-level soft-wrapping in the preview pane.
// Equivalent to fzf's --preview-window wrap-word.
func WithPreviewWrapWord() Option {
	return func(m *Model) {
		m.previewWrapWord = true
		m.previewVP.SoftWrap = true
		sign := m.previewWrapSign
		if sign == "" {
			// Apply wrapping without a gutter sign — default SoftWrap handles linebreaks.
			return
		}
		signW := lipgloss.Width(sign)
		m.previewVP.LeftGutterFunc = func(ctx viewport.GutterContext) string {
			if ctx.Soft {
				return sign
			}
			return string(make([]byte, signW)) // ASCII space filler
		}
	}
}

// WithPreviewWrapSign sets the glyph displayed on soft-wrapped continuation
// lines in the preview pane (e.g. "↩"). When non-empty this is shown in the
// left gutter of the preview viewport. Requires [WithPreviewWrapWord].
// Equivalent to fzf's --preview-wrap-sign.
func WithPreviewWrapSign(sign string) Option {
	return func(m *Model) {
		m.previewWrapSign = sign
		if m.previewWrapWord && sign != "" {
			signW := lipgloss.Width(sign)
			spaces := make([]byte, signW)
			for i := range spaces {
				spaces[i] = ' '
			}
			spaceStr := string(spaces)
			m.previewVP.LeftGutterFunc = func(ctx viewport.GutterContext) string {
				if ctx.Soft {
					return sign
				}
				return spaceStr
			}
		}
	}
}

// WithInfoStyle sets where the match count / info text is displayed.
// Use [InfoDefault] (match count above the list) or [InfoHidden] (suppressed).
func WithInfoStyle(s InfoStyle) Option {
	return func(m *Model) {
		m.infoStyle = s
	}
}

// WithOuterBorder wraps the entire picker in a lipgloss border.
// Pass a border type such as [lipgloss.RoundedBorder], [lipgloss.NormalBorder],
// etc. The border colour can be customised via [WithColor] ("outer-border" key).
// Equivalent to fzf's --border.
func WithOuterBorder(b lipgloss.Border) Option {
	return func(m *Model) {
		m.showOuterBorder = true
		if m.outerBorderStyle.GetBorderStyle() == (lipgloss.Border{}) {
			m.outerBorderStyle = lipgloss.NewStyle().
				Border(b).
				BorderForeground(lipgloss.Color("240"))
		} else {
			m.outerBorderStyle = m.outerBorderStyle.Border(b)
		}
	}
}

// WithNoColor disables all ANSI colour output by replacing every Styles field
// with an empty (colourless) lipgloss.Style. Equivalent to fzf's --no-color.
func WithNoColor() Option {
	return func(m *Model) {
		m.noColor = true
		m.styles = Styles{}
	}
}

// WithNoClear disables the alternate screen buffer so that bfzf renders inline
// and leaves its output in the terminal scrollback when it exits.
// By default bfzf uses the alternate screen buffer so quitting leaves no
// residue (equivalent to piping through fzf).  Pass this option when you embed
// bfzf inside a larger Bubble Tea program or want persistent scrollback output.
func WithNoClear() Option {
	return func(m *Model) {
		m.useAltScreen = false
	}
}
