# Debugging Zif with Delve

Delve (dlv) is the Go debugger, similar to gdb for C/C++. Here's how to use it to debug panics and crashes in Zif.

## Installation

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Basic Usage

### 1. Start Zif under Delve

**Important**: Zif uses a terminal UI (bubbletea), so output may not be visible in the debugger. See "Terminal UI Output" section below for solutions.

```bash
dlv debug .
```

This will:
- Compile Zif with debug symbols
- Start the debugger
- Drop you into a debugger prompt

**Note**: You may not see the terminal UI output when running under Delve. See solutions below.

### 2. Set breakpoints

At the dlv prompt:
```
(dlv) break main.main
(dlv) break session.SessionTicker
(dlv) break session.(*Session).mudReader
```

Or set breakpoints on specific lines:
```
(dlv) break main.go:110
```

### 3. Run the program

```
(dlv) continue
# or just
(dlv) c
```

### 4. When a panic occurs

Delve will automatically stop at panics. You can then:

```
(dlv) stack          # Show stack trace
(dlv) locals         # Show local variables
(dlv) args           # Show function arguments
(dlv) print variable # Print a specific variable
(dlv) goroutines     # List all goroutines
```

### 5. Step through code

```
(dlv) next            # Step over (n)
(dlv) step            # Step into (s)
(dlv) stepout         # Step out (so)
(dlv) continue        # Continue execution (c)
```

## Advanced: Catching Panics

### Set a breakpoint on panic

```
(dlv) break runtime.gopanic
(dlv) continue
```

When a panic occurs, you'll be stopped at the panic point and can inspect the stack.

### Inspect panic value

When stopped at `runtime.gopanic`:
```
(dlv) print arg0      # The panic value
(dlv) stack           # Full stack trace
```

## Running with Arguments

To run with command-line flags:

```bash
dlv debug . -- --kallisti
```

Or set them in the debugger:
```
(dlv) args --kallisti
(dlv) continue
```

## Attaching to Running Process

If Zif is already running:

1. Find the PID: `ps aux | grep zif`
2. Attach: `dlv attach <PID>`

Note: This requires appropriate permissions.

## Useful Commands

- `help` - Show all commands
- `list` - Show source code around current location
- `vars` - Show package variables
- `regs` - Show CPU registers
- `disassemble` - Show assembly code
- `trace <function>` - Set a tracepoint (logs when function is called)
- `restart` - Restart the program

## Example Session

```bash
$ dlv debug .
Type 'help' for list of commands.
(dlv) break main.main
Breakpoint 1 set at 0x... for main.main() ./main.go:546
(dlv) continue
> main.main() ./main.go:546:1 (hits goroutine(1):1 total:1) (PC: 0x...)
(dlv) break runtime.gopanic
Breakpoint 2 set at 0x... for runtime.gopanic() /usr/lib/go/src/runtime/panic.go:...
(dlv) continue
# ... program runs ...
# When panic occurs:
> runtime.gopanic() /usr/lib/go/src/runtime/panic.go:... (PC: 0x...)
(dlv) stack
(dlv) print arg0
(dlv) goroutines
```

## Terminal UI Output Issues

When debugging terminal UIs like Zif, you may not see output in the debugger. Here are solutions:

### Option 1: Attach to Running Process (Recommended)

Run Zif normally in one terminal, then attach Delve from another:

**Terminal 1:**
```bash
./zif
# Note the PID when it starts
```

**Terminal 2:**
```bash
# Find the PID
ps aux | grep zif

# Attach Delve
dlv attach <PID>
(dlv) break runtime.gopanic
(dlv) continue
```

### Option 2: Use Headless Mode

Run Delve in headless mode and connect with a client:

**Terminal 1:**
```bash
dlv debug --headless --listen=:2345 --api-version=2 .
```

**Terminal 2:**
```bash
dlv connect :2345
(dlv) break runtime.gopanic
(dlv) continue
```

### Option 3: Redirect Output

See output in a log file while debugging:

```bash
dlv debug . 2>&1 | tee debug-output.log
```

Or in the debugger, check the log file:
```bash
tail -f debug.log  # Zif's debug log
tail -f ~/.config/zif/panic.log  # Panic log
```

### Option 4: Use Terminal Flag

Try specifying a terminal explicitly:

```bash
dlv debug --tty=/dev/tty .
```

### Option 5: Check Logs Instead

Since Zif logs to files, you can monitor those while debugging:

```bash
# In one terminal, monitor logs
tail -f debug.log ~/.config/zif/panic.log

# In another, run under debugger
dlv debug .
```

## Alternative: Using GDB

Go programs can also be debugged with gdb, though Delve is recommended:

```bash
go build -gcflags="-N -l"  # Build with debug symbols, no optimizations
gdb ./zif
(gdb) run
(gdb) bt                    # Backtrace when it crashes
```

**Note**: GDB also has issues with terminal UIs. The attach method works better.

## Debugging Runtime Fatal Errors

If you hit `runtime.fatal()` (more serious than a panic), gather this information:

```bash
# 1. Get the fatal error message
(dlv) print s

# 2. Get full stack trace
(dlv) stack

# 3. See all goroutines and their states
(dlv) goroutines

# 4. Check if there's a panic value
(dlv) print runtime.g

# 5. Get more context - go up the stack
(dlv) up
(dlv) locals
(dlv) args

# 6. Continue up to see the full call chain
(dlv) up
(dlv) stack
```

Common fatal errors:
- `concurrent map writes` - Multiple goroutines writing to a map without sync
- `index out of range` - Array/slice bounds violation
- `nil pointer dereference` - Accessing nil pointer
- `deadlock` - All goroutines are blocked

The stack trace will show where the fatal error originated.

