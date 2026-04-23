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
	height          string
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
	cursor          string
	marker          string
	popup           string
	// new fzf-parity flags
	reverse          bool
	exact            bool
	query            string
	print0           bool
	headerLines      int
	previewWindow    string // "hidden" → start hidden
	markerSelected   string
	markerUnselected string
	inputWidth       int
	bind             []string // raw "key:action" strings
	// wrap flags
	wrapWord         bool
	wrapSign         string
	previewWrapSign  string
	// ui appearance
	infoHidden       bool
	outerBorder      string // "rounded","sharp","bold","block","double","none"
	noColor          bool
	noClear          bool   // disable alternate screen (leave output in scrollback)
}

// multiString is a flag.Value that accumulates repeated --bind values.
type multiString []string

func (ms *multiString) String() string  { return strings.Join(*ms, ", ") }
func (ms *multiString) Set(s string) error {
	*ms = append(*ms, s)
	return nil
}

func parseFlags() config {
	cfg := config{}

	flag.BoolVar(&cfg.multi, "m", false, "enable unlimited multi-select")
	flag.BoolVar(&cfg.multi, "multi", false, "enable unlimited multi-select")
	flag.IntVar(&cfg.limit, "limit", 1, "max selections (overridden by -multi)")
	flag.StringVar(&cfg.prompt, "prompt", "❯ ", "search prompt")
	flag.StringVar(&cfg.placeholder, "placeholder", "Filter…", "placeholder text")
	flag.StringVar(&cfg.height, "height", "", `component height: absolute lines (e.g. "20") or percentage (e.g. "40%"); empty = full screen`)
	flag.StringVar(&cfg.groupPrefix, "group-prefix", "", "lines with this prefix become group headers (prefix stripped)")
	flag.StringVar(&cfg.spinnerPrefix, "spinner-prefix", "", "lines with this prefix get an animated spinner (prefix stripped)")
	flag.StringVar(&cfg.cursor, "cursor", "", `cursor prefix glyph (default "❯ ")`)
	flag.StringVar(&cfg.marker, "marker", "", "multi-select marker style: circles (default), squares, filled, arrows, checkmarks, stars, diamonds")
	flag.StringVar(&cfg.popup, "popup", "", `start in tmux/Zellij popup; value is geometry: [center|top|bottom|left|right][,W%][,H%] (e.g. "center", "left,40%,90%")`)
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
	// fzf-parity flags
	flag.BoolVar(&cfg.reverse, "reverse", false, "render list in reverse order (last item at top)")
	flag.BoolVar(&cfg.exact, "exact", false, "exact match mode: disable fuzzy, use substring matching")
	flag.StringVar(&cfg.query, "query", "", "initial query string for pre-filtering")
	flag.BoolVar(&cfg.print0, "print0", false, "output NUL-separated results instead of newline-separated")
	flag.IntVar(&cfg.headerLines, "header-lines", 0, "treat first N input lines as a pinned non-scrolling header (excluded from matching)")
	flag.StringVar(&cfg.previewWindow, "preview-window", "", `preview window options; "hidden" starts with preview hidden (ctrl+/ to toggle)`)
	flag.StringVar(&cfg.markerSelected, "marker-selected", "", `raw glyph for selected items in multi-select mode (e.g. "▶ ")`)
	flag.StringVar(&cfg.markerUnselected, "marker-unselected", "", "raw glyph for unselected items in multi-select mode")
	flag.IntVar(&cfg.inputWidth, "input-width", 0, "constrain search input to N columns (0 = full width)")
	var bindSpec multiString
	flag.Var(&bindSpec, "bind", `runtime key binding key:action (repeatable, e.g. "ctrl+/:toggle-preview")`)
	flag.BoolVar(&cfg.wrapWord, "wrap-word", false, "enable word-level wrapping of long item labels in the list")
	flag.StringVar(&cfg.wrapSign, "wrap-sign", "", `glyph prepended to continuation lines when --wrap-word is active (e.g. "↩ ")`)
	flag.StringVar(&cfg.previewWrapSign, "preview-wrap-sign", "", `glyph shown on soft-wrapped continuation lines in preview pane (e.g. "↩")`)
	flag.BoolVar(&cfg.infoHidden, "no-info", false, "hide the match-count info line")
	flag.StringVar(&cfg.outerBorder, "border", "", `wrap entire picker in a border: rounded (default when flag set), sharp, bold, block, double`)
	flag.BoolVar(&cfg.noColor, "no-color", false, "disable all ANSI colour output")
	flag.BoolVar(&cfg.noClear, "no-clear", false, "disable alternate screen: leave picker output in scrollback on exit (default: alt-screen is used)")

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
		fmt.Fprintln(os.Stderr, "Bind actions:")
		fmt.Fprintln(os.Stderr, "  toggle-preview       toggle the preview pane")
		fmt.Fprintln(os.Stderr, "  clear-query          clear the search input")
		fmt.Fprintln(os.Stderr, "  abort                quit without selecting")
		fmt.Fprintln(os.Stderr, "  accept               confirm current selection")
		fmt.Fprintln(os.Stderr, "  reload(cmd)          re-run shell command and replace items")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, `  ls -l | bfzf --preview 'stat {-1}'   # {-1} = filename column`)
		fmt.Fprintln(os.Stderr, `  ls    | bfzf --preview 'cat {}'`)
		fmt.Fprintln(os.Stderr, `  printf '#Fruits\nApple\nBanana' | bfzf --group-prefix '#'`)
		fmt.Fprintln(os.Stderr, `  printf '@Loading\nReady' | bfzf --spinner-prefix '@'`)
		fmt.Fprintln(os.Stderr, `  cat items.json | bfzf --json`)
		fmt.Fprintln(os.Stderr, `  bfzf -m item1 item2 item3`)
		fmt.Fprintln(os.Stderr, `  ls | bfzf --reverse --query go`)
		fmt.Fprintln(os.Stderr, `  ls | bfzf --preview 'cat {}' --preview-window hidden --bind ctrl+/:toggle-preview`)
	}

	flag.Parse()
	cfg.bind = []string(bindSpec)
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
	if cfg.height != "" {
		if h, pct, ok := parseHeightArg(cfg.height); ok {
			if pct > 0 {
				opts = append(opts, bfzf.WithHeightPercent(pct))
			} else {
				opts = append(opts, bfzf.WithHeight(h))
			}
		}
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
	if cfg.cursor != "" {
		opts = append(opts, bfzf.WithCursorPrefix(cfg.cursor))
	}
	if cfg.marker != "" {
		if ms, ok := namedMarkerStyle(cfg.marker); ok {
			opts = append(opts, bfzf.WithMarkerStyle(ms))
		} else {
			fmt.Fprintf(os.Stderr, "bfzf: unknown marker style %q (circles|squares|filled|arrows|checkmarks|stars|diamonds)\n", cfg.marker)
		}
	}
	// New fzf-parity options.
	if cfg.reverse {
		opts = append(opts, bfzf.WithReverse())
	}
	if cfg.exact {
		opts = append(opts, bfzf.WithExact())
	}
	if cfg.query != "" {
		opts = append(opts, bfzf.WithQuery(cfg.query))
	}
	if cfg.headerLines > 0 {
		opts = append(opts, bfzf.WithHeaderLines(cfg.headerLines))
	}
	if cfg.previewWindow == "hidden" {
		opts = append(opts, bfzf.WithPreviewHidden())
	} else if cfg.previewWindow == "wrap-word" {
		opts = append(opts, bfzf.WithPreviewWrapWord())
	} else if cfg.previewWindow == "hidden,wrap-word" || cfg.previewWindow == "wrap-word,hidden" {
		opts = append(opts, bfzf.WithPreviewHidden())
		opts = append(opts, bfzf.WithPreviewWrapWord())
	}
	if cfg.markerSelected != "" || cfg.markerUnselected != "" {
		opts = append(opts, bfzf.WithMarkerGlyphs(cfg.markerSelected, cfg.markerUnselected))
	}
	if cfg.inputWidth > 0 {
		opts = append(opts, bfzf.WithInputWidth(cfg.inputWidth))
	}
	if cfg.wrapWord {
		opts = append(opts, bfzf.WithWrapWord())
	}
	if cfg.wrapSign != "" {
		opts = append(opts, bfzf.WithWrapSign(cfg.wrapSign))
	}
	if cfg.previewWrapSign != "" {
		opts = append(opts, bfzf.WithPreviewWrapSign(cfg.previewWrapSign))
	}
	if cfg.infoHidden {
		opts = append(opts, bfzf.WithInfoStyle(bfzf.InfoHidden))
	}
	if cfg.outerBorder != "" {
		opts = append(opts, bfzf.WithOuterBorder(parseBorderType(cfg.outerBorder)))
	}
	if cfg.noColor {
		opts = append(opts, bfzf.WithNoColor())
	}
	if cfg.noClear {
		opts = append(opts, bfzf.WithNoClear())
	}
	for _, bindSpec := range cfg.bind {
		keyStr, fn, err := parseBind(bindSpec, cfg.groupPrefix, cfg.spinnerPrefix)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bfzf:", err)
			os.Exit(1)
		}
		opts = append(opts, bfzf.WithBind(keyStr, fn))
	}
	// Popup mode: after items are ready, re-launch inside tmux/Zellij popup.
	if cfg.popup != "" && os.Getenv("BFZF_IN_POPUP") == "" {
		if err := runPopup(cfg.popup, items, cfg.groupPrefix, cfg.spinnerPrefix, stdinUsed); err != nil {
			fmt.Fprintln(os.Stderr, "bfzf:", err)
			os.Exit(1)
		}
		os.Exit(0)
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
	for i, item := range selected {
		if cfg.print0 {
			fmt.Fprintf(w, "%s\x00", item.Label())
		} else {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprint(w, item.Label())
		}
	}
	if !cfg.print0 {
		fmt.Fprintln(w) // trailing newline
	}
	_ = w.Flush()
}

