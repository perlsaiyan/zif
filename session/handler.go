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
	Plugins  *PluginRegistry
	Sub      chan tea.Msg
}

type Session struct {
	Name         string
	Handler      *SessionHandler
	Birth        time.Time
	Context      context.Context
	Cancel       context.CancelFunc
	Content      string
	Ringlog      RingLog
	Address      string
	Socket       net.Conn
	MSDP         *kallisti.MSDPHandler
	TTCount      int
	PasswordMode bool
	Connected    bool
	Sub          chan tea.Msg
	Tickers      *TickerRegistry
	Actions      *ActionRegistry
	Events       *EventRegistry
	Queue        *QueueRegistry
	Data         map[string]interface{}
}

// HandleInput processes the input command.
func (s *Session) HandleInput(cmd string) {
	if cmd == "" {
		if s.Connected {
			s.Socket.Write([]byte("\n"))
		}
		return
	}

	if cmd[0] == '#' {
		s.ParseInternalCommand(cmd)
	} else {
		s.ParseCommand(cmd + "\n")
	}
}

// ActiveSession returns the currently active session.
func (s *SessionHandler) ActiveSession() *Session {
	return s.Sessions[s.Active]
}

// NewHandler creates and initializes a new SessionHandler.
func NewHandler() SessionHandler {
	sub := make(chan tea.Msg, 50)
	s := Session{
		Name:    "zif",
		Content: Motd(),
		MSDP:    kallisti.NewMSDP(),
		Sub:     sub,
		Birth:   time.Now(),
	}
	sh := SessionHandler{
		Active:   "zif",
		Sessions: make(map[string]*Session),
		Plugins:  NewPluginRegistry(),
		Sub:      sub,
	}
	s.Handler = &sh
	sh.Sessions["zif"] = &s
	return sh
}

// AddSession adds a new session to the session handler.
func (s *SessionHandler) AddSession(name, address string) {
	newSession := &Session{
		Name:  name,
		Birth: time.Now(),
		MSDP:  kallisti.NewMSDP(),
		Sub:   s.Sub,

		Actions: NewActionRegistry(),
		Events:  NewEventRegistry(),
		Queue:   NewQueueRegistry(),

		Ringlog: NewRingLog(),
		Handler: s,

		Data: make(map[string]interface{}),
	}

	// Initialize the priority Queue

	s.Sessions[name] = newSession
	ctx := context.Background()
	newSession.Context, newSession.Cancel = context.WithCancel(ctx)

	if address == "" {
		s.ActiveSession().Output("created nil session: " + name + "\n")
		return
	}

	s.ActiveSession().Output("attempt to connect to: " + address + "\n")
	s.Active = name
	s.Sub <- SessionChangeMsg{ActiveSession: s.ActiveSession()}

	var err error
	newSession.Address = address
	newSession.Socket, err = net.Dial("tcp", address)
	if err != nil {
		log.Printf("Error connecting to %s: %v", address, err)
		delete(s.Sessions, name)
		return
	}

	newSession.Connected = true
	NewTickerRegistry(newSession.Context, newSession)

	for _, v := range s.Plugins.Plugins {
		log.Printf("Activating plugin: %s", v.Name)
		newSession.Output("Activating plugin: " + v.Name + "\n")
		f, err := v.Plugin.Lookup("RegisterSession")
		if err != nil {
			log.Printf("RegisterSession() lookup failure on plugin %s", v.Name)
			continue
		}
		f.(func(*Session))(newSession)
	}

	go newSession.mudReader()
}

// Motd returns the message of the day.
func Motd() string {
	return "\n\n\x1b[38;2;165;80;223m" +
		" ░▒▓████████▓▒░▒▓█▓▒░▒▓████████▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "         ▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "      ░▒▓██▓▒░░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "    ░▒▓██▓▒░  ░▒▓█▓▒░▒▓██████▓▒░\n" +
		"\x1b[38;2;165;80;223m" + "  ░▒▓██▓▒░    ░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + " ░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░\n" +
		"\x1b[38;2;165;80;223m" + " ░▒▓████████▓▒░▒▓█▓▒░▒▓█▓▒░\n\n" +
		" Zero Insertion Force Mud Client\n\n"
}

// PluginMOTD returns the message of the day for plugins.
func (h *SessionHandler) PluginMOTD() string {
	var msg string
	for _, p := range h.Plugins.Plugins {
		f, err := p.Plugin.Lookup("MOTD")
		if err != nil {
			log.Printf("MOTD() lookup failure on plugin %s", p.Name)
			continue
		}
		msg += f.(func() string)()
	}
	return msg
}
