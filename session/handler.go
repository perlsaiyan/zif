package session

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/perlsaiyan/zif/config"
	kallisti "github.com/perlsaiyan/zif/protocol"
	lua "github.com/yuin/gopher-lua"
)

// LineTerminator is the RFC 854 (Telnet) standard line terminator: CR LF
const LineTerminator = "\r\n"

// ContextInjector is a function type that injects context into a Lua state
type ContextInjector func(*Session, *lua.LState) error

// MSDPUpdateHook is a function type called when MSDP data is updated
type MSDPUpdateHook func(*Session, map[string]interface{})

// MUDLineHook is a function type called when a MUD line is processed
type MUDLineHook func(*Session, string, string)

type SessionHandler struct {
	Active             string
	Sessions           map[string]*Session
	Plugins            *PluginRegistry
	Sub                chan tea.Msg
	PendingSessionData map[string]interface{} // Pre-populated data for the next AddSession call
}

type Session struct {
	Name           string
	Handler        *SessionHandler
	Birth          time.Time
	Context        context.Context
	Cancel         context.CancelFunc
	Content        string
	Ringlog        RingLog
	Address        string
	Socket         net.Conn
	MSDP           *kallisti.MSDPHandler
	TTCount        int
	PasswordMode   bool
	Connected      bool
	Sub            chan tea.Msg
	Tickers        *TickerRegistry
	Actions        *ActionRegistry
	Aliases        *AliasRegistry
	Events         *EventRegistry
	Queue          *QueueRegistry
	Data           map[string]interface{}
	LuaState       *lua.LState
	Modules        *ModuleRegistry
	EchoNegotiated bool // Infinite loop protection: track if we've responded to ECHO negotiation
	LoginComplete  bool // Track if we've completed login (entered the game)

	// Context injection system
	contextInjectors map[string]ContextInjector
	msdpUpdateHooks  map[string]MSDPUpdateHook
	mudLineHooks     map[string]MUDLineHook
}

// HandleInput processes the input command.
func (s *Session) HandleInput(cmd string) {
	if cmd == "" {
		if s.Connected {
			s.Socket.Write([]byte(LineTerminator))
		}
		return
	}

	// Output the command in bright white (unless in password mode)
	if !s.PasswordMode {
		// Try multiple ANSI formats for maximum compatibility
		// Format 1: RGB (most compatible with modern terminals)
		// Format 2: Bright white (8-bit color)
		// Using both: set bright white, then RGB as fallback
		coloredCmd := "\x1b[1;37m" + cmd + "\x1b[0m\n" // Bright white: \x1b[1;37m
		s.Output(coloredCmd)
	}

	// Check for aliases first (before internal commands)
	if s.Aliases != nil && s.MatchAlias(cmd) {
		return // Alias handled the command
	}

	if cmd[0] == '#' {
		s.ParseInternalCommand(cmd)
	} else {
		s.ParseCommand(cmd)
	}
}

// ActiveSession returns the currently active session.
// May return nil if the active session name doesn't exist in the Sessions map.
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

	// Initialize context for the default session
	ctx := context.Background()
	s.Context, s.Cancel = context.WithCancel(ctx)

	// Initialize Lua state and registries for the default session
	s.LuaState = lua.NewState()
	s.Actions = NewActionRegistry()
	s.Aliases = NewAliasRegistry()
	s.Events = NewEventRegistry()
	s.Queue = NewQueueRegistry()
	s.Ringlog = NewRingLog()
	s.Modules = NewModuleRegistry()
	s.Data = make(map[string]interface{})
	s.contextInjectors = make(map[string]ContextInjector)
	s.msdpUpdateHooks = make(map[string]MSDPUpdateHook)
	s.mudLineHooks = make(map[string]MUDLineHook)

	// Initialize ticker registry (requires context)
	NewTickerRegistry(s.Context, &s)

	// Register Lua API
	s.RegisterLuaAPI()

	// Ensure config directories exist
	if err := config.EnsureConfigDirs(); err != nil {
		log.Printf("Warning: failed to ensure config directories: %v", err)
	}

	// Load global modules first
	if err := LoadGlobalModules(&s); err != nil {
		log.Printf("Warning: failed to load global modules: %v", err)
	}

	// Load session-specific modules for default session
	if err := LoadSessionModules(&s, "zif"); err != nil {
		log.Printf("Warning: failed to load session modules: %v", err)
	}

	return sh
}