// ────────────────────────────────────────────────────────────────────────────
// Height flag helpers
// ────────────────────────────────────────────────────────────────────────────

// parseHeightArg parses a height string such as "20" or "40%" into either an
// absolute line count or a percentage.  Returns (abs, 0, true) for absolute,
// (0, pct, true) for percentage, or (0, 0, false) on parse error.
func parseHeightArg(s string) (abs int, pct int, ok bool) {
	if strings.HasSuffix(s, "%") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "%"))
		if err != nil || n < 1 || n > 100 {
			return 0, 0, false
		}
		return 0, n, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, 0, false
	}
	return n, 0, true
}

// ────────────────────────────────────────────────────────────────────────────
// Bind helpers
// ────────────────────────────────────────────────────────────────────────────

// parseBind parses a "key:action" string and returns the key string and a
// BindFunc. Supported actions:
//   - toggle-preview
//   - toggle-wrap
//   - toggle-wrap-word
//   - toggle-preview-wrap-word
//   - clear-query
//   - abort
//   - accept
//   - reload(shell-cmd)
func parseBind(spec, groupPrefix, spinnerPrefix string) (string, bfzf.BindFunc, error) {
	i := strings.Index(spec, ":")
	if i < 0 {
		return "", nil, fmt.Errorf("invalid --bind %q: expected key:action", spec)
	}
	keyStr := spec[:i]
	action := strings.TrimSpace(spec[i+1:])
	if keyStr == "" {
		return "", nil, fmt.Errorf("invalid --bind %q: empty key", spec)
	}
	var fn bfzf.BindFunc
	switch {
	case action == "toggle-preview":
		fn = bfzf.BindTogglePreview()
	case action == "toggle-wrap":
		fn = bfzf.BindToggleWrap()
	case action == "toggle-wrap-word":
		fn = bfzf.BindToggleWrapWord()
	case action == "toggle-preview-wrap-word":
		fn = bfzf.BindTogglePreviewWrapWord()
	case action == "clear-query":
		fn = bfzf.BindClearQuery()
	case action == "abort":
		fn = func(m *bfzf.Model) tea.Cmd { return m.Quit() }
	case action == "accept":
		fn = func(m *bfzf.Model) tea.Cmd { return m.ForceSubmit() }
	case strings.HasPrefix(action, "reload(") && strings.HasSuffix(action, ")"):
		reloadCmd := action[len("reload(") : len(action)-1]
		fn = bfzf.BindReloadItems(func() []bfzf.Item {
			out, err := exec.Command("sh", "-c", reloadCmd).Output() // #nosec G204
			if err != nil {
				return nil
			}
			var lines []string
			for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					lines = append(lines, line)
				}
			}
			return parseItems(lines, groupPrefix, spinnerPrefix)
		})
	default:
		return "", nil, fmt.Errorf("unrecognised bind action %q in %q", action, spec)
	}
	return keyStr, fn, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Marker style helpers
