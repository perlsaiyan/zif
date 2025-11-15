package session

import (
	"encoding/csv"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
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
	{Name: "plugins", Fn: CmdPlugins},
	{Name: "queue", Fn: CmdQueue},
	{Name: "ringtest", Fn: CmdRingtest},
	{Name: "session", Fn: CmdSession},
	{Name: "sessions", Fn: CmdSessions},
	{Name: "test", Fn: CmdTestTicker},
	{Name: "tickers", Fn: CmdTickers},
}

var internalCommandHelp = map[string]string{
	"aliases":  "Show aliases",
	"cancel":   "Cancel test for timers",
	"help":     "This help command",
	"modules":  "Show modules or enable/disable: #modules [enable|disable] <name>",
	"msdp":     "Show MSDP values",
	"session":  "Usage: #session <name> <host:port>",
	"sessions": "Show current sessions",
	"test":     "Just a test command/playground",
	"tickers":  "Show tickers",
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
	log.Printf("CmdMSDP: Starting, data has %d keys", len(s.MSDP.Data))
	
	if len(s.MSDP.Data) == 0 {
		s.Output("No MSDP data available.\n")
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(s.MSDP.Data))
	for k := range s.MSDP.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	log.Printf("CmdMSDP: Processing %d keys", len(keys))
	
	// Build the entire output string first to avoid blocking on channel sends
	var output strings.Builder
	output.WriteString("MSDP Values:\n")
	
	for i, key := range keys {
		log.Printf("CmdMSDP: Processing key %d/%d: %s", i+1, len(keys), key)
		value := s.MSDP.Data[key]
		
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
	if len(fields) < 1 {
		s.Output("Usage: #session <name> <address:port>" + "\n")
		return
	} else if len(fields) == 1 {
		_, ok := h.Sessions[fields[0]]
		if ok {
			h.Active = fields[0]
			s.Sub <- SessionChangeMsg{ActiveSession: h.ActiveSession()}

		} else {
			s.Output("Invalid session.\n")
		}
	} else if len(fields) == 2 {
		h.AddSession(fields[0], fields[1])
		h.Active = fields[0]
		s.Sub <- SessionChangeMsg{ActiveSession: h.ActiveSession()}
	} else {
		h.ActiveSession().Output("Usage: #session <name> <address:port>" + "\n")
	}

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

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, strings.ToLower(parsed[0])) {
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

func (s *Session) ParseCommand(cmd string) {
	// Note: Command has already been added to Content (colored) in HandleInput()
	// and an UpdateMessage was already sent, so we don't need to send another one

	// TODO: We'll want to check this for aliases and/or variables
	if s.Connected {
		s.Socket.Write([]byte(cmd + "\n"))
	}

	// No need to send UpdateMessage here - Output() already sent one with the colored command
}
