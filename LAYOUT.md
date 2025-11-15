# Flexible Split-Screen Layout System

This document describes the new flexible split-screen layout system for Zif, inspired by Tintin++'s split and draw commands but integrated with Charm's Bubble Tea framework.

## Overview

The layout system allows you to dynamically split the screen into multiple panes, similar to how Tintin++ works. You can create horizontal (left/right) or vertical (top/bottom) splits, add different types of panes (comms windows, sidebars, graphs), and manage them all through simple commands.

## Features

- **Dynamic Splitting**: Split any pane horizontally or vertically
- **Multiple Pane Types**: Viewport (main MUD output), Comms (logs), Sidebar (info), Graph (charts)
- **Flexible Layout**: Tree-based structure allows nested splits
- **Command-Based Control**: Use `#split`, `#unsplit`, `#panes`, `#focus` commands
- **Backward Compatible**: Toggle between old and new layout systems with F4

## Usage

### Enabling the Layout System

Press **F4** to toggle between the legacy layout system and the new flexible layout system.

### Commands

#### `#split` - Split a pane

Split a pane into two panes.

**Usage:**
```
#split [h|v] [pane_id] [type] [split_percent]
```

**Parameters:**
- `h` or `v`: Direction - `h` for horizontal (left/right), `v` for vertical (top/bottom)
- `pane_id`: ID of the pane to split (default: "main")
- `type`: Type of new pane - `viewport`, `comms`, `sidebar`, or `graph` (default: `sidebar`)
- `split_percent`: Percentage for split position, 5-95 (default: 50)

**Examples:**
```
#split h main comms 30          # Split main pane horizontally, 30% left, add comms pane
#split v main sidebar 25         # Split main pane vertically, 25% top, add sidebar
#split h                         # Split active pane horizontally at 50%
```

#### `#unsplit` - Remove a pane

Remove a pane and merge its space with its sibling.

**Usage:**
```
#unsplit <pane_id>
```

**Example:**
```
#unsplit sidebar_1              # Remove the sidebar_1 pane
```

#### `#panes` - List all panes

Show a list of all panes with their IDs, types, and sizes.

**Usage:**
```
#panes
```

#### `#pane` - Show pane information

Display detailed information about a specific pane.

**Usage:**
```
#pane <pane_id>
```

**Example:**
```
#pane main
```

#### `#focus` - Set active pane

Set which pane is currently active (receives keyboard input for scrolling).

**Usage:**
```
#focus <pane_id>
```

**Example:**
```
#focus comms_1
```

## Pane Types

- **viewport**: Main MUD output (default for main pane)
- **comms**: Communications/logs window
- **sidebar**: Information sidebar
- **graph**: Graph/chart display (for future use)

## Examples

### Example 1: Add a Comms Window

```
#split h main comms 30
```

This splits the main pane horizontally, giving 30% to a new comms pane on the left and 70% to the main viewport on the right.

### Example 2: Add a Sidebar

```
#split v main sidebar 20
```

This splits the main pane vertically, giving 20% to a sidebar at the top and 80% to the main viewport at the bottom.

### Example 3: Complex Layout

```
#split h main comms 25          # Split main: 25% comms, 75% main
#split v comms_1 sidebar 30     # Split comms: 30% sidebar, 70% comms
#focus main                     # Focus back on main viewport
```

This creates a three-pane layout with a sidebar, comms window, and main viewport.

## Technical Details

### Architecture

The layout system uses a tree structure:
- **Leaf nodes** are `Pane` objects containing viewports
- **Internal nodes** are `Container` objects that split space between children
- The root is either a single `Pane` or a `Container`

### Integration

- The layout system is integrated into `main.go` but can be toggled on/off
- When enabled, it replaces the legacy left/right sidebar system
- Session content updates automatically go to the "main" pane
- All panes support viewport scrolling with standard keys (pgup, pgdown, home, end)

### Future Enhancements

- Resizable splits (drag to adjust)
- Pane titles and borders
- Custom render functions for graph panes
- Save/load layout configurations
- Mouse support for pane selection

## Keyboard Shortcuts

- **F4**: Toggle between legacy and new layout systems
- **F2/F3**: Toggle sidebars (legacy system only)
- **PgUp/PgDn/Home/End**: Scroll active pane viewport

