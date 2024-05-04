package session

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type CommandFunction func(*SessionHandler, string)
type Command struct {
	Name string
	Fn   CommandFunction
}

var internalCommands = []Command{
	{Name: "cancel", Fn: CmdCancelTicker},
	{Name: "help", Fn: CmdHelp},
	{Name: "msdp", Fn: CmdMSDP},
	{Name: "session", Fn: CmdSession},
	{Name: "sessions", Fn: CmdSessions},
}

func CmdMSDP(s *SessionHandler, cmd string) {
	buf := fmt.Sprintf("PC in Room: %v, PC in Zone: %v\nRoom: %s\n", s.ActiveSession().MSDP.PCInRoom, s.ActiveSession().MSDP.PCInZone, s.ActiveSession().MSDP.RoomName)
	s.ActiveSession().Output(buf)
}

func CmdSession(s *SessionHandler, cmd string) {
	fields := strings.Fields(cmd)
	if len(fields) < 1 {
		s.ActiveSession().Output("Usage: #session <name> <address:port>" + "\n")
		return
	} else if len(fields) == 1 {
		_, ok := s.Sessions[fields[0]]
		if ok {
			s.Active = fields[0]
			s.Sub <- SessionChangeMsg{ActiveSession: s.ActiveSession()}

		} else {
			s.ActiveSession().Output("Invalid session.\n")
		}
	} else if len(fields) == 2 {
		s.AddSession(fields[0], fields[1])
		s.Active = fields[0]
		s.Sub <- SessionChangeMsg{ActiveSession: s.ActiveSession()}
	} else {
		s.ActiveSession().Output("Usage: #session <name> <address:port>" + "\n")
	}

}
func CmdHelp(s *SessionHandler, cmd string) {
	s.ActiveSession().Content += fmt.Sprintf("Here's your help: %s\n", cmd)
}

func makeRow(name string, address string) table.Row {
	return table.NewRow(table.RowData{
		"name":    name,
		"address": address,
		"time":    "00:00:00",
	})
}

func CmdSessions(s *SessionHandler, cmd string) {
	var rows []table.Row
	for i := range s.Sessions {
		if s.Sessions[i].Name == s.ActiveSession().Name {
			rows = append(rows, makeRow("> "+s.Sessions[i].Name, s.Sessions[i].Address))
		} else {
			rows = append(rows, makeRow(s.Sessions[i].Name, s.Sessions[i].Address))
		}
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("address", "Address", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("time", "Uptime", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.ActiveSession().Output(t.View() + "\n")
}

func (m *SessionHandler) ParseInternalCommand(cmd string) {
	m.ActiveSession().Content += cmd + "\n"
	parsed := strings.Fields(cmd[1:])
	args := strings.SplitN(cmd, " ", 2)

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, strings.ToLower(parsed[0])) {
			if len(args) < 2 {
				internalCommands[lookup].Fn(m, "")
				m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
				return
			} else {
				internalCommands[lookup].Fn(m, args[1])
				m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
				return
			}
		}
	}
	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
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
