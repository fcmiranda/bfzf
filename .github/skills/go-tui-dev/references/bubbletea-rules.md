# Bubble Tea v2 Rules

API reference for `charm.land/bubbletea/v2` as used in this repo.

## tea.Model Interface

```go
type Model interface {
    Init()              tea.Cmd
    Update(tea.Msg)     (tea.Model, tea.Cmd)
    View()              tea.View          // NOT string
}
```

`View()` must return `tea.NewView(string)`.  Returning a plain string is a compile error.

## Messages

```go
// Terminal resize
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.resize()
    m.ready = true

// Key press
case tea.KeyPressMsg:
    if key.Matches(msg, m.keymap.Quit) { ... }

// Async result (custom message type)
case myResultMsg:
    m.content = msg.value
```

## Commands

```go
// Return a single command
return m, tea.Quit

// Return multiple commands
return m, tea.Batch(cmd1, cmd2, cmd3)

// Fire an async goroutine that returns a message
func doWork() tea.Msg {
    result := expensiveOp()
    return myResultMsg{value: result}
}
cmds = append(cmds, doWork)
```

## Key Matching

```go
import "charm.land/bubbles/v2/key"

binding := key.NewBinding(
    key.WithKeys("ctrl+c"),
    key.WithHelp("ctrl+c", "abort"),
)

// In Update:
case tea.KeyPressMsg:
    if key.Matches(msg, binding) { ... }
```

## Spinner Setup

```go
import "charm.land/bubbles/v2/spinner"

s := spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("214"))),
)

// Init:
cmds = append(cmds, s.Tick)

// Update:
s, cmd = s.Update(msg)
cmds = append(cmds, cmd)

// View:
line.WriteString(s.View())
```

## Viewport Setup

```go
import "charm.land/bubbles/v2/viewport"

vp := viewport.New(
    viewport.WithWidth(80),
    viewport.WithHeight(20),
)

vp.SetContent("line1\nline2\n...")
vp.SetYOffset(vp.YOffset() + 1)  // scroll down
vp.GotoTop()
vp.GotoBottom()
total := vp.TotalLineCount()      // int — NOT (int, _, _)
```

## TextInput Setup

```go
import "charm.land/bubbles/v2/textinput"

ti := textinput.New()
ti.Placeholder = "Filter..."
ti.Prompt = "❯ "
ti.Focus()

// In Update (non-key messages):
if _, isKey := msg.(tea.KeyPressMsg); !isKey {
    ti, cmd = ti.Update(msg)
}

// Width must be set after knowing terminal width:
ti.SetWidth(m.width)
```

## AltScreen vs Inline

```go
// Full-screen (typical for pickers)
p := tea.NewProgram(m, tea.WithAltScreen())

// Inline (embed in shell output)
p := tea.NewProgram(m)
```

## Stdin / TTY Pattern for CLI Pipes

```go
// When stdin is a pipe, redirect keyboard input from /dev/tty.
// Render TUI on stderr so stdout stays clean for piped output.
if !term.IsTerminal(os.Stdin.Fd()) {
    tty, _ := os.Open("/dev/tty")
    p := tea.NewProgram(m,
        tea.WithInput(tty),
        tea.WithOutput(os.Stderr),
    )
    ...
}
```
