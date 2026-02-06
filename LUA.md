# Lua Module Guide

Zif supports Lua modules for extending functionality with triggers, aliases, timers, events, and layout control.

## Module Structure

Each module is a directory containing an `init.lua` entry point, with optional subdirectories:

```
MyModule/
├── init.lua           # Entry point (required)
├── triggers/
│   └── my_trigger.lua
├── aliases/
│   └── my_alias.lua
└── scripts/
    └── my_script.lua
```

All `.lua` files in subdirectories are automatically loaded after `init.lua`.

## Module Locations

Modules are loaded from two locations, in order:

1. **Global modules** — `~/.config/zif/modules/MyModule/`
   Loaded for every session.

2. **Session modules** — `~/.config/zif/sessions/<session-name>/modules/MyModule/`
   Loaded only for that session. Can override or extend global modules.

## Managing Modules

```
#modules              List all loaded modules
#modules enable Name  Enable a disabled module
#modules disable Name Disable a module (disables its triggers, aliases, and timers)
```

## API Reference

### Session Functions

#### `session.send(command)`
Send a command to the MUD server (appends CR LF automatically).
```lua
session.send("kill dragon")
session.send("cast 'fireball'")
```

#### `session.output(text)`
Display text in the session output window. Not sent to the MUD.
```lua
session.output("Hello from Lua!\n")
session.output("\x1b[1;31mRed text\x1b[0m\n")  -- ANSI colors work
```

#### `session.get_data(key)` / `session.set_data(key, value)`
Read/write session-level key-value storage. Shared between Lua and Go plugins.
```lua
session.set_data("my_counter", 42)
local count = session.get_data("my_counter")
```

### Triggers

#### `session.register_trigger(name, pattern, callback, color)`
Fire a callback when MUD output matches a regex pattern.

Parameters:
- `name` — unique trigger name
- `pattern` — Go-style regex (not PCRE)
- `callback(ansi_line, line, matches)` — function called on match
  - `ansi_line` — the full line with ANSI color codes
  - `line` — the line with ANSI codes stripped
  - `matches` — table of regex capture groups (`matches[1]` is the full match)
- `color` — `true` to match against the ANSI line, `false` (default) to match against stripped text

```lua
-- Simple trigger
session.register_trigger("gold_pickup", "You receive (\\d+) gold", function(ansi, line, matches)
    session.output("Got " .. matches[2] .. " gold!\n")
end, false)

-- Color-aware trigger (matches ANSI escape sequences in the pattern)
session.register_trigger("red_text", "\\x1b\\[1;31m", function(ansi, line, matches)
    session.output("Saw red text!\n")
end, true)
```

### Aliases

#### `session.register_alias(name, pattern, callback)`
Intercept user input matching a regex pattern. If an alias matches, the input is consumed and not sent to the MUD.

Parameters:
- `name` — unique alias name
- `pattern` — Go-style regex matched against user input
- `callback(matches)` — function called on match; `matches[1]` is the full match

```lua
-- Simple alias
session.register_alias("go_home", "^gohome$", function(matches)
    session.send("recall")
    session.send("north")
    session.send("enter portal")
end)

-- Alias with capture groups
session.register_alias("repeat_cmd", "^do (\\d+) (.+)$", function(matches)
    local count = tonumber(matches[2])
    local cmd = matches[3]
    for i = 1, count do
        session.send(cmd)
    end
end)
```

### Timers

#### `session.add_timer(name, interval_ms, callback)`
Create a repeating timer.

```lua
session.add_timer("my_ticker", 5000, function()
    session.send("look")
end)
```

#### `session.add_one_shot_timer(name, delay_ms, callback)`
Create a timer that fires once then removes itself.

```lua
session.add_one_shot_timer("delayed_action", 2000, function()
    session.output("Two seconds have passed!\n")
end)
```

#### `session.remove_timer(name)`
Cancel a running timer.

```lua
session.remove_timer("my_ticker")
```

### Events

#### `session.register_event(event_name, callback)`
Listen for named events fired by Go code or plugins. The callback receives event data as a Lua table.

