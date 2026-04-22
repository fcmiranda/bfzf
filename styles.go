package bfzf

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// ────────────────────────────────────────────────────────────────────────────
// Preset
// ────────────────────────────────────────────────────────────────────────────

// Preset is a named layout+style combination similar to fzf's --style option.
type Preset int

const (
	// PresetDefault is the default look: no borders, plain vertical separator
	// between list and preview pane.
	PresetDefault Preset = iota

	// PresetFull enables list border, input border, and preview border so that
	// every pane has a rounded box around it and titles are embedded in the top
	// border line fzf-style.
	PresetFull

	// PresetMinimal strips every decoration (no help line, no borders, no title).
	PresetMinimal
)

// Styles holds all lipgloss styles used to render the picker.
type Styles struct {
	// Header styles the non-selectable group header text.
	Header lipgloss.Style

	// CursorText styles the label of the item at the cursor position.
	CursorText lipgloss.Style

	// ItemText styles the label of non-cursor selectable items.
	ItemText lipgloss.Style

	// MatchHighlight styles the characters matched by fuzzy search.
	MatchHighlight lipgloss.Style

	// CursorIndicator styles the glyph drawn on the cursor row.
	CursorIndicator lipgloss.Style

	// SelectedPrefix styles the "◉" prefix for multi-select.
	SelectedPrefix lipgloss.Style

	// UnselectedPrefix styles the "○" prefix for multi-select.
	UnselectedPrefix lipgloss.Style

	// Help styles the help line at the bottom.
	Help lipgloss.Style

	// NoMatches styles the "no matches" message.
	NoMatches lipgloss.Style

	// ListTitle styles the optional title bar drawn above the list.
	ListTitle lipgloss.Style

	// ListBorder is applied as a lipgloss border around the entire list pane.
	// Leave zero-value to disable the list border.
	ListBorder lipgloss.Style

	// PreviewBorder styles the divider character(s) between the list and
	// the preview pane. Also used as the outer border around the preview when
	// WithPreviewBorder is enabled.
	PreviewBorder lipgloss.Style

	// PreviewTitle styles the one-line title bar at the top of the preview pane
	// that shows the focused item's label.
	PreviewTitle lipgloss.Style

	// PreviewLineCount styles the "n/total" scroll indicator in the preview.
	PreviewLineCount lipgloss.Style

	// PreviewFocused is applied to the preview title bar when the preview has
	// keyboard focus (i.e. after pressing the FocusPreview key).
	PreviewFocused lipgloss.Style

	// Input styles the text-input search box.
	Input lipgloss.Style

	// InputBorder is applied as a lipgloss border around the text-input search box.
	// Leave zero-value to disable the input border.
	InputBorder lipgloss.Style

	// PreviewScrollbar styles the scrollbar track character (│) rendered on the
	// right edge of the preview viewport when content overflows.
	PreviewScrollbar lipgloss.Style

	// PreviewScrollbarThumb styles the scrollbar thumb character (┃) rendered on
	// the right edge of the preview viewport.
	PreviewScrollbarThumb lipgloss.Style
}

// DefaultStyles returns an opinionated dark-terminal style set.
func DefaultStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			PaddingLeft(1),

		CursorText: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),

		ItemText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),

		MatchHighlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220")),

		CursorIndicator: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),

		SelectedPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")),

		UnselectedPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		NoMatches: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),

		ListTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			PaddingLeft(1),

		ListBorder: lipgloss.Style{}, // zero = no border by default

		PreviewBorder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		PreviewTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Italic(true).
			PaddingLeft(1),

		PreviewLineCount: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		PreviewFocused: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			PaddingLeft(1),

		Input: lipgloss.Style{}, // zero = no extra styling by default

		InputBorder: lipgloss.Style{}, // zero = no border by default

		PreviewScrollbar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")),

		PreviewScrollbarThumb: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Color spec
// ────────────────────────────────────────────────────────────────────────────

