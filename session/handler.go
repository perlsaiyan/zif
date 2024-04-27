package session

import (
	"net"

	tea "github.com/charmbracelet/bubbletea"
	kallisti "github.com/perlsaiyan/zif/protocol"
)

// The active session has changed
type SessionChangeMsg struct {
	ActiveSession *Session
}

type SessionHandler struct {
	Active   string
	Sessions map[string]*Session
	Sub      chan tea.Msg
}

type Session struct {
	Name string

	Content      string
	Socket       net.Conn
	MSDP         *kallisti.MSDPHandler
	TTCount      int
	PasswordMode bool
}

func (s *SessionHandler) HandleInput(cmd string) {
	if len(cmd) > 0 {
		if cmd[0] == '#' {
			s.ParseInternalCommand(cmd)

		} else {
			s.ParseCommand(cmd + "\n")
		}
	} else {
		//m.socket.Write([]byte("\n"))
	}
}

func (s SessionHandler) ActiveSession() *Session {
	return s.Sessions[s.Active]
}

func NewHandler() SessionHandler {
	s := Session{
		Name:    "Zif",
		Content: "",
		MSDP:    &kallisti.MSDPHandler{},
		Socket:  nil,
	}
	sh := SessionHandler{
		Active:   "Zif",
		Sessions: make(map[string]*Session),
		Sub:      make(chan tea.Msg),
	}
	sh.Sessions["Zif"] = &s
	return sh
}

func (s *SessionHandler) AddSession(name string) {
	new := Session{
		Name: name,
	}

	s.Sessions[name] = &new
}
