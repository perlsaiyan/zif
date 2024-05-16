package session

import (
	"context"
	"log"
	"net"
	"time"

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

	Birth        time.Time
	Context      context.Context
	Cancel       context.CancelFunc
	Content      string
	Address      string
	Socket       net.Conn
	MSDP         *kallisti.MSDPHandler
	TTCount      int
	PasswordMode bool
	Connected    bool

	Sub chan tea.Msg

	// Various Registries
	Tickers *TickerRegistry
	Plugins *PluginRegistry
	Actions *ActionRegistry
	Events  *EventRegistry
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
	sub := make(chan tea.Msg, 50)
	s := Session{
		Name:    "zif",
		Content: Motd(),
		MSDP:    kallisti.NewMSDP(),
		Socket:  nil,
		Sub:     sub,
		Birth:   time.Now(),
	}
	sh := SessionHandler{
		Active:   "zif",
		Sessions: make(map[string]*Session),
		Sub:      sub,
	}
	sh.Sessions["zif"] = &s
	//sh.Sub <- UpdateMessage{Session: "zif"}
	return sh
}

func (s *SessionHandler) AddSession(name string, address string) {
	new := Session{
		Name:    name,
		Birth:   time.Now(),
		MSDP:    kallisti.NewMSDP(),
		Sub:     s.Sub,
		Actions: NewActionRegistry(),
		Events:  NewEventRegistry(),
	}

	s.Sessions[name] = &new
	ctx := context.Background()
	new.Context, new.Cancel = context.WithCancel(ctx)

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
		NewTickerRegistry(new.Context, s.Sessions[name])

		//spawn reader, ticker, etc
		go s.Sessions[name].mudReader(s.Sub)

	} else {
		s.ActiveSession().Output("created nil session: " + name + "\n")
	}
}

func Motd() string {
	return "\n\n\x1b[38;2;165;80;223m" + " ░▒▓████████▓▒░▒▓█▓▒░▒▓████████▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "         ▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "      ░▒▓██▓▒░░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "    ░▒▓██▓▒░  ░▒▓█▓▒░▒▓██████▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "  ░▒▓██▓▒░    ░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + " ░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + " ░▒▓████████▓▒░▒▓█▓▒░▒▓█▓▒░\n\n" +
		" Zero Insertion Force Mud Client\n\n"
}
