package session

import (
	"encoding/csv"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type CommandFunction func(*Session, string, *SessionHandler)
type Command struct {
	Name string
	Fn   CommandFunction
}

var internalCommands = []Command{
	{Name: "actions", Fn: CmdActions},
	{Name: "cancel", Fn: CmdCancelTicker},
	{Name: "events", Fn: CmdEvents},
	{"help", CmdHelp},
	{Name: "msdp", Fn: CmdMSDP},
	{Name: "plugins", Fn: CmdPlugins},
	{Name: "session", Fn: CmdSession},
	{Name: "sessions", Fn: CmdSessions},
	{Name: "test", Fn: CmdTestTicker},
	{Name: "tickers", Fn: CmdTickers},
}

var internalCommandHelp = map[string]string{
	"cancel":   "Cancel test for timers",
	"help":     "This help command",
	"msdp":     "Show MSDP values",
	"session":  "Usage: #session <name> <host:port>",
	"sessions": "Show current sessions",
	"test":     "Just a test command/playground",
	"tickers":  "Show tickers",
}

func CmdMSDP(s *Session, cmd string, h *SessionHandler) {
	buf := fmt.Sprintf("PC in Room: %v, PC in Zone: %v\nRoom: %s\n", s.MSDP.PCInRoom, s.MSDP.PCInZone, s.MSDP.RoomName)
	s.Output(buf)
}

func CmdTest(s *Session, cmd string, h *SessionHandler) {
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

func CmdSession(s *Session, cmd string, h *SessionHandler) {
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
func CmdHelp(s *Session, cmd string, h *SessionHandler) {
	msg := "Commands:\n"
	for k, v := range internalCommandHelp {
		msg += fmt.Sprintf("%15s - %s\n", k, v)
	}
	s.Output(msg)
}

func makeRow(name string, address string, start time.Time) table.Row {

	return table.NewRow(table.RowData{
		"name":    name,
		"address": address,
		"time":    time.Since(start).Round(time.Second),
	})
}

func CmdSessions(s *Session, cmd string, h *SessionHandler) {
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

func (h *SessionHandler) ParseInternalCommand(cmd string) {
	h.ActiveSession().Content += cmd + "\n"
	parsed := strings.Fields(cmd[1:])
	args := strings.SplitN(cmd, " ", 2)

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, strings.ToLower(parsed[0])) {
			if len(args) < 2 {
				internalCommands[lookup].Fn(h.ActiveSession(), "", h)
				h.Sub <- UpdateMessage{Session: h.ActiveSession().Name}
				return
			} else {
				internalCommands[lookup].Fn(h.ActiveSession(), args[1], h)
				h.Sub <- UpdateMessage{Session: h.ActiveSession().Name}
				return
			}
		}
	}
	h.Sub <- UpdateMessage{Session: h.ActiveSession().Name}
}

// TODO: Probably should have this as a Session method, not SessionHandler
func (m *SessionHandler) ParseCommand(cmd string) {
	if !m.ActiveSession().PasswordMode {
		m.ActiveSession().Content += cmd + "\n"
	}

	// TODO: We'll want to check this for aliases and/or variables
	if m.ActiveSession().Connected {
		m.ActiveSession().Socket.Write([]byte(cmd + "\n"))
	}

	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
}
