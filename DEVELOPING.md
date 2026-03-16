# Developing Lattice Plugins

This guide covers everything you need to build a Lattice plugin. Plugins are standalone programs that add new modules to the dashboard without requiring Lattice to be recompiled.

## Table of contents

- [How plugins work](#how-plugins-work)
- [Plugin protocol](#plugin-protocol)
- [Building a Go plugin (with SDK)](#building-a-go-plugin-with-sdk)
- [Building a plugin in any language](#building-a-plugin-in-any-language)
- [Using the styles package](#using-the-styles-package)
- [Config and environment variables](#config-and-environment-variables)
- [Testing your plugin](#testing-your-plugin)
- [Publishing your plugin](#publishing-your-plugin)
- [Architecture reference](#architecture-reference)

---

## How plugins work

1. Lattice reads the user's config (`~/.config/lattice/config.yaml`)
2. For each module, it first checks the built-in registry
3. If no built-in module matches, it searches for a binary named `lattice-<type>` in:
   - `~/.config/lattice/plugins/`
   - `$PATH`
4. If found, Lattice starts the binary and communicates with it over **stdin/stdout** using newline-delimited JSON
5. The plugin process stays alive for the lifetime of the dashboard

```
┌──────────┐  JSON stdin   ┌──────────────────┐
│  Lattice │ ────────────> │  lattice-spotify  │
│          │ <──────────── │  (your plugin)    │
└──────────┘  JSON stdout  └──────────────────┘
```

## Plugin protocol

Communication uses **newline-delimited JSON** — one JSON object per line, terminated by `\n`.

### Request (Lattice -> Plugin)

```json
{
  "type": "init | update | view",
  "config": {"key": "value"},
  "width": 40,
  "height": 10
}
```

| Field    | Type              | Present in  | Description                                    |
|----------|-------------------|-------------|------------------------------------------------|
| `type`   | `string`          | all         | `"init"`, `"update"`, or `"view"`              |
| `config` | `map[string]string` | `init`    | User's config for this module                  |
| `width`  | `int`             | `view`      | Available content width in characters          |
| `height` | `int`             | `view`      | Available content height in lines              |

### Response (Plugin -> Lattice)

```json
{
  "name": "MODULE TITLE",
  "content": "rendered text\nwith newlines",
  "min_width": 30,
  "min_height": 5,
  "interval": 10,
  "error": ""
}
```

| Field        | Type     | When to set        | Description                                      |
|--------------|----------|--------------------|--------------------------------------------------|
| `name`       | `string` | `init`             | Display title for the module box                 |
| `content`    | `string` | `update`, `view`   | Rendered text content (can include ANSI escapes) |
| `min_width`  | `int`    | `init`             | Preferred minimum width (characters)             |
| `min_height` | `int`    | `init`             | Preferred minimum height (lines)                 |
| `interval`   | `int`    | `init`             | Seconds between `update` requests (0 = none)     |
| `error`      | `string` | any                | Error message (displayed instead of content)     |

All fields are optional — omit or set to zero/empty if not needed.

### Request lifecycle

```
Lattice                          Plugin
  │                                │
  │──── init (with config) ───────>│
  │<──── name, interval, size ─────│
  │                                │
  │──── view (width, height) ─────>│  (on every render)
  │<──── content ──────────────────│
  │                                │
  │──── update ───────────────────>│  (every `interval` seconds)
  │<──── content ──────────────────│
  │                                │
  │  ... repeats until quit ...    │
```

The `view` request is sent synchronously during rendering. The `update` request is sent periodically based on the interval you set in the `init` response. Use `update` for background data fetching, and `view` for rendering with the current dimensions.

## Building a Go plugin (with SDK)

The easiest way. The `pkg/plugin` package handles the JSON read/write loop for you.

### 1. Create a new module

```bash
mkdir lattice-mymod && cd lattice-mymod
go mod init github.com/you/lattice-mymod
go get github.com/floatpane/lattice/pkg/plugin
```

### 2. Write the plugin

```go
// main.go
package main

import (
    "fmt"
    "net/http"
    "time"

    "github.com/floatpane/lattice/pkg/plugin"
)

var lastFetch string

func main() {
    plugin.Run(func(req plugin.Request) plugin.Response {
        switch req.Type {
        case "init":
            // Set the module title and refresh interval.
            return plugin.Response{
                Name:      "MY MODULE",
                MinWidth:  30,
                MinHeight: 4,
                Interval:  30, // fetch new data every 30 seconds
            }

        case "update":
            // Fetch data in the background. Lattice calls this
            // every `interval` seconds.
            lastFetch = fetchData(req.Config["api_key"])
            return plugin.Response{Content: lastFetch}

        case "view":
            // Render for the given dimensions. Called on every
            // screen refresh. Keep this fast.
            if lastFetch == "" {
                return plugin.Response{Content: "Loading..."}
            }
            // You can use req.Width and req.Height to adapt the layout.
            return plugin.Response{Content: lastFetch}
        }
        return plugin.Response{}
    })
}

func fetchData(apiKey string) string {
    // Your data fetching logic here
    return fmt.Sprintf("Data fetched at %s", time.Now().Format("15:04:05"))
}
```

### 3. Build and install

```bash
# Build the binary (must be named lattice-<name>)
go build -o lattice-mymod .

# Option A: Copy to the plugins directory
cp lattice-mymod ~/.config/lattice/plugins/

# Option B: Install globally
go install .
```

### 4. Add to config

```yaml
# ~/.config/lattice/config.yaml
modules:
  - type: mymod
    config:
      api_key: "your-key-here"
```

### 5. Run Lattice

```bash
lattice
```

## Building a plugin in any language

Any executable that reads JSON from stdin and writes JSON to stdout works. The binary must be named `lattice-<name>`.

### Python example

```python
#!/usr/bin/env python3
"""lattice-pyexample — a Lattice plugin in Python."""

import json
import sys
import datetime

def handle(req):
    if req["type"] == "init":
        return {
            "name": "PYTHON EXAMPLE",
            "min_width": 25,
            "min_height": 3,
            "interval": 5,
        }

    if req["type"] == "update":
        now = datetime.datetime.now().strftime("%H:%M:%S")
        return {"content": f"Updated at {now}"}

    if req["type"] == "view":
        width = req.get("width", 30)
        return {"content": f"Width: {width} chars"}

    return {}

for line in sys.stdin:
    req = json.loads(line.strip())
    resp = handle(req)
    print(json.dumps(resp), flush=True)
```

Make it executable and install:

```bash
chmod +x lattice-pyexample
cp lattice-pyexample ~/.config/lattice/plugins/
```

### Node.js example

```javascript
#!/usr/bin/env node
// lattice-nodeexample

const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });

rl.on('line', (line) => {
  const req = JSON.parse(line);
  let resp = {};

  switch (req.type) {
    case 'init':
      resp = { name: 'NODE EXAMPLE', interval: 10, min_width: 25, min_height: 3 };
      break;
    case 'update':
    case 'view':
      resp = { content: `Hello from Node.js! ${new Date().toLocaleTimeString()}` };
      break;
  }

  console.log(JSON.stringify(resp));
});
```

### Rust, C, or anything else

Same rules apply:
1. Name the binary `lattice-<name>`
2. Read one JSON line from stdin
3. Write one JSON line to stdout
4. Repeat until stdin closes

## Using the styles package

Go plugins can import `pkg/styles` for consistent colors and helpers:

```go
import "github.com/floatpane/lattice/pkg/styles"
```

### Available colors

| Variable          | Light       | Dark        | Use for                |
|-------------------|-------------|-------------|------------------------|
| `styles.Subtle`   | `#D9DCCF`   | `#383838`   | Borders, labels        |
| `styles.Accent`   | `#43BF6D`   | `#73F59F`   | Positive values, titles|
| `styles.Warn`     | `#F25D94`   | `#F55385`   | Warnings, high CPU     |
| `styles.Highlight` | `#874BFD`  | `#7D56F4`   | Memory, highlights     |
| `styles.DimText`  | `#9B9B9B`   | `#5C5C5C`   | Secondary text         |

All colors are `lipgloss.AdaptiveColor` and automatically adjust to light/dark terminals.

### Helper functions

```go
// Draw a progress bar
styles.RenderBar(75.0, 20, styles.Accent)
// ███████████████░░░░░

// Render a label-value pair
styles.RenderStat("CPU", "45%")
// CPU              45%

// Truncate a string
styles.Truncate("A very long string here", 15)
// A very long st…
```

## Config and environment variables

When a user configures your module in their `config.yaml`:

```yaml
modules:
  - type: mymod
    config:
      api_key: "abc123"
      city: "Berlin"
```

These values arrive in the `init` request's `config` field:

```json
{"type": "init", "config": {"api_key": "abc123", "city": "Berlin"}}
```

### For Go plugins (using pkg/config)

If you're writing an in-tree module, `ModuleConfig.Get()` provides config-with-env-fallback:

```go
func NewMyModule(cfg config.ModuleConfig) module.Module {
    apiKey := cfg.Get("api_key", "MY_MODULE_API_KEY", "")
    // Checks: cfg.config["api_key"] -> $MY_MODULE_API_KEY -> ""
}
```

### For external plugins

Handle it yourself in the `init` handler — read from the config map, fall back to environment variables:

```go
case "init":
    apiKey := req.Config["api_key"]
    if apiKey == "" {
        apiKey = os.Getenv("MY_MODULE_API_KEY")
    }
```

## Testing your plugin

### Manual testing

You can test your plugin by piping JSON to it:

```bash
# Test init
echo '{"type":"init","config":{"city":"London"}}' | ./lattice-mymod

# Test a full session
echo '{"type":"init","config":{}}
{"type":"view","width":40,"height":10}
{"type":"update"}
{"type":"view","width":40,"height":10}' | ./lattice-mymod
```

### Automated testing

For Go plugins, test the handler function directly:

```go
func TestHandler(t *testing.T) {
    resp := myHandler(plugin.Request{Type: "init"})
    if resp.Name == "" {
        t.Error("init should return a name")
    }
    if resp.Interval == 0 {
        t.Error("init should set an interval")
    }
}
```

### Debugging

Plugin stderr is not captured by Lattice, so you can use it for debug logging:

```go
fmt.Fprintln(os.Stderr, "debug: fetching data...")
```

```python
print("debug: fetching data...", file=sys.stderr)
```

## Publishing your plugin

### Naming convention

Name your repository `lattice-<name>` so that `go install` produces the correct binary name:

```
github.com/you/lattice-spotify  ->  binary: lattice-spotify  ->  type: spotify
```

### Users install with

```bash
lattice import github.com/you/lattice-spotify@latest
```

This runs `go install` and places the binary in `~/.config/lattice/plugins/`.

### Non-Go plugins

For plugins written in other languages, provide install instructions. Users can place the binary in `~/.config/lattice/plugins/` or anywhere in their `$PATH`.

## Architecture reference

### Module resolution order

When Lattice encounters a module type in the config:

1. Check the **built-in registry** (Go modules compiled into the binary)
2. Check `~/.config/lattice/plugins/lattice-<name>`
3. Check `$PATH` for `lattice-<name>`
4. If not found, the module is silently skipped

### Public packages (pkg/)

These are the stable, importable packages:

| Package                | Description                             |
|------------------------|-----------------------------------------|
| `pkg/module`           | `Module` interface                      |
| `pkg/config`           | `Config`, `ModuleConfig`, `Load()`      |
| `pkg/registry`         | `Register()`, `Get()`, `List()`         |
| `pkg/styles`           | Colors, `RenderBar`, `RenderStat`, `Truncate` |
| `pkg/plugin`           | `Request`, `Response`, `Run()` SDK      |

### Internal packages (internal/)

These are implementation details and cannot be imported:

| Package                | Description                             |
|------------------------|-----------------------------------------|
| `internal/layout`      | Grid layout engine                      |
| `internal/modules`     | Built-in module implementations         |
| `internal/plugin`      | Plugin process runner                   |

### Key design decisions

- **Exec-based plugins over Go plugins**: Go's `plugin` package is fragile and platform-limited. Exec-based plugins work with any language and survive Lattice upgrades.
- **JSON over stdin/stdout**: Simple, debuggable, and language-agnostic. No sockets, no HTTP, no dependencies.
- **Single-process plugins**: Each plugin is one long-running process. No fork-per-request overhead.
- **Synchronous view, async update**: `view` is called during rendering and must be fast. `update` runs on a timer for background work.
