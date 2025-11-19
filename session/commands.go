package session

import (
	"encoding/csv"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/perlsaiyan/zif/layout"
)

type CommandFunction func(*Session, string)

type Command struct {
	Name string
	Fn   CommandFunction
}

var internalCommands = []Command{
	{Name: "actions", Fn: CmdActions},
	{Name: "aliases", Fn: CmdAliases},
	{Name: "cancel", Fn: CmdCancelTicker},
	{Name: "events", Fn: CmdEvents},
	{"help", CmdHelp},
	{Name: "modules", Fn: CmdModules},
	{Name: "msdp", Fn: CmdMSDP},
	{Name: "pane", Fn: nil},  // Layout command, handled separately
	{Name: "panes", Fn: nil}, // Layout command, handled separately
	{Name: "plugins", Fn: CmdPlugins},
	{Name: "queue", Fn: CmdQueue},
	{Name: "ringtest", Fn: CmdRingtest},
	{Name: "session", Fn: CmdSession},
	{Name: "sessions", Fn: CmdSessions},
	{Name: "split", Fn: nil},   // Layout command, handled separately
	{Name: "unsplit", Fn: nil}, // Layout command, handled separately
	{Name: "focus", Fn: nil},   // Layout command, handled separately
	{Name: "test", Fn: CmdTestTicker},
	{Name: "tickers", Fn: CmdTickers},
}

var internalCommandHelp = map[string]string{
	"aliases":  "Show aliases",
	"cancel":   "Cancel test for timers",
	"focus":    "Set active pane: #focus <pane_id>",
	"help":     "This help command",
	"modules":  "Show modules or enable/disable: #modules [enable|disable] <name>",
	"msdp":     "Show MSDP values",
	"pane":     "Show pane info: #pane <pane_id>",
	"panes":    "List all panes",
	"session":  "Usage: #session <name> <host:port>",
	"sessions": "Show current sessions",
	"split":    "Split pane: #split [h|v] [pane_id] [type] [percent]",
	"test":     "Just a test command/playground",
	"tickers":  "Show tickers",
	"unsplit":  "Remove pane: #unsplit <pane_id>",
}

func (s *Session) AddCommand(c Command, help string) {
	internalCommands = append(internalCommands, c)
	internalCommandHelp[c.Name] = help

	// sort internal commands alphabetically by Name
	sort.Slice(internalCommands, func(i, j int) bool {
		return internalCommands[i].Name < internalCommands[j].Name
	})
}

