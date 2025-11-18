package main

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	lua "github.com/yuin/gopher-lua"
	"github.com/perlsaiyan/zif/session"
)

var Info session.PluginInfo = session.PluginInfo{Name: "Kallisti", Version: "0.1a"}

type KallistiData struct {
	CurrentRoomRingLogID int
	LastPrompt           int
	LastLine             int64 // Unix timestamp in nanoseconds of last MUD line received
	Atlas                *sqlx.DB
	World                map[string]AtlasRoomRecord
	Travel               KallistiTravel
}

type KallistiTravel struct {
	On       bool
	To       string
	Distance int
	Length   int
}

// RegisterSession is called when a Session activates the plugin
func RegisterSession(s *session.Session) {
	if s == nil {
		return
	}
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Output("Kallisti plugin loaded\n")
	s.Data["kallisti"] = &KallistiData{CurrentRoomRingLogID: -1,
		LastPrompt: -1,
		LastLine:   0,
		World:      make(map[string]AtlasRoomRecord),
	}
	d := s.Data["kallisti"].(*KallistiData)

	// Connect to our world.db
	d.Atlas = ConnectAtlasDB()
	LoadAllRooms(s)
	LoadAllExits(s)

	// Register context injector
	s.RegisterContextInjector("kallisti", kallistiContextInjector)

	// Register MSDP update hook
	s.RegisterMSDPUpdateHook("kallisti", kallistiMSDPHook)

	// Register MUD line hook
	s.RegisterMUDLineHook("kallisti", kallistiLineHook)

	// Events
	s.AddEvent("core.prompt", session.Event{Name: "RoomScanner", Enabled: true, Fn: ParseRoom})

	// Actions
	s.AddAction(session.Action{
		Name:    "RoomScanner",
		Pattern: "\x1b\\[1;35m",
		Color:   true,
		Enabled: true,
		Fn:      PossibleRoomScanner,
	})

	// Tickers
	s.AddTicker(&session.TickerRecord{
		Name:       "Autoheal",
		Interval:   2000,
		Fn:         Autoheal,
		Iterations: 0,
	})
	s.AddTicker(&session.TickerRecord{
		Name:       "Autobuf",
		Interval:   2000,
		Fn:         Autoheal,
		Iterations: 0,
	})

	// Commands

	s.AddCommand(session.Command{Name: "room", Fn: CmdRoom}, "Show room information")
	s.AddCommand(session.Command{Name: "path", Fn: CmdBFSRoomToRoom}, "Find a path between two rooms")
	s.AddCommand(session.Command{Name: "map", Fn: CmdShowMap}, "Show a map")
}

// kallistiContextInjector injects the kallisti global table into Lua
func kallistiContextInjector(s *session.Session, L *lua.LState) error {
	// Check if kallisti data exists
	if _, ok := s.Data["kallisti"]; !ok {
		return nil // No kallisti data, skip injection
	}

	// Create kallisti global table
	kallistiTable := L.NewTable()

	// Add function_test() function
	L.SetField(kallistiTable, "function_test", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("Kallisti context injection is working!"))
		return 1
	}))

	// Add last_line value (updated when injector is refreshed)
	var lastLineValue lua.LNumber
	if d, ok := s.Data["kallisti"].(*KallistiData); ok {
		lastLineValue = lua.LNumber(float64(d.LastLine))
	} else {
		lastLineValue = lua.LNumber(0)
	}
	L.SetField(kallistiTable, "last_line", lastLineValue)

	// Add current_room() function (placeholder for now)
	L.SetField(kallistiTable, "current_room", L.NewFunction(func(L *lua.LState) int {
		// TODO: Return actual room table
		roomTable := L.NewTable()
		L.Push(roomTable)
		return 1
	}))

	// Add group table
	groupTable := L.NewTable()
	
	// Add group.healers_present() function
	L.SetField(groupTable, "healers_present", L.NewFunction(func(L *lua.LState) int {
		// TODO: Process GROUP MSDP and return healers
		healersTable := L.NewTable()
		L.Push(healersTable)
		return 1
	}))

	L.SetField(kallistiTable, "group", groupTable)

	// Set as global
	L.SetGlobal("kallisti", kallistiTable)

	return nil
}

// kallistiMSDPHook is called when MSDP data is updated
func kallistiMSDPHook(s *session.Session, msdpData map[string]interface{}) {
	if s == nil || msdpData == nil {
		return
	}
	// Check if GROUP or ROOM_VNUM changed
	needsUpdate := false
	if _, ok := msdpData["GROUP"]; ok {
		needsUpdate = true
	}
	if _, ok := msdpData["ROOM_VNUM"]; ok {
		needsUpdate = true
	}

	if needsUpdate {
		// Update context injector to refresh Lua state
		s.UpdateContextInjector("kallisti")
	}
}

// kallistiLineHook is called when a MUD line is processed
func kallistiLineHook(s *session.Session, line string) {
	if s == nil || s.Data == nil {
		return
	}
	if d, ok := s.Data["kallisti"].(*KallistiData); ok && d != nil {
		// Update last line timestamp
		d.LastLine = time.Now().UnixNano()
		// Update context injector to refresh Lua state
		s.UpdateContextInjector("kallisti")
	}
}

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
