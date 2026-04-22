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
	}
}
