package session

import (
	"log"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	kallisti "github.com/perlsaiyan/zif/protocol"
)

type SessionHandler struct {
	Active   string
	Sessions map[string]*Session
	Sub      chan tea.Msg
}

type Session struct {
	Name string

	Content      string
	Address      string
	Socket       net.Conn
	MSDP         *kallisti.MSDPHandler
	TTCount      int
	PasswordMode bool
	Connected    bool
}

func (s *SessionHandler) HandleInput(cmd string) {
	if len(cmd) > 0 {
		if cmd[0] == '#' {
			s.ParseInternalCommand(cmd)

		} else {
			s.ParseCommand(cmd + "\n")
		}
	} else {
		// Just press enter
		if s.ActiveSession().Connected {
			s.ActiveSession().Socket.Write([]byte("\n"))
		}
	}
}

func (s SessionHandler) ActiveSession() *Session {
	return s.Sessions[s.Active]
}

func NewHandler() SessionHandler {
	s := Session{
		Name:    "zif",
		Content: "",
		MSDP:    kallisti.NewMSDP(),
		Socket:  nil,
	}
	sh := SessionHandler{
		Active:   "zif",
		Sessions: make(map[string]*Session),
		Sub:      make(chan tea.Msg, 50),
	}
	sh.Sessions["zif"] = &s
	return sh
}

func (s *SessionHandler) AddSession(name string, address string) {
	new := Session{
		Name: name,
		MSDP: kallisti.NewMSDP(),
	}

	s.Sessions[name] = &new

	if len(address) > 1 {
		s.ActiveSession().Output("attempt to connect to: " + address + "\n")
		var err error
		s.Sessions[name].Address = address
		s.Sessions[name].Socket, err = net.Dial("tcp", address)
		if err != nil {
			log.Printf("Error: %v\n", err)
			delete(s.Sessions, name)
			return
		}

		s.Sessions[name].Connected = true
		//spawn reader
		go s.Sessions[name].mudReader(s.Sub)

	} else {
		s.ActiveSession().Output("created nil session: " + name + "\n")
	}
}