```lua
-- Built-in event: fires on every MUD prompt (telnet Go Ahead)
session.register_event("core.prompt", function(evt)
    -- evt is a table with event-specific fields
end)

-- Kallisti plugin events (when kallisti plugin is active):
session.register_event("kallisti.craft", function(evt)
    session.output("Crafted: " .. evt.Output .. " from " .. evt.Input .. "\n")
end)

session.register_event("kallisti.death", function(evt)
    session.output(evt.Name .. " died!\n")
end)
```

### MSDP (MUD Server Data Protocol)

Access real-time game data provided by the MUD server.

```lua
session.msdp_get_string("ROOM_NAME")       -- returns string
session.msdp_get_int("HEALTH")             -- returns number
session.msdp_get_bool("IN_COMBAT")         -- returns boolean
session.msdp_get_array("EXITS")            -- returns table (1-indexed)
session.msdp_get_table("GROUP")            -- returns table (string keys)
session.msdp_get_all()                     -- returns table of all MSDP data
```

### Layout Control

Create and manage split-screen panes from Lua.

#### `session.layout_split(direction, pane_id, pane_type, split_percent)`
Split a pane. Returns the new pane's auto-generated ID.
- `direction` — `"h"` (horizontal/left-right) or `"v"` (vertical/top-bottom)
- `pane_id` — pane to split (default: `"main"`)
- `pane_type` — `"viewport"`, `"comms"`, `"sidebar"`, or `"graph"`
- `split_percent` — percentage for the original pane (5-95)

```lua
local sidebar_id = session.layout_split("h", "main", "sidebar", 70)
```

#### `session.layout_unsplit(pane_id)`
Remove a pane.
```lua
session.layout_unsplit("sidebar_1")
```

#### `session.layout_set_content(pane_id, content)`
Set the text content of a pane.
```lua
session.layout_set_content("sidebar_1", "Hello from sidebar!\nLine 2\n")
```

#### `session.layout_set_border(pane_id, border_type, color)`
Set pane border style. `border_type`: `"normal"`, `"rounded"`, `"thick"`, `"double"`, `"none"`. `color` is optional (e.g., `"#ff0000"`).

#### `session.layout_focus(pane_id)`
Set which pane receives keyboard scrolling (PgUp/PgDn/Home/End).

#### `session.layout_list_panes()` / `session.layout_pane_info(pane_id)`
List all panes or get details about a specific pane. Output goes to the session window.

### Progress Bars

Create animated progress bars inside panes.

```lua
session.progress_create("sidebar_1", 40)     -- create with width 40
session.progress_update("sidebar_1", 0.5)    -- set to 50% (0.0 to 1.0)
session.progress_destroy("sidebar_1")        -- remove
```

### Module Metadata

```lua
local name = module.get_name()  -- current module name (directory name)
local path = module.get_path()  -- full filesystem path to module directory
```

## Regex Notes

Zif uses Go's `regexp` package, not PCRE. Key differences from Perl/PCRE regex:
- No lookahead/lookbehind (`(?=...)`, `(?!...)`)
- No backreferences (`\1`)
- No possessive quantifiers (`a++`)
- Backslashes must be doubled in Lua strings: `"\\d+"` for `\d+`
- Named groups use `(?P<name>...)` syntax

## Example: Complete Module

```
~/.config/zif/modules/AutoQuaff/
├── init.lua
└── triggers/
    └── low_health.lua
```

**init.lua:**
```lua
session.output("AutoQuaff module loaded!\n")
session.set_data("autoquaff_threshold", 50)
```

**triggers/low_health.lua:**
```lua
session.register_trigger("autoquaff", "Your health: (\\d+)/(\\d+)", function(ansi, line, matches)
    local current = tonumber(matches[2])
    local max = tonumber(matches[3])
    local threshold = session.get_data("autoquaff_threshold") or 50
    if max > 0 and (current / max * 100) < threshold then
        session.send("quaff heal")
        session.output("[AutoQuaff] Health low, quaffing!\n")
    end
end, false)
```

## Inspecting Registered Items

Use these commands to see what's currently active:

```
#actions    List all triggers with their patterns, enabled status, and fire count
#aliases    List all aliases
#tickers    List all timers
#events     List all event handlers
#modules    List all modules
```
