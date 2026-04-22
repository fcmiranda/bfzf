// Command bfzf is a CLI fuzzy finder built on Bubble Tea / Bubbles / Lip Gloss.
//
// Usage:
//
//	bfzf [flags] [item ...]
//	ls | bfzf [flags]
//	cat items.json | bfzf --json
//
// Items are read from positional arguments, stdin (one per line), or a JSON
// file. The TUI renders on stderr; selected item(s) are printed to stdout so
// the tool composes naturally in shell pipelines:
//
//	ls | bfzf | xargs cat
//
// Preview field selectors (fzf-compatible):
//
//	{}    full item label
//	{n}   nth whitespace-split field (1-based, e.g. {1} = first word)
//	{-1}  last whitespace-split field
//	{-n}  nth-from-last field
//
// JSON input format:
//
//	Simple:  ["item1", "item2", ...]
//	Rich:    [{"label":"item", "header":true, "spinner":true}, ...]
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fecavmi/bfzf"
)

// ────────────────────────────────────────────────────────────────────────────
// CLI-specific item types
// ────────────────────────────────────────────────────────────────────────────

// cliSpinnerItem is a simple item with an animated spinner.
type cliSpinnerItem struct {
	text string
	s    spinner.Model
}

func (c cliSpinnerItem) Label() string          { return c.text }
func (c cliSpinnerItem) FilterValue() string    { return c.text }
func (c cliSpinnerItem) IsHeader() bool         { return false }
func (c cliSpinnerItem) Spinner() spinner.Model { return c.s }

// newCLISpinnerItem creates a spinner-annotated item with an orange Dot spinner.
func newCLISpinnerItem(text string) cliSpinnerItem {
	return cliSpinnerItem{
		text: text,
		s: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("214"))),
		),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Flag definitions
// ────────────────────────────────────────────────────────────────────────────

type config struct {
	multi           bool
	limit           int
	prompt          string
	placeholder     string
	height          int
	groupPrefix     string
	spinnerPrefix   string
	previewCmd      string
	previewPosition string
	previewSize     int
	previewBorder   bool
	noSort          bool
	delimiter       string
	nul             bool
	jsonInput       bool
	listTitle       string
	listBorder      bool
	noInput         bool
	inputBorder     bool
	preset          string
	previewWidth    int
	previewHeight   int
	colorSpec       string
}

func parseFlags() config {
	cfg := config{}

	flag.BoolVar(&cfg.multi, "m", false, "enable unlimited multi-select")
	flag.BoolVar(&cfg.multi, "multi", false, "enable unlimited multi-select")
	flag.IntVar(&cfg.limit, "limit", 1, "max selections (overridden by -multi)")
	flag.StringVar(&cfg.prompt, "prompt", "❯ ", "search prompt")
	flag.StringVar(&cfg.placeholder, "placeholder", "Filter…", "placeholder text")
	flag.IntVar(&cfg.height, "height", 0, "terminal lines to use (0 = full screen)")
	flag.StringVar(&cfg.groupPrefix, "group-prefix", "", "lines with this prefix become group headers (prefix stripped)")
	flag.StringVar(&cfg.spinnerPrefix, "spinner-prefix", "", "lines with this prefix get an animated spinner (prefix stripped)")
	flag.StringVar(&cfg.previewCmd, "preview", "", "shell command for preview; use {} for full label, {-1} for last field, {n} for nth field")
	flag.StringVar(&cfg.previewPosition, "preview-position", "right", "preview panel position: right (default) or bottom")
	flag.IntVar(&cfg.previewSize, "preview-size", 40, "preview pane size in percent (10–90)")
	flag.BoolVar(&cfg.previewBorder, "preview-border", false, "draw a box border around the preview pane")
	flag.BoolVar(&cfg.noSort, "no-sort", false, "preserve input order (disable score-based sorting)")
	flag.StringVar(&cfg.delimiter, "delimiter", "\n", "field delimiter for plain-text input")
	flag.BoolVar(&cfg.nul, "0", false, "use NUL (\\x00) as delimiter")
	flag.BoolVar(&cfg.jsonInput, "json", false, "parse stdin as JSON (array of strings or objects)")
	flag.StringVar(&cfg.listTitle, "header", "", "title text shown above the list")
	flag.BoolVar(&cfg.listBorder, "list-border", false, "draw a box border around the list pane")
	flag.BoolVar(&cfg.noInput, "no-input", false, "hide the search input (navigation only)")
	flag.BoolVar(&cfg.inputBorder, "input-border", false, "draw a box border around the search input")
	flag.StringVar(&cfg.preset, "style", "default", "style preset: default, full, or minimal")
	flag.IntVar(&cfg.previewWidth, "preview-width", 0, "preview pane width in columns (0 = use --preview-size %)")
	flag.IntVar(&cfg.previewHeight, "preview-height", 0, "preview pane height in lines (0 = use --preview-size %)")
	flag.StringVar(&cfg.colorSpec, "color", "", `color spec: key:value[,key:value…] (e.g. "fg+:212,border:99")`)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: bfzf [flags] [item ...]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Fuzzy finder with groups, spinners, preview and JSON input.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Preview field selectors:")
		fmt.Fprintln(os.Stderr, "  {}    full item label")
		fmt.Fprintln(os.Stderr, "  {-1}  last whitespace-split field  (e.g. filename from eza/ls -l)")
		fmt.Fprintln(os.Stderr, "  {1}   first field")
		fmt.Fprintln(os.Stderr, "  {n}   nth field (1-based)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, `  ls -l | bfzf --preview 'stat {-1}'   # {-1} = filename column`)
		fmt.Fprintln(os.Stderr, `  ls    | bfzf --preview 'cat {}'`)
		fmt.Fprintln(os.Stderr, `  printf '#Fruits\nApple\nBanana' | bfzf --group-prefix '#'`)
		fmt.Fprintln(os.Stderr, `  printf '@Loading\nReady' | bfzf --spinner-prefix '@'`)
		fmt.Fprintln(os.Stderr, `  cat items.json | bfzf --json`)
		fmt.Fprintln(os.Stderr, `  bfzf -m item1 item2 item3`)
	}

	flag.Parse()
	return cfg
}