// ────────────────────────────────────────────────────────────────────────────

// namedMarkerStyle resolves a style name to a [bfzf.MarkerStyle].
func namedMarkerStyle(name string) (bfzf.MarkerStyle, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "circles", "circle":
		return bfzf.MarkerCircles, true
	case "squares", "square":
		return bfzf.MarkerSquares, true
	case "filled":
		return bfzf.MarkerFilled, true
	case "arrows", "arrow":
		return bfzf.MarkerArrows, true
	case "checkmarks", "checkmark", "check":
		return bfzf.MarkerCheckmarks, true
	case "stars", "star":
		return bfzf.MarkerStars, true
	case "diamonds", "diamond":
		return bfzf.MarkerDiamonds, true
	}
	return bfzf.MarkerStyle{}, false
}

// parseBorderType maps fzf --border type strings to lipgloss.Border values.
// Unknown values default to RoundedBorder.
func parseBorderType(s string) lipgloss.Border {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sharp", "solid":
		return lipgloss.NormalBorder()
	case "bold", "thick":
		return lipgloss.ThickBorder()
	case "block":
		return lipgloss.BlockBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "none":
		return lipgloss.Border{}
	default: // "rounded" or any other value
		return lipgloss.RoundedBorder()
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Popup helpers
// ────────────────────────────────────────────────────────────────────────────

type popupSpec struct {
	position string // center | top | bottom | left | right
	width    string // e.g. "80%" or "40"
	height   string
}

// parsePopupSpec parses the --popup flag value.
//
//	"center"           → center, 50% × 50%
//	"80%"              → center, 80% × 80%
//	"100%,50%"         → center, 100% × 50%
//	"left,40%"         → left, 40% wide
//	"left,40%,90%"     → left, 40% wide × 90% tall
//	"top,40%"          → top, full width × 40% tall
//	"bottom,80%,40%"   → bottom, 80% wide × 40% tall
func parsePopupSpec(s string) popupSpec {
	spec := popupSpec{position: "center", width: "50%", height: "50%"}
	if s == "" {
		return spec
	}

	parts := strings.Split(s, ",")
	i := 0
	switch parts[0] {
	case "center", "top", "bottom", "left", "right":
		spec.position = parts[0]
		i = 1
	}

	// Collect non-empty, non-"border-native" parts as sizes.
	var sizes []string
	for ; i < len(parts); i++ {
		p := strings.TrimSpace(parts[i])
		if p != "" && p != "border-native" {
			sizes = append(sizes, p)
		}
	}

	switch spec.position {
	case "left", "right":
		if len(sizes) > 0 {
			spec.width = sizes[0]
		}
		if len(sizes) > 1 {
			spec.height = sizes[1]
		}
	case "top", "bottom":
		if len(sizes) == 1 {
			spec.height = sizes[0]
			spec.width = "100%"
		} else if len(sizes) >= 2 {
			spec.width = sizes[0]
			spec.height = sizes[1]
		}
	default: // center
		if len(sizes) == 1 {
			spec.width = sizes[0]
			spec.height = sizes[0]
		} else if len(sizes) >= 2 {
			spec.width = sizes[0]
			spec.height = sizes[1]
		}
	}
	return spec
}

// runPopup serialises items to a temp file (when stdin was the source), builds
// an inner bfzf command, and runs it inside a tmux or Zellij popup.  The
// selected output written by the subprocess is forwarded to our stdout.
func runPopup(popupArg string, items []bfzf.Item, groupPrefix, spinnerPrefix string, stdinUsed bool) error {
	spec := parsePopupSpec(popupArg)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find bfzf executable: %w", err)
	}

	// Temp file that receives the selected output of the inner bfzf process.
	outFile, err := os.CreateTemp("", "bfzf-out-*")
	if err != nil {
		return fmt.Errorf("cannot create temp output file: %w", err)
	}
	outPath := outFile.Name()
	outFile.Close()
	defer os.Remove(outPath) // #nosec G304

	// When items came from stdin, serialize them so the subprocess can re-read.
	var inputPath string
	if stdinUsed {
		inFile, err := os.CreateTemp("", "bfzf-in-*")
		if err != nil {
			return fmt.Errorf("cannot create temp input file: %w", err)
		}
		inputPath = inFile.Name()
		bw := bufio.NewWriter(inFile)
		for _, item := range items {
			label := item.Label()
			switch {
			case item.IsHeader() && groupPrefix != "":
				fmt.Fprintln(bw, groupPrefix+label)
			case spinnerPrefix != "":
				if _, ok := item.(bfzf.SpinnerItem); ok {
					fmt.Fprintln(bw, spinnerPrefix+label)
					continue
				}
				fallthrough
			default:
				fmt.Fprintln(bw, label)
			}
		}
		if err := bw.Flush(); err != nil {
			inFile.Close()
			os.Remove(inputPath)
			return fmt.Errorf("cannot write temp input file: %w", err)
		}
		inFile.Close()
		defer os.Remove(inputPath)
	}

	// Build inner shell command: BFZF_IN_POPUP=1 bfzf [original-args] [<input] >output
	var parts []string
	parts = append(parts, "BFZF_IN_POPUP=1", shellQuote(exe))
	for _, arg := range os.Args[1:] {
		parts = append(parts, shellQuote(arg))
	}
	if inputPath != "" {
		parts = append(parts, "<", shellQuote(inputPath))
	}
	parts = append(parts, ">", shellQuote(outPath))
	innerCmd := strings.Join(parts, " ")

	// Launch via the detected multiplexer.
	switch {
	case os.Getenv("TMUX") != "":
		err = runTmuxPopup(spec, innerCmd)
	case os.Getenv("ZELLIJ") != "" || os.Getenv("ZELLIJ_SESSION_NAME") != "":
		err = runZellijPopup(spec, innerCmd)
	default:
		return fmt.Errorf("--popup requires tmux 3.3+ ($TMUX) or Zellij 0.44+ ($ZELLIJ)")
	}
	if err != nil {
		// User cancelled or popup failed — treat as no selection.
		return nil
	}

	// Forward selected output to our stdout.
	data, err := os.ReadFile(outPath) // #nosec G304
	if err != nil || len(data) == 0 {
		return nil
	}
	_, err = os.Stdout.Write(data)
	return err
}

// runTmuxPopup launches innerCmd inside a tmux display-popup window.
func runTmuxPopup(spec popupSpec, innerCmd string) error {
	args := []string{"display-popup", "-E", "-w", spec.width, "-h", spec.height}
	switch spec.position {
	case "top":
		args = append(args, "-y", "0")
	case "bottom":
		args = append(args, "-y", "S")
	case "left":
		args = append(args, "-x", "0")
	case "right":
		args = append(args, "-x", "R")
	// center: tmux default (no -x/-y needed)
	}
	args = append(args, "--", "sh", "-c", innerCmd)
	cmd := exec.Command("tmux", args...) // #nosec G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr // popup renders on terminal, not our stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runZellijPopup launches innerCmd inside a Zellij floating pane.
func runZellijPopup(spec popupSpec, innerCmd string) error {
	_ = spec // Zellij run does not support explicit float sizing yet
	args := []string{"run", "--floating", "--close-on-exit", "--", "sh", "-c", innerCmd}
	cmd := exec.Command("zellij", args...) // #nosec G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

