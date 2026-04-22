// Example program demonstrating bfzf features:
//   - Grouped options with non-selectable headers
//   - Options with animated spinners (indicating in-progress state)
//   - Multi-select mode
//   - Fuzzy search with highlighted matches
package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fecavmi/bfzf"
)

// ── Custom item types ────────────────────────────────────────────────────────

// langItem is a selectable programming language entry.
type langItem struct {
	name string
}

func (l langItem) Label() string       { return l.name }
func (l langItem) FilterValue() string { return l.name }
func (l langItem) IsHeader() bool      { return false }

// spinnerLangItem is a selectable item that displays a spinner (e.g. "loading").
type spinnerLangItem struct {
	name    string
	spinDef spinner.Model
}

func (s spinnerLangItem) Label() string       { return s.name }
func (s spinnerLangItem) FilterValue() string { return s.name }
func (s spinnerLangItem) IsHeader() bool      { return false }
func (s spinnerLangItem) Spinner() spinner.Model { return s.spinDef }

// newSpinnerItem creates a SpinnerItem with the given preset and ANSI 256 color code.
func newSpinnerItem(name string, preset spinner.Spinner, ansiColor string) spinnerLangItem {
	s := spinner.New(
		spinner.WithSpinner(preset),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(ansiColor))),
	)
	return spinnerLangItem{name: name, spinDef: s}
}

// ── Item list ────────────────────────────────────────────────────────────────

func buildItems() []bfzf.Item {
	return []bfzf.Item{
		// ── Group 1: Systems languages ───────────────────────────────────────
		bfzf.NewHeader("── Systems"),
		langItem{"Go"},
		langItem{"Rust"},
		langItem{"C"},
		langItem{"C++"},
		langItem{"Zig"},

		// ── Group 2: Scripting languages (all stable) ────────────────────────
		bfzf.NewHeader("── Scripting"),
		langItem{"Python"},
		langItem{"Ruby"},
		langItem{"Lua"},
		langItem{"Bash"},

		// ── Group 3: JVM languages ───────────────────────────────────────────
		bfzf.NewHeader("── JVM"),
		langItem{"Java"},
		langItem{"Kotlin"},
		langItem{"Scala"},
		langItem{"Clojure"},

		// ── Group 4: Loading / in-progress items (spinner examples) ──────────
		bfzf.NewHeader("── Experimental  ⟳"),
		newSpinnerItem("Carbon  (compiling stdlib…)", spinner.Dot, "214"),
		newSpinnerItem("Nim  (fetching packages…)", spinner.MiniDot, "86"),
		newSpinnerItem("Vale  (running benchmarks…)", spinner.Moon, "171"),
		newSpinnerItem("Mojo  (analysing types…)", spinner.Pulse, "213"),

		// ── Group 5: Web / frontend ──────────────────────────────────────────
		bfzf.NewHeader("── Web"),
		langItem{"TypeScript"},
		langItem{"JavaScript"},
		langItem{"Elm"},
		langItem{"PureScript"},
		langItem{"ReScript"},
	}
}

// ── Styles ───────────────────────────────────────────────────────────────────

func customStyles() bfzf.Styles {
	s := bfzf.DefaultStyles()

	// Slightly bolder header dividers using a dim cyan.
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("87")).
		PaddingLeft(0)

	// Bright prompt highlight colour.
	s.CursorText = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("51"))

	s.CursorIndicator = lipgloss.NewStyle().
		Foreground(lipgloss.Color("51"))

	s.MatchHighlight = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")).
		Underline(true)

	s.SelectedPrefix = lipgloss.NewStyle().
		Foreground(lipgloss.Color("78"))

	s.UnselectedPrefix = lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	return s
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	items := buildItems()

	m := bfzf.New(
		items,
		bfzf.WithLimit(0), // unlimited multi-select
		bfzf.WithPlaceholder("Search languages…"),
		bfzf.WithStyles(customStyles()),
	)

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fm, ok := result.(bfzf.Model)
	if !ok || !fm.Submitted() {
		fmt.Fprintln(os.Stderr, "aborted.")
		os.Exit(0)
	}

	selected := fm.Selected()
	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "nothing selected.")
		os.Exit(0)
	}

	names := make([]string, len(selected))
	for i, item := range selected {
		names[i] = item.Label()
	}
	fmt.Println(strings.Join(names, "\n"))
}
