# Golden-File Testing for View() Output

Pattern for snapshot-testing Bubble Tea `View()` output without a real terminal.

---

## Directory Layout

```
<package>/
├── <pkg>_test.go
└── testdata/
    ├── TestView_Empty.golden
    ├── TestView_Filtered.golden
    ├── TestView_CursorAtBottom.golden
    └── TestView_MultiSelect.golden
```

---

## Golden Helper

```go
// internal/golden/golden.go
package golden

import (
    "os"
    "path/filepath"
    "testing"
)

// RequireEqual fails the test if got doesn't match the stored golden file.
// Set UPDATE_GOLDEN=1 to regenerate all golden files.
func RequireEqual(t *testing.T, got []byte) {
    t.Helper()
    path := filepath.Join("testdata", t.Name()+".golden")
    if os.Getenv("UPDATE_GOLDEN") == "1" {
        _ = os.MkdirAll(filepath.Dir(path), 0o755)
        if err := os.WriteFile(path, got, 0o644); err != nil {
            t.Fatalf("golden: write %s: %v", path, err)
        }
        return
    }
    want, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("golden: read %s: %v (run with UPDATE_GOLDEN=1 to create)", path, err)
    }
    if string(got) != string(want) {
        t.Errorf("View() output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
    }
}
```

---

## Driving the Model in Tests

Because Bubble Tea models are pure values, you can call `Update` directly
without starting a program:

```go
func sendKey(m bfzf.Model, keys ...string) bfzf.Model {
    // Construct a KeyPressMsg that matches the given key string.
    // Simplest approach: use the actual key binding defined in DefaultKeyMap().
    msg := tea.KeyPressMsg{Code: tea.KeyCode(keys[0])}
    next, _ := m.Update(msg)
    return next.(bfzf.Model)
}
```

For bindings defined via `key.WithKeys(...)` the easiest way is to construct
the message using `tea.KeyPressMsg{Sym: tea.KeySym..., Runes: []rune{...}}`.
Use `tea/teatest` if you need full program lifecycle testing.

---

## Required Test Cases

Always write tests for these states:

| Test Name | Setup | What to assert |
|-----------|-------|---------------|
| `TestView_Empty` | No items | "No matches" message visible |
| `TestView_Unfiltered` | Items, no query | All items visible, first item at cursor |
| `TestView_Filtered` | Type a query that matches subset | Only matching items; headers hidden if no matches |
| `TestView_CursorDown` | Navigate down once | Second item highlighted |
| `TestView_CursorAtBottom` | Navigate to last item | Viewport scrolled; last item at cursor |
| `TestView_MultiSelect` | Toggle two items | Both show ◉ prefix in output |
| `TestView_NoInput` | `WithNoInput()` | No text-input row in output |
| `TestView_WithTitle` | `WithListTitle("Foo")` | Title text visible |

---

## ANSI Stripping

In CI (no colour profile), `View()` output may differ from local runs if
lipgloss detects a non-colour terminal. Normalise before comparison:

```go
// stripANSI removes ANSI escape sequences for stable golden comparisons.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }
```

Apply in your helper:

```go
got := stripANSI(string(m.View()))
golden.RequireEqual(t, []byte(got))
```

---

## Running Tests

```bash
# Run all tests
go test ./...

# Regenerate all golden files
UPDATE_GOLDEN=1 go test ./...

# Run a single test
go test -run TestView_Filtered ./...
```

---

## teatest Integration (Optional)

For full-lifecycle tests including `Init()` and async messages:

```go
import "github.com/charmbracelet/x/exp/teatest"

func TestProgram_Submit(t *testing.T) {
    m := bfzf.New(items, bfzf.WithHeight(10))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Send key presses
    tm.Send(tea.KeyPressMsg{Sym: tea.KeyEnter})

    tm.WaitFinished(t, time.Second)
    final := tm.FinalModel(t).(bfzf.Model)

    if !final.Submitted() {
        t.Fatal("expected submitted")
    }
}
```
