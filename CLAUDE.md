# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build everything (kallisti plugin + main binary)
make

# Build main binary only
go build .

# Build kallisti plugin only (produces kallisti.so)
go build --buildmode=plugin ./plugins/kallisti

# Cross-compile for Windows
make windows

# Run tests
go test ./...

# Run tests for a specific package
go test ./session/
go test ./plugins/kallisti/
```

## Architecture Overview

Zif is a terminal MUD client built on Bubble Tea (bubbletea). The main model (`ZifModel` in `main.go`) owns the input, layout, session handler, and status bar. All inter-component communication happens via messages on `SessionHandler.Sub` channel.

### Package Structure

- **session/** — Core package (~3400 lines). Contains `Session` (individual MUD connection), `SessionHandler` (manages multiple sessions), and all registries (actions, aliases, events, tickers, modules, queue). This is where most logic lives.
- **protocol/** — Telnet and MSDP protocol handling. `MSDPHandler` stores parsed MSDP data in a thread-safe map. `protocol/msdp/parser.go` handles byte-level MSDP parsing.
- **layout/** — Tree-based split-pane UI system. `Node` interface with `Pane` (leaf) and `Container` (internal split) nodes. Supports arbitrary horizontal/vertical splits.
- **config/** — XDG directory management and session auto-start config (`sessions.yaml`).
- **plugins/** — Go plugin system using `-buildmode=plugin`. Kallisti is the main plugin (Legends of Kallisti MUD support with mapping, pathfinding, auto-travel).
- **cmd/** — Standalone utilities (`cmd/map/` for map rendering, `cmd/msdp/` for protocol testing).

### Data Flow

**Input:** User → `HandleInput()` → alias check → `#command` dispatch OR `socket.Write()` with CR LF

**Output:** MUD socket → `mudReader()` goroutine (single-byte reads, 20ms timeout) → ringlog → action matching → event firing → MUD line hooks → `UpdateMessage` on channel → Bubble Tea UI

**MSDP:** Telnet IAC/SB/SE → `HandleSB()` → `ParseMSDP()` → `MSDPHandler.Data` map → MSDP update hooks

### Extension Model

Three layers of extensibility, all using a registry pattern with enable/disable and execution counting:

1. **Lua modules** — Directories with `init.lua` under `~/.config/zif/modules/` (global) or `~/.config/zif/sessions/<name>/modules/` (per-session). Lua API exposed via `session:send()`, `session:output()`, `session:register_trigger()`, `session:register_alias()`, `session:add_timer()`, etc. See `session/lua_api.go`.

2. **Go plugins** — Dynamic `.so` libraries loaded at runtime. Must export `RegisterSession(s *session.Session)`. Can inject Lua context (`RegisterContextInjector`), hook MSDP updates (`RegisterMSDPUpdateHook`), and hook MUD output lines (`RegisterMUDLineHook`).

3. **Built-in commands** — `#`-prefixed commands registered in `session/commands.go` via `ParseInternalCommand()`.

### Key Types

- `session.Session` — Holds socket, content buffer, Lua state, MSDP handler, all registries, and hook maps. Created via `SessionHandler.AddSession()`.
- `session.Action` — Regex-triggered callback on MUD output (can match with or without ANSI color codes).
- `session.Alias` — Regex-matched user input rewriter.
- `session.Event` — Named event handler (e.g., "core.prompt" fired on telnet Go Ahead).
- `session.TickerRecord` — Periodic timer with interval in milliseconds.
- `session.QueueItem` — Priority queue item with optional dependency chaining and condition checks.
- `layout.Layout` — Root of the pane tree. `FindPane(id)` to locate panes, `Split()`/`Unsplit()` to modify.
- `protocol.MSDPHandler` — Thread-safe (RWMutex) MSDP data store with typed getters.

### CGO Dependency

SQLite (`mattn/go-sqlite3`) requires CGO. The ringlog system uses an in-memory SQLite database as a circular buffer (10k lines, WAL mode). This affects cross-compilation — Windows builds need a cross-compiler (`mingw-w64-gcc`).

### Configuration Paths

All configuration follows XDG Base Directory spec. Key functions in `config/xdg.go`: `GetConfigDir()`, `GetSessionDir()`, `GetSessionModulesDir()`. Default base: `~/.config/zif/`.

### Debugging

- Zif logs to `debug.log` in the working directory (via `tea.LogToFile`)
- Panics are caught and logged to `~/.config/zif/panic.log`
- See `DEBUGGING.md` for Delve debugger instructions
- See `LAYOUT.md` for the split-pane layout system documentation