// ────────────────────────────────────────────────────────────────────────────
// Input reading
// ────────────────────────────────────────────────────────────────────────────

// readLines reads lines from r split by delimiter.
func readLines(r *os.File, delimiter string) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB max line
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), delimiter); i >= 0 {
			return i + len(delimiter), data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	var lines []string
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// ────────────────────────────────────────────────────────────────────────────
// JSON input
// ────────────────────────────────────────────────────────────────────────────

// jsonEntry is the rich object format for JSON input.
// All fields are optional; "label" is required for rich entries.
//
//	{"label": "Apple"}
//	{"label": "── Fruits", "header": true}
//	{"label": "Building…", "spinner": true}
type jsonEntry struct {
	Label   string `json:"label"`
	Header  bool   `json:"header"`
	Spinner bool   `json:"spinner"`
}

// readJSON parses stdin as a JSON array of strings or objects.
// Mixed arrays are not allowed; format is detected from the first element.
func readJSON(r *os.File) ([]bfzf.Item, error) {
	dec := json.NewDecoder(bufio.NewReader(r))

	// Expect opening '['
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("JSON parse: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("JSON input must be an array")
	}

	var items []bfzf.Item
	for dec.More() {
		// Peek the raw token to decide the array element type.
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("JSON decode element: %w", err)
		}

		// Try string first.
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if s != "" {
				items = append(items, bfzf.NewItem(s))
			}
			continue
		}

		// Then try rich object.
		var entry jsonEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, fmt.Errorf("JSON element is neither a string nor an object: %s", string(raw))
		}
		if entry.Label == "" {
			continue
		}
		switch {
		case entry.Header:
			items = append(items, bfzf.NewHeader(entry.Label))
		case entry.Spinner:
			items = append(items, newCLISpinnerItem(entry.Label))
		default:
			items = append(items, bfzf.NewItem(entry.Label))
		}
	}
	return items, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Item parsing (plain text — group-prefix / spinner-prefix annotation)
// ────────────────────────────────────────────────────────────────────────────

func parseItems(lines []string, groupPrefix, spinnerPrefix string) []bfzf.Item {
	items := make([]bfzf.Item, 0, len(lines))
	for _, line := range lines {
		switch {
		case groupPrefix != "" && strings.HasPrefix(line, groupPrefix):
			items = append(items, bfzf.NewHeader(strings.TrimPrefix(line, groupPrefix)))
		case spinnerPrefix != "" && strings.HasPrefix(line, spinnerPrefix):
			items = append(items, newCLISpinnerItem(strings.TrimPrefix(line, spinnerPrefix)))
		default:
			items = append(items, bfzf.NewItem(line))
		}
	}
	return items
}