// ApplyColorSpec parses a comma-separated key:value color specification
// (similar to fzf's --color flag) and patches s in place.
//
// Supported keys:
//
//	fg             item text foreground
//	fg+            cursor item foreground
//	bg             item text background
//	bg+            cursor item background
//	hl             fuzzy match highlight foreground
//	header         group header foreground
//	prompt         cursor indicator foreground
//	pointer        alias for prompt
//	info           help / line-count foreground
//	border         all three border foregrounds at once
//	list-border    list pane border foreground
//	preview-border preview pane border foreground
//	input-border   search-input border foreground
//	scrollbar      preview scrollbar track foreground
//	scrollbar-thumb preview scrollbar thumb foreground
//
// Color values accept ANSI 256 numbers (e.g. "212"), hex strings ("#ff87d7"),
// or named ANSI 4-bit names (e.g. "red", "bright-blue").
//
// Unknown keys are silently ignored so that forward-compatible specs work.
func ApplyColorSpec(spec string, s *Styles) error {
	for _, pair := range strings.Split(spec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, ':')
		if idx < 0 {
			return fmt.Errorf("bfzf: invalid color spec %q (missing ':')", pair)
		}
		key := strings.TrimSpace(pair[:idx])
		val := strings.TrimSpace(pair[idx+1:])
		c, err := parseColor(val)
		if err != nil {
			return fmt.Errorf("bfzf: invalid color value %q for key %q: %w", val, key, err)
		}
		switch key {
		case "fg":
			s.ItemText = s.ItemText.Foreground(c)
		case "fg+":
			s.CursorText = s.CursorText.Foreground(c)
		case "bg":
			s.ItemText = s.ItemText.Background(c)
		case "bg+":
			s.CursorText = s.CursorText.Background(c)
		case "hl":
			s.MatchHighlight = s.MatchHighlight.Foreground(c)
		case "header":
			s.Header = s.Header.Foreground(c)
		case "prompt", "pointer":
			s.CursorIndicator = s.CursorIndicator.Foreground(c)
		case "info":
			s.Help = s.Help.Foreground(c)
			s.PreviewLineCount = s.PreviewLineCount.Foreground(c)
		case "border":
			s.ListBorder = s.ListBorder.BorderForeground(c)
			s.PreviewBorder = s.PreviewBorder.BorderForeground(c)
			s.InputBorder = s.InputBorder.BorderForeground(c)
		case "list-border":
			s.ListBorder = s.ListBorder.BorderForeground(c)
		case "preview-border":
			s.PreviewBorder = s.PreviewBorder.BorderForeground(c)
		case "input-border":
			s.InputBorder = s.InputBorder.BorderForeground(c)
		case "scrollbar":
			s.PreviewScrollbar = s.PreviewScrollbar.Foreground(c)
		case "scrollbar-thumb":
			s.PreviewScrollbarThumb = s.PreviewScrollbarThumb.Foreground(c)
		// Unknown keys are silently ignored.
		}
	}
	return nil
}

// parseColor converts a user-supplied color string to a [color.Color].
// Accepts:
//   - ANSI 256 integer strings ("212", "0"–"255")
//   - Hex strings ("#ff87d7")
//   - 4-bit ANSI names ("red", "bright-blue", "black", …)
func parseColor(s string) (color.Color, error) {
	if strings.HasPrefix(s, "#") {
		c := lipgloss.Color(s)
		return c, nil
	}
	if _, err := strconv.Atoi(s); err == nil {
		c := lipgloss.Color(s)
		return c, nil
	}
	// Named 4-bit colors.
	named := map[string]color.Color{
		"black":          lipgloss.Black,
		"red":            lipgloss.Red,
		"green":          lipgloss.Green,
		"yellow":         lipgloss.Yellow,
		"blue":           lipgloss.Blue,
		"magenta":        lipgloss.Magenta,
		"cyan":           lipgloss.Cyan,
		"white":          lipgloss.White,
		"bright-black":   lipgloss.BrightBlack,
		"bright-red":     lipgloss.BrightRed,
		"bright-green":   lipgloss.BrightGreen,
		"bright-yellow":  lipgloss.BrightYellow,
		"bright-blue":    lipgloss.BrightBlue,
		"bright-magenta": lipgloss.BrightMagenta,
		"bright-cyan":    lipgloss.BrightCyan,
		"bright-white":   lipgloss.BrightWhite,
	}
	if c, ok := named[strings.ToLower(s)]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unknown color %q", s)
}
