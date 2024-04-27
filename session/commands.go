package session

import (
	"fmt"
	"strings"
)

var internalCommands = []Command{
	{Name: "help", Fn: CmdHelp},
}

type Command struct {
	Name string
	Fn   CommandFunction
}
type CommandFunction func(*SessionHandler, string)

func CmdHelp(s *SessionHandler, cmd string) {
	s.ActiveSession().Content += fmt.Sprintf("Here's your help: %s\n", cmd)
}

func (m *SessionHandler) ParseInternalCommand(cmd string) {
	m.ActiveSession().Content += cmd + "\n"
	parsed := strings.Fields(cmd[1:])
	args := strings.SplitN(cmd, " ", 2)

	for lookup := range internalCommands {
		if strings.HasPrefix(internalCommands[lookup].Name, strings.ToLower(parsed[0])) {
			internalCommands[lookup].Fn(m, args[1])
		}
	}
	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
}

func (m *SessionHandler) ParseCommand(cmd string) {
	m.ActiveSession().Content += cmd
	m.Sub <- UpdateMessage{Session: m.ActiveSession().Name}
}
