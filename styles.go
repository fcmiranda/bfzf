package bfzf

import (
	"charm.land/lipgloss/v2"
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
