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
	{Name: "help", Fn: CmdHelp},
	{Name: "sessions", Fn: CmdSessions},
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
		rows = append(rows, makeRow(s.Sessions[i].Name, s.Sessions[i].Address))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("address", "Address", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("time", "Uptime", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.ActiveSession().Content += t.View() + "\n"
}

func (m *SessionHandler) ParseInternalCommand(cmd string) {
	m.ActiveSession().Content += cmd + "\n"
	parsed := strings.Fields(cmd[1:])
	args := strings.SplitN(cmd, " ", 2)

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, strings.ToLower(parsed[0])) {
			if len(args) < 2 {
				internalCommands[lookup].Fn(m, "")
			} else {
				internalCommands[lookup].Fn(m, args[1])
			}
		}
	}
	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
}

func (m *SessionHandler) ParseCommand(cmd string) {
	m.ActiveSession().Content += cmd
	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
}