// ────────────────────────────────────────────────────────────────────────────
// Preview template expansion
// ────────────────────────────────────────────────────────────────────────────

// fieldRefRe matches {}, {n}, {-n} in preview command templates.
var fieldRefRe = regexp.MustCompile(`\{(-?\d*)\}`)

// ansiEscRe matches ANSI/VT escape sequences (colors, cursor moves, etc.).
var ansiEscRe = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)

// stripANSI removes all ANSI escape sequences from s.
// This matches fzf's behaviour when constructing {} substitutions.
func stripANSI(s string) string {
	return ansiEscRe.ReplaceAllString(s, "")
}

// shellQuote wraps s in single quotes, safely escaping internal single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// expandPreviewTemplate substitutes field references in tmpl with values taken
// from the item label split on whitespace. ANSI escape codes are stripped from
// the label before splitting so that coloured ls/eza output works correctly:
//
//	{}    →  full label (ANSI-stripped, shell-quoted)
//	{1}   →  first whitespace-split field
//	{n}   →  nth field (1-based)
//	{-1}  →  last field  (e.g. filename from eza/ls -l)
//	{-n}  →  nth-from-last field
//
// Out-of-range indices are replaced with an empty string.
func expandPreviewTemplate(tmpl, label string) string {
	// Strip ANSI codes so colour-escaped ls/eza output doesn't break the cmd.
	clean := strings.TrimSpace(stripANSI(label))
	fields := strings.Fields(clean)
	n := len(fields)

	return fieldRefRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		inner := match[1 : len(match)-1] // strip braces
		if inner == "" {
			// {} → full clean label
			return shellQuote(clean)
		}
		idx, err := strconv.Atoi(inner)
		if err != nil {
			return match // unknown token — leave as-is
		}
		switch {
		case idx > 0 && idx <= n:
			return shellQuote(fields[idx-1])
		case idx < 0 && -idx <= n:
			return shellQuote(fields[n+idx])
		default:
			return "''" // out of range → empty quoted string
		}
	})
}

// makeShellPreview returns a [bfzf.PreviewFunc] that runs cmdTemplate as a
// shell command with field substitution applied to the focused item.
// Color-forcing environment variables are injected so tools like bat, highlight,
// and ls --color automatically produce coloured output in the preview pane.
func makeShellPreview(cmdTemplate string) bfzf.PreviewFunc {
	return func(item bfzf.Item) string {
		cmd := expandPreviewTemplate(cmdTemplate, item.Label())

		c := exec.Command("sh", "-c", cmd) // #nosec G204 — intentional user command
		// Inherit the current environment and layer in color-forcing variables
		// so preview commands produce ANSI-coloured output (matches fzf behaviour).
		c.Env = append(os.Environ(),
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
			"CLICOLOR_FORCE=1",  // BSD/macOS ls, many CLIs
			"FORCE_COLOR=3",     // npm/Node ecosystem
		)
		out, err := c.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
				return strings.TrimRight(string(exitErr.Stderr), "\n")
			}
			return fmt.Sprintf("preview error: %v", err)
		}
		return strings.TrimRight(string(out), "\n")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Main
// ────────────────────────────────────────────────────────────────────────────