// AddSession adds a new session to the session handler.
// Returns an error if the session could not be created or connected.
func (s *SessionHandler) AddSession(name, address string) error {
	// Validate session name - no spaces, must be non-empty
	if name == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if strings.Contains(name, " ") {
		return fmt.Errorf("session name cannot contain spaces")
	}

	// Validate address format - should contain : for host:port
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if !strings.Contains(address, ":") {
		return fmt.Errorf("invalid address format: expected host:port")
	}

	newSession := &Session{
		Name:  name,
		Birth: time.Now(),
		MSDP:  kallisti.NewMSDP(),
		Sub:   s.Sub,

		Actions: NewActionRegistry(),
		Aliases: NewAliasRegistry(),
		Events:  NewEventRegistry(),
		Queue:   NewQueueRegistry(),

		Ringlog: NewRingLog(),
		Handler: s,

		Data:     make(map[string]interface{}),
		LuaState: lua.NewState(),
		Modules:  NewModuleRegistry(),

		contextInjectors: make(map[string]ContextInjector),
		msdpUpdateHooks:  make(map[string]MSDPUpdateHook),
		mudLineHooks:     make(map[string]MUDLineHook),
	}

	// Initialize the priority Queue

	s.Sessions[name] = newSession
	ctx := context.Background()
	newSession.Context, newSession.Cancel = context.WithCancel(ctx)

	// Output to current active session if it exists
	if activeSess := s.ActiveSession(); activeSess != nil {
		activeSess.Output("attempt to connect to: " + address + "\n")
	}

	var err error
	newSession.Address = address
	newSession.Socket, err = net.Dial("tcp", address)
	if err != nil {
		log.Printf("Error connecting to %s: %v", address, err)
		delete(s.Sessions, name)
		// Output error to current active session if it exists
		if activeSess := s.ActiveSession(); activeSess != nil {
			activeSess.Output(fmt.Sprintf("Failed to connect to %s: %v\n", address, err))
		}
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	newSession.Connected = true
	NewTickerRegistry(newSession.Context, newSession)

	// Register Lua API
	newSession.RegisterLuaAPI()

	// Ensure config directories exist
	if err := config.EnsureConfigDirs(); err != nil {
		log.Printf("Warning: failed to ensure config directories: %v", err)
	}

	// Load global modules first
	if err := LoadGlobalModules(newSession); err != nil {
		log.Printf("Warning: failed to load global modules: %v", err)
	}

	// Load session-specific modules
	if err := LoadSessionModules(newSession, name); err != nil {
		log.Printf("Warning: failed to load session modules: %v", err)
	}

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

	// Inject context after plugins have registered their injectors
	if err := newSession.InjectContext(); err != nil {
		log.Printf("Warning: failed to inject context: %v", err)
	}

	// Merge any pending session data (e.g. credentials from sessions.yaml)
	// before mudReader starts so triggers can access them immediately
	if s.PendingSessionData != nil {
		for k, v := range s.PendingSessionData {
			newSession.Data[k] = v
		}
	}

	go newSession.mudReader()
	return nil
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

// OnMSDPUpdate calls all registered MSDP update hooks with the updated MSDP data
func (s *Session) OnMSDPUpdate(msdpData map[string]interface{}) {
	if s == nil || s.msdpUpdateHooks == nil {
		return
	}
	for _, hook := range s.msdpUpdateHooks {
		if hook != nil {
			hook(s, msdpData)
		}
	}
}

// OnMUDLine calls all registered MUD line hooks with the line content
func (s *Session) OnMUDLine(line string, stripped string) {
	if s == nil || s.mudLineHooks == nil {
		return
	}
	for _, hook := range s.mudLineHooks {
		if hook != nil {
			hook(s, line, stripped)
		}
	}
}
