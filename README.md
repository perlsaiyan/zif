# Zif, the quick little MUD client

A modern MUD client written in Go with Lua scripting support, built on bubbletea for a terminal-based UI.

## Features

- **Lua-based Module System**: Extend functionality with Lua modules for triggers, aliases, and scripts
- **XDG Directory Support**: Follows XDG Base Directory specification for configuration
- **MSDP Support**: Automatic parsing and handling of Mud Server Data Protocol
- **Session Management**: Multiple simultaneous MUD connections
- **Command Echo**: Commands are displayed in bright white in the output window
- **Module Management**: Enable/disable modules on the fly

## Installation

Build from source:
```bash
go build .
```

### Command-Line Flags

- `--kallisti` - Load the Kallisti plugin for Legends of Kallisti MUD
- `--no-autostart` - Skip auto-loading sessions from `sessions.yaml` at startup
- `--help` - Show help message

## Configuration

Zif uses XDG directories for configuration:
- **Config Directory**: `$XDG_CONFIG_HOME/zif` (defaults to `~/.config/zif`)
- **Global Modules**: `$XDG_CONFIG_HOME/zif/modules/`
- **Session Configs**: `$XDG_CONFIG_HOME/zif/sessions/<session-name>/`
- **Session Modules**: `$XDG_CONFIG_HOME/zif/sessions/<session-name>/modules/`

### Auto-Start Sessions

Zif can automatically start MUD sessions at startup by configuring them in `~/.config/zif/sessions.yaml`. This file is automatically created with an empty configuration on first run.

**Configuration Format:**
```yaml
sessions:
  - name: "session1"
    address: "mud.example.com:4000"
    autostart: true
  - name: "session2"
    address: "another.mud.com:23"
    autostart: false
```

- `name`: The session name (required)
- `address`: The MUD server address in `host:port` format (required)
- `autostart`: Whether to automatically start this session at launch (default: `false`)

Only sessions with `autostart: true` will be automatically connected when Zif starts. You can skip auto-loading entirely by using the `--no-autostart` command-line flag.

## Lua Module System

Zif supports Lua modules for extending functionality. Modules are organized in directories with the following structure:

```
modules/
├── ModuleName/
│   ├── init.lua           # Entry point: Registers everything, sets metadata
│   ├── triggers/
│   │   └── trigger_name.lua  # Defines/registers triggers
│   ├── aliases/
│   │   └── alias_name.lua    # Defines/registers aliases
│   └── scripts/
│       └── script_name.lua   # Background scripts with timers
```

### Module Loading

- **Global modules** load first from `~/.config/zif/modules/`
- **Session-specific modules** load second from `~/.config/zif/sessions/<session-name>/modules/`
- Session modules can override or extend global modules

### Lua API

Modules have access to a comprehensive Lua API:

**Session Functions:**
- `session:send(command)` - Send command to MUD
- `session:output(text)` - Output text to session
- `session:get_data(key)` / `session:set_data(key, value)` - Session data storage
- `session:register_trigger(name, pattern, func, color)` - Register a trigger
- `session:register_alias(name, pattern, func)` - Register an alias
- `session:add_timer(name, interval_ms, func)` - Register a periodic timer
- `session:get_ringlog(limit)` - Read ringlog entries

**Module Functions:**
- `module:get_name()` - Get current module name
- `module:get_path()` - Get current module path

### Example Module

See `~/.config/zif/modules/SampleModule/` for a complete example.

**init.lua:**
```lua
local module_name = module.get_name()
session.output("SampleModule loaded: " .. module_name .. "\n")
```

**aliases/sample.lua:**
```lua
session.register_alias("sample", "^sample$", function(matches)
    session.output("Making me dance!\n")
    session.send("dance")
end)
```

## Commands

Zif provides several built-in commands (prefixed with `#`):

- `#help` - Show help for all commands
- `#session <name> [address:port]` - Create or switch to a session
- `#sessions` - List all sessions
- `#modules` - List all loaded modules
- `#modules enable <name>` - Enable a module
- `#modules disable <name>` - Disable a module
- `#actions` - List all triggers/actions
- `#aliases` - List all aliases
- `#tickers` - List all timers
- `#events` - List all event handlers
- `#queue` - Show command queue
- `#msdp` - Display MSDP data

## Kallisti Plugin

The Kallisti plugin provides enhanced features for playing Legends of Kallisti MUD. It includes automatic map display, room tracking, pathfinding, and automated travel.

### Installation

Build the plugin:
```bash
go build --buildmode=plugin ./plugins/kallisti
```

Run Zif with the plugin:
```bash
./zif -kallisti
```

### Features

- **Automatic Map Pane**: When the plugin is active, a map pane is automatically created on the right side of the screen showing your current location and surrounding area
- **Room Tracking**: Automatically scans and tracks room information as you move
- **Pathfinding**: Find optimal paths between rooms using `#path <vnum>`
- **Auto-Travel**: Automatically navigate to a destination room
- **Room Information**: Display detailed room information with `#room`
- **Map Display**: Show a text-based map with `#map`
- **Autoheal/Autobuf**: Automated health and buff management (configurable)

### Commands

- `#room` - Show information about the current room
- `#path <vnum>` - Find and travel to a room by VNUM
- `#map` - Display a text-based map in the output window

### Map Pane

The map pane automatically appears when:
- The kallisti plugin is loaded (`-kallisti` flag)
- You're connected to a MUD session

The map updates in real-time as you move, showing:
- Your current location (marked with `@`)
- Surrounding terrain
- Room connections

You can remove the map pane with `#unsplit map` or create it manually with `#split h main sidebar 30`.

## Building

### Cross compile for Windows
```bash
pacman -S mingw-w64-gcc
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build --buildmode=exe
```