func main() {
	cfg := parseFlags()

	// ── Collect items ─────────────────────────────────────────────────────────

	var (
		items     []bfzf.Item
		stdinUsed bool
	)

	if flag.NArg() > 0 {
		// Positional arguments take priority over stdin.
		items = parseItems(flag.Args(), cfg.groupPrefix, cfg.spinnerPrefix)
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintln(os.Stderr, "bfzf: cannot stat stdin:", err)
			os.Exit(1)
		}

		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// No pipe — behave like fzf: use $FZF_DEFAULT_COMMAND or find(1).
			// stdinUsed stays false; /dev/tty is always available.
			defaultCmd := os.Getenv("FZF_DEFAULT_COMMAND")
			if defaultCmd == "" {
				defaultCmd = "find . -type f -not -path '*/.*'"
			}
			out, err := exec.Command("sh", "-c", defaultCmd).Output() // #nosec G204
			if err != nil {
				fmt.Fprintln(os.Stderr, "bfzf: default command failed:", err)
				os.Exit(1)
			}
			var rawLines []string
			for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					rawLines = append(rawLines, line)
				}
			}
			items = parseItems(rawLines, cfg.groupPrefix, cfg.spinnerPrefix)
		} else {
			stdinUsed = true

			if cfg.jsonInput {
				items, err = readJSON(os.Stdin)
				if err != nil {
					fmt.Fprintln(os.Stderr, "bfzf:", err)
					os.Exit(1)
				}
			} else {
				delim := cfg.delimiter
				if cfg.nul {
					delim = "\x00"
				}
				rawLines, err := readLines(os.Stdin, delim)
				if err != nil {
					fmt.Fprintln(os.Stderr, "bfzf: error reading stdin:", err)
					os.Exit(1)
				}
				items = parseItems(rawLines, cfg.groupPrefix, cfg.spinnerPrefix)
			}
		}
	}

	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "bfzf: no items to display.")
		os.Exit(1)
	}

	// ── Build bfzf options ────────────────────────────────────────────────────

	limit := cfg.limit
	if cfg.multi {
		limit = 0
	}

	opts := []bfzf.Option{
		bfzf.WithLimit(limit),
		bfzf.WithPrompt(cfg.prompt),
		bfzf.WithPlaceholder(cfg.placeholder),
	}
	if cfg.height > 0 {
		opts = append(opts, bfzf.WithHeight(cfg.height))
	}
	if cfg.noSort {
		opts = append(opts, bfzf.WithNoSort())
	}
	if cfg.listTitle != "" {
		opts = append(opts, bfzf.WithListTitle(cfg.listTitle))
	}
	if cfg.listBorder {
		opts = append(opts, bfzf.WithListBorder())
	}
	if cfg.noInput {
		opts = append(opts, bfzf.WithNoInput())
	}
	if cfg.inputBorder {
		opts = append(opts, bfzf.WithInputBorder())
	}
	switch cfg.preset {
	case "full":
		opts = append(opts, bfzf.WithPreset(bfzf.PresetFull))
	case "minimal":
		opts = append(opts, bfzf.WithPreset(bfzf.PresetMinimal))
	}
	if cfg.previewWidth > 0 {
		opts = append(opts, bfzf.WithPreviewWidth(cfg.previewWidth))
	}
	if cfg.previewHeight > 0 {
		opts = append(opts, bfzf.WithPreviewHeight(cfg.previewHeight))
	}
	if cfg.colorSpec != "" {
		opts = append(opts, bfzf.WithColor(cfg.colorSpec))
	}
	if cfg.previewCmd != "" {
		opts = append(opts,
			bfzf.WithPreview(makeShellPreview(cfg.previewCmd)),
			bfzf.WithPreviewSize(cfg.previewSize),
		)
		if cfg.previewBorder {
			opts = append(opts, bfzf.WithPreviewBorder())
		}
		switch cfg.previewPosition {
		case "bottom":
			opts = append(opts, bfzf.WithPreviewPosition(bfzf.PreviewBottom))
		default:
			opts = append(opts, bfzf.WithPreviewPosition(bfzf.PreviewRight))
		}
	}

	// ── Tea program options ───────────────────────────────────────────────────

	var programOpts []tea.ProgramOption
	// TUI renders on stderr so stdout stays clean for piped output.
	programOpts = append(programOpts, tea.WithOutput(os.Stderr))

	// When stdin was consumed (pipe mode), open /dev/tty for key events.
	if stdinUsed {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err == nil {
			defer tty.Close() // #nosec G307
			programOpts = append(programOpts, tea.WithInput(tty))
		}
	}

	// ── Run ───────────────────────────────────────────────────────────────────

	m := bfzf.New(items, opts...)
	p := tea.NewProgram(m, programOpts...)
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bfzf:", err)
		os.Exit(1)
	}

	fm, ok := result.(bfzf.Model)
	if !ok || !fm.Submitted() {
		os.Exit(1)
	}

	selected := fm.Selected()
	if len(selected) == 0 {
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	for _, item := range selected {
		fmt.Fprintln(w, item.Label())
	}
	_ = w.Flush()
}

