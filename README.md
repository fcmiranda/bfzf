# bfzf

`bfzf` is an `fzf`-inspired fuzzy picker built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## What is the project?
It provides a reusable and highly customisable fuzzy-finding component for terminal applications written in Go.

## Features / How it works
- **Real-time fuzzy search** with match highlighting
- **Grouped options:** Non-selectable headers visually separate groups
- **Animated Spinners:** Per-option animated Bubble spinner support
- **Multi-select:** Configurable limit for single or multiple selection
- **Customisable:** Fully customisable styles and key bindings

## How to setup

To add `bfzf` to your project, simply run:

```bash
go get github.com/fecavmi/bfzf
```

## How to build

As `bfzf` is a Go library, it is built as part of your application. You can build or run the provided example to see it in action:

```bash
cd example
go build -o bfzf-example
./bfzf-example
```

Alternatively, you can just run it directly:

```bash
go run example/main.go
```

## Examples of Usage

### Basic Usage

Here is a minimal example demonstrating how to use `bfzf`:

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/fecavmi/bfzf"
)

func main() {
	items := []bfzf.Item{
		bfzf.NewHeader("Fruits"),
		bfzf.NewItem("Apple"),
		bfzf.NewItem("Banana"),
		bfzf.NewItem("Orange"),
	}

	m := bfzf.New(items, bfzf.WithHeight(12))
	p := tea.NewProgram(m, tea.WithAltScreen())
	
	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := final.(bfzf.Model); ok && fm.Submitted() {
		fmt.Println("Selected:", fm.Selected()[0].Label())
	}
}
```

### Advanced Usage

Check out the [`example/main.go`](example/main.go) file for a more comprehensive example that demonstrates advanced features, such as:
- Infinite multi-select
- Animated spinners for items (loading state)
- Custom styles
- Grouped headers