func formatMSDPValue(v interface{}, indent int) string {
	// Safety: prevent infinite recursion or excessive depth
	const maxDepth = 10
	if indent > maxDepth {
		return "... (max depth exceeded)"
	}

	indentStr := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		// Limit array size to prevent huge output
		const maxArrayItems = 100
		displayItems := val
		truncated := false
		if len(val) > maxArrayItems {
			displayItems = val[:maxArrayItems]
			truncated = true
		}

		// For simple arrays of strings/ints, show inline
		allSimple := true
		for _, item := range displayItems {
			switch item.(type) {
			case string, int, bool:
				// simple
			default:
				allSimple = false
				break
			}
		}
		if allSimple && len(displayItems) <= 5 {
			var items []string
			for _, item := range displayItems {
				items = append(items, formatMSDPValue(item, 0))
			}
			result := "[" + strings.Join(items, ", ") + "]"
			if truncated {
				result += fmt.Sprintf(" ... (%d more items)", len(val)-maxArrayItems)
			}
			return result
		}
		// Complex or long array - format multi-line
		var lines []string
		lines = append(lines, "[")
		for i, item := range displayItems {
			itemStr := formatMSDPValue(item, indent+1)
			if i == len(displayItems)-1 {
				lines = append(lines, fmt.Sprintf("%s  %s", indentStr, itemStr))
			} else {
				lines = append(lines, fmt.Sprintf("%s  %s,", indentStr, itemStr))
			}
		}
		if truncated {
			lines = append(lines, fmt.Sprintf("%s  ... (%d more items)", indentStr, len(val)-maxArrayItems))
		}
		lines = append(lines, fmt.Sprintf("%s]", indentStr))
		return strings.Join(lines, "\n")
	case map[string]interface{}:
		if len(val) == 0 {
			return "{}"
		}
		// Limit table size to prevent huge output
		const maxTableItems = 50
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		displayKeys := keys
		truncated := false
		if len(keys) > maxTableItems {
			displayKeys = keys[:maxTableItems]
			truncated = true
		}

		// Always format tables multi-line for readability
		var lines []string
		lines = append(lines, "{")
		for i, k := range displayKeys {
			itemStr := formatMSDPValue(val[k], indent+1)
			if i == len(displayKeys)-1 && !truncated {
				lines = append(lines, fmt.Sprintf("%s  %s: %s", indentStr, k, itemStr))
			} else {
				lines = append(lines, fmt.Sprintf("%s  %s: %s,", indentStr, k, itemStr))
			}
		}
		if truncated {
			lines = append(lines, fmt.Sprintf("%s  ... (%d more keys)", indentStr, len(keys)-maxTableItems))
		}
		lines = append(lines, fmt.Sprintf("%s}", indentStr))
		return strings.Join(lines, "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func CmdMSDP(s *Session, cmd string) {
	// Get a safe copy of the data to avoid concurrent access
	data := s.MSDP.GetAllData()

	log.Printf("CmdMSDP: Starting, data has %d keys", len(data))

	if len(data) == 0 {
		s.Output("No MSDP data available.\n")
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	log.Printf("CmdMSDP: Processing %d keys", len(keys))

	// Build the entire output string first to avoid blocking on channel sends
	var output strings.Builder
	output.WriteString("MSDP Values:\n")

	for i, key := range keys {
		log.Printf("CmdMSDP: Processing key %d/%d: %s", i+1, len(keys), key)
		value := data[key]

		log.Printf("CmdMSDP: Formatting value for %s", key)
		formatted := formatMSDPValue(value, 0)
		log.Printf("CmdMSDP: Formatted %s, length %d chars", key, len(formatted))

		// Handle multi-line values by indenting subsequent lines
		lines := strings.Split(formatted, "\n")
		if len(lines) == 1 {
			// Single line value
			output.WriteString(fmt.Sprintf("  %s: %s\n", key, formatted))
		} else {
			// Multi-line value - indent all lines after the first
			output.WriteString(fmt.Sprintf("  %s: %s\n", key, lines[0]))
			for _, line := range lines[1:] {
				output.WriteString(fmt.Sprintf("    %s\n", line))
			}
		}
		log.Printf("CmdMSDP: Finished key %s", key)
	}

	log.Printf("CmdMSDP: Completed all keys, sending output (%d bytes)", output.Len())
	// Send the entire output in one go to avoid blocking
	s.Output(output.String())
	log.Printf("CmdMSDP: Output sent")
}

func CmdTest(s *Session, cmd string) {
	r := csv.NewReader(strings.NewReader(cmd))
	r.Comma = ' '
	r.LazyQuotes = true
	record, err := r.Read()
	if err != nil {
		log.Printf("Error: %v", err)
	}

	out := strings.Join(record, ", ")
	msg := fmt.Sprintf("Got the args: %v\n", out)
	s.Output(msg)
}

func CmdSession(s *Session, cmd string) {
	h := s.Handler
	fields := strings.Fields(cmd)

	// Handle no arguments
	if len(fields) < 1 {
		s.Output("Usage: #session <name> <address:port>" + "\n")
		return
	}

	// Handle too many arguments
	if len(fields) > 2 {
		s.Output("Usage: #session <name> <address:port>" + "\n")
		return
	}

	sessionName := fields[0]

	// Single argument: switch to existing session
	if len(fields) == 1 {
		if _, exists := h.Sessions[sessionName]; !exists {
			s.Output(fmt.Sprintf("Invalid session: %s\n", sessionName))
			return
		}

		h.Active = sessionName
		activeSession := h.ActiveSession()
		if activeSession == nil {
			// Revert active session if lookup fails (shouldn't happen, but be safe)
			s.Output("Error: Could not activate session.\n")
			return
		}

		s.Sub <- SessionChangeMsg{ActiveSession: activeSession}
		return
	}

	// Two arguments: create new session
	// Note: AddSession already validates session name format, so we don't need to duplicate it
	address := fields[1]

	err := h.AddSession(sessionName, address)
	if err != nil {
		s.Output(fmt.Sprintf("Error creating session: %v\n", err))
		return
	}

	// Verify session exists in map (more reliable than checking ActiveSession before setting Active)
	if _, exists := h.Sessions[sessionName]; !exists {
		s.Output("Error: Session was created but not found in session map.\n")
		return
	}

	// Set as active and notify
	h.Active = sessionName
	activeSession := h.ActiveSession()
	if activeSession == nil {
		// This shouldn't happen if the session exists, but handle it gracefully
		s.Output("Error: Session exists but could not be activated.\n")
		return
	}

	s.Sub <- SessionChangeMsg{ActiveSession: activeSession}
}
func CmdHelp(s *Session, cmd string) {
	msg := "Commands:\n"

	var sortedHelp []string
	for k := range internalCommandHelp {
		sortedHelp = append(sortedHelp, k)
	}
	sort.Strings(sortedHelp)
	s.Output(msg)
	for _, v := range sortedHelp {
		msg = fmt.Sprintf("%+15s: %-40s\n", v, internalCommandHelp[v])
		s.Output(msg)
	}

}

func makeRow(name string, address string, start time.Time) table.Row {

	return table.NewRow(table.RowData{
		"name":    name,
		"address": address,
		"time":    time.Since(start).Round(time.Second),
	})
}

func CmdSessions(s *Session, cmd string) {
	h := s.Handler
	var rows []table.Row
	for i := range h.Sessions {
		if h.Sessions[i].Name == h.ActiveSession().Name {
			rows = append(rows, makeRow("> "+h.Sessions[i].Name, h.Sessions[i].Address, h.Sessions[i].Birth))
		} else {
			rows = append(rows, makeRow("  "+h.Sessions[i].Name, h.Sessions[i].Address, h.Sessions[i].Birth))
		}
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("address", "Address", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("time", "Uptime", 30).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}

func CmdModules(s *Session, cmd string) {
	fields := strings.Fields(cmd)

	if len(fields) == 0 {
		// List all modules
		var rows []table.Row
		for _, module := range s.Modules.Modules {
			enabledStr := "disabled"
			if module.Enabled {
				enabledStr = "enabled"
			}
			rows = append(rows, table.NewRow(table.RowData{
				"name":     module.Name,
				"path":     module.Path,
				"enabled":  enabledStr,
				"triggers": len(module.Triggers),
				"aliases":  len(module.Aliases),
				"timers":   len(module.Timers),
			}))
		}

		t := table.New([]table.Column{
			table.NewColumn("name", "Name", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
			table.NewColumn("path", "Path", 40).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
			table.NewColumn("enabled", "Status", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
			table.NewColumn("triggers", "Triggers", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
			table.NewColumn("aliases", "Aliases", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
			table.NewColumn("timers", "Timers", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		}).
			WithRows(rows).
			BorderRounded()

		s.Output(t.View() + "\n")
	} else if len(fields) >= 2 {
		// Enable or disable module
		action := strings.ToLower(fields[0])
		moduleName := fields[1]

		switch action {
		case "enable":
			if err := s.EnableModule(moduleName); err != nil {
				s.Output(fmt.Sprintf("Error enabling module %s: %v\n", moduleName, err))
			} else {
				s.Output(fmt.Sprintf("Enabled module: %s\n", moduleName))
			}
		case "disable":
			if err := s.DisableModule(moduleName); err != nil {
				s.Output(fmt.Sprintf("Error disabling module %s: %v\n", moduleName, err))
			} else {
				s.Output(fmt.Sprintf("Disabled module: %s\n", moduleName))
			}
		default:
			s.Output("Usage: #modules [enable|disable] <name>\n")
		}
	} else {
		s.Output("Usage: #modules [enable|disable] <name>\n")
	}
}

func (s *Session) ParseInternalCommand(cmd string) {
	// Note: Command has already been added to Content (colored) in HandleInput()
	// so we don't add it again here to avoid duplication
	parsed := strings.Fields(cmd[1:])
	args := strings.SplitN(cmd, " ", 2)

	// Handle layout commands first (they're handled in main.go via messages)
	cmdName := strings.ToLower(parsed[0])
	if cmdName == "split" {
		CmdLayoutSplit(s, strings.Join(parsed[1:], " "))
		s.Sub <- UpdateMessage{Session: s.Name}
		return
	} else if cmdName == "unsplit" {
		CmdLayoutUnsplit(s, strings.Join(parsed[1:], " "))
		s.Sub <- UpdateMessage{Session: s.Name}
		return
	} else if cmdName == "panes" {
		CmdLayoutPanes(s, "")
		s.Sub <- UpdateMessage{Session: s.Name}
		return
	} else if cmdName == "pane" {
		CmdLayoutPaneInfo(s, strings.Join(parsed[1:], " "))
		s.Sub <- UpdateMessage{Session: s.Name}
		return
	} else if cmdName == "focus" {
		CmdLayoutFocus(s, strings.Join(parsed[1:], " "))
		s.Sub <- UpdateMessage{Session: s.Name}
		return
	}

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, cmdName) {
			if internalCommands[lookup].Fn == nil {
				// Layout command, skip
				continue
			}
			if len(args) < 2 {
				internalCommands[lookup].Fn(s, "")
				s.Sub <- UpdateMessage{Session: s.Name}
				return
			} else {
				internalCommands[lookup].Fn(s, args[1])
				s.Sub <- UpdateMessage{Session: s.Name}
				return
			}
		}
	}
	s.Sub <- UpdateMessage{Session: s.Name}
}

// Layout command functions

func CmdLayoutSplit(s *Session, cmd string) {
	fields := strings.Fields(cmd)
	if len(fields) < 1 {
		s.Output("Usage: #split [h|v] [pane_id] [type] [split_percent]\n")
		s.Output("  h = horizontal (left/right), v = vertical (top/bottom)\n")
		s.Output("  pane_id = ID of pane to split (default: active pane)\n")
		s.Output("  type = viewport|comms|sidebar|graph (default: sidebar)\n")
		s.Output("  split_percent = 5-95 (default: 50)\n")
		s.Output("Example: #split h main comms 30\n")
		return
	}

	directionStr := strings.ToLower(fields[0])
	var direction layout.SplitDirection
	if directionStr == "h" || directionStr == "horizontal" {
		direction = layout.SplitHorizontal
	} else if directionStr == "v" || directionStr == "vertical" {
		direction = layout.SplitVertical
	} else {
		s.Output("Direction must be 'h' (horizontal) or 'v' (vertical)\n")
		return
	}

	paneID := "main"
	paneType := layout.PaneTypeSidebar
	splitPercent := 50

	if len(fields) >= 2 {
		paneID = fields[1]
	}
	if len(fields) >= 3 {
		paneType = layout.ParsePaneType(fields[2])
	}
	if len(fields) >= 4 {
		percent, err := strconv.Atoi(fields[3])
		if err != nil || percent < 5 || percent > 95 {
			s.Output("Split percentage must be between 5 and 95\n")
			return
		}
		splitPercent = percent
	}

	// Generate new pane ID
	newPaneID := layout.GeneratePaneID(string(paneType))

	s.Output(fmt.Sprintf("Splitting pane %s %s at %d%%\n", paneID, direction, splitPercent))

	// Send layout command message
	s.Sub <- layout.LayoutCommandMsg{
		Command: "split",
		Args:    []string{paneID, newPaneID, string(direction), fmt.Sprintf("%d", splitPercent), string(paneType)},
		Session: s,
	}
}

func CmdLayoutUnsplit(s *Session, cmd string) {
	fields := strings.Fields(cmd)
	paneID := ""

	if len(fields) >= 1 {
		paneID = fields[0]
	}

	if paneID == "" {
		s.Output("Usage: #unsplit [pane_id]\n")
		s.Output("  Removes the specified pane and merges its space\n")
		return
	}

	s.Output(fmt.Sprintf("Removing pane %s\n", paneID))

	s.Sub <- layout.LayoutCommandMsg{
		Command: "unsplit",
		Args:    []string{paneID},
		Session: s,
	}
}

func CmdLayoutPanes(s *Session, cmd string) {
	s.Sub <- layout.LayoutCommandMsg{
		Command: "list",
		Args:    []string{},
		Session: s,
	}
}

func CmdLayoutPaneInfo(s *Session, cmd string) {
	fields := strings.Fields(cmd)
	paneID := ""

	if len(fields) >= 1 {
		paneID = fields[0]
	}

	if paneID == "" {
		s.Output("Usage: #pane [pane_id]\n")
		s.Output("  Shows information about a specific pane\n")
		return
	}

	s.Sub <- layout.LayoutCommandMsg{
		Command: "info",
		Args:    []string{paneID},
		Session: s,
	}
}

func CmdLayoutFocus(s *Session, cmd string) {
	fields := strings.Fields(cmd)
	paneID := ""

	if len(fields) >= 1 {
		paneID = fields[0]
	}

	if paneID == "" {
		s.Output("Usage: #focus [pane_id]\n")
		s.Output("  Sets the active pane\n")
		return
	}

	s.Output(fmt.Sprintf("Focusing pane %s\n", paneID))

	s.Sub <- layout.LayoutCommandMsg{
		Command: "focus",
		Args:    []string{paneID},
		Session: s,
	}
}

func (s *Session) ParseCommand(cmd string) {
	// Note: Command has already been added to Content (colored) in HandleInput()
	// and an UpdateMessage was already sent, so we don't need to send another one

	// TODO: We'll want to check this for aliases and/or variables
	if s.Connected {
		s.Socket.Write([]byte(cmd + LineTerminator))
	}

	// No need to send UpdateMessage here - Output() already sent one with the colored command
}
