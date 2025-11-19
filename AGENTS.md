# Zif Developer Guide for AI Agents

This document provides a technical overview of the Zif project to assist AI agents in navigating, understanding, and modifying the codebase.

## Project Overview

Zif is a modern MUD (Multi-User Dungeon) client written in Go. It features:
- **UI**: Terminal-based UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **Scripting**: Lua-based module system for triggers, aliases, and timers.
- **Plugins**: Go plugin system for heavy extensions (e.g., Kallisti plugin).
- **Protocol**: Native support for MSDP (Mud Server Data Protocol).

## Project Structure

### Key Directories
- **`cmd/`**: Command-line entry points.
- **`config/`**: Configuration loading and management.
- **`layout/`**: Split-screen layout management (tree-based).
- **`modules/`**: Lua module integration and core Lua API bindings.
- **`plugins/`**: Go plugins (e.g., `plugins/kallisti`).
- **`protocol/`**: Telnet and MSDP protocol handling.
- **`screens/`**: UI screens (e.g., connection screen, main session screen).
- **`session/`**: Core session logic, connection handling, and state management.
- **`zif/`**: Core application logic and state.

### Key Files
- **`main.go`**: Application entry point. Initializes the Bubble Tea program.
- **`go.mod`**: Go dependencies.
- **`Makefile`**: Build scripts.

## Build & Run

### Standard Build
```bash
go build .
```

### Plugin Build (Kallisti)
The Kallisti plugin must be built as a Go plugin (`.so`) before it can be loaded.
```bash
go build --buildmode=plugin ./plugins/kallisti
```

### Running
Run the client:
```bash
./zif
```

Run with the Kallisti plugin:
```bash
./zif --kallisti
```

## Configuration (XDG)

Zif follows the XDG Base Directory specification.
- **Config Root**: `~/.config/zif/` (or `$XDG_CONFIG_HOME/zif/`)
- **Sessions**: `~/.config/zif/sessions.yaml`
- **Global Modules**: `~/.config/zif/modules/`
- **Session Modules**: `~/.config/zif/sessions/<session_name>/modules/`

## Lua Scripting API

Modules are written in Lua and interact with the Go backend via the `session` object.

### Core Functions
- `session:send(command)`: Send a command to the MUD.
- `session:output(text)`: Print text to the local client output.
- `session:register_trigger(name, pattern, func)`: Trigger callback on output match.
- `session:register_alias(name, pattern, func)`: Alias callback on input match.
- `session:add_timer(name, interval_ms, func)`: Periodic timer execution.

## Debugging

### Delve (`dlv`)
Since Zif uses a TUI, standard stdout debugging is difficult. Use Delve in headless mode or attach to a running process.

**Attach to running process:**
1. Run `./zif`
2. Find PID: `ps aux | grep zif`
3. Attach: `dlv attach <PID>`

**Headless mode:**
```bash
dlv debug --headless --listen=:2345 --api-version=2 .
# Connect from another terminal
dlv connect :2345
```

### Logging
- **`debug.log`**: General application logs.
- **`~/.config/zif/panic.log`**: Stack traces from crashes.

## Common Tasks

### Adding a New Command
1. Define the command in `session/command.go` or relevant handler.
2. Register it in the command parser.

### Modifying the UI
1. UI components are standard Bubble Tea models.
2. Look in `screens/` for high-level views.
3. `layout/` handles the split-pane architecture.

### Updating Protocol Support
1. `protocol/telnet.go` handles basic Telnet negotiation.
2. `protocol/msdp.go` handles MSDP data parsing.
