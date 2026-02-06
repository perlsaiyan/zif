package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	"github.com/perlsaiyan/zif/session"
	lua "github.com/yuin/gopher-lua"
)

var Info session.PluginInfo = session.PluginInfo{Name: "Kallisti", Version: "0.1a"}

type KallistiData struct {
	CurrentRoomRingLogID int
	LastPrompt           int
	LastLine             int64 // Unix timestamp in nanoseconds of last MUD line received
	Atlas                *sqlx.DB
	World                map[string]AtlasRoomRecord
	Travel               KallistiTravel
	CurrentRoom          CurrentRoom
	Triggers             []KallistiTrigger
}

type KallistiTrigger struct {
	Name    string
	Pattern *regexp.Regexp
	Fn      func(*session.Session, []string)
}

type CurrentRoom struct {
	Vnum        string
	Title       string
	Description string
	Exits       []string
	Objects     []RoomEntity
	Mobs        []RoomEntity
}

type RoomEntity struct {
	Name     string
	Quantity int
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
		Triggers:   make([]KallistiTrigger, 0),
	}
	d := s.Data["kallisti"].(*KallistiData)

	// Connect to our world.db
	d.Atlas = ConnectAtlasDB(s.Name)
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

	// Register Kallisti Triggers

	// Crafting: "You craft [Output] made from [Input]."
	AddKallistiTrigger(s, "Crafting", `^You craft (.+) made from (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 3 {
			return
		}
		// matches[1] = Output, matches[2] = Input
		evt := NewKallistiCraftEvent("bone", "weapon", matches[2], matches[1])
		s.FireEvent("kallisti.craft", evt)
	})

	// Carving: "You carve [Input] into [Output]."
	AddKallistiTrigger(s, "Carving", `^You carve (.+) into (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 3 {
			return
		}
		// matches[1] = Input, matches[2] = Output
		evt := NewKallistiCraftEvent("bone", "bone", matches[1], matches[2])
		s.FireEvent("kallisti.craft", evt)
	})

	// Brewing: "You brew [Output]."
	AddKallistiTrigger(s, "Brewing", `^You brew (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 2 {
			return
		}
		// matches[1] = Output
		evt := NewKallistiCraftEvent("herb", "potion", "herbs", matches[1])
		s.FireEvent("kallisti.craft", evt)
	})

	// Death: "[Name] is dead! R.I.P."
	AddKallistiTrigger(s, "Death", `^(.+?)(?: \(your follower\))? is dead!  R.I.P\.`, func(s *session.Session, matches []string) {
		if len(matches) < 2 {
			return
		}
		evt := NewKallistiDeathEvent(matches[1])
		s.FireEvent("kallisti.death", evt)
	})
}

func AddKallistiTrigger(s *session.Session, name string, pattern string, fn func(*session.Session, []string)) {
	if d, ok := s.Data["kallisti"].(*KallistiData); ok {
		re, err := regexp.Compile(pattern)
		if err != nil {
			s.Output("Error compiling trigger " + name + ": " + err.Error() + "\n")
			return
		}
		d.Triggers = append(d.Triggers, KallistiTrigger{
			Name:    name,
			Pattern: re,
			Fn:      fn,
		})
	}
}

func ProcessKallistiTriggers(s *session.Session, line string, stripped string) {
	if d, ok := s.Data["kallisti"].(*KallistiData); ok {
		// Strip trailing whitespace from stripped line for better matching
		clean := strings.TrimRight(stripped, "\r\n")
		for _, t := range d.Triggers {
			if matches := t.Pattern.FindStringSubmatch(clean); matches != nil {
				t.Fn(s, matches)
			}
		}
	}
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

	// Add current_room() function
	L.SetField(kallistiTable, "current_room", L.NewFunction(func(L *lua.LState) int {
		if d, ok := s.Data["kallisti"].(*KallistiData); ok {
			roomTable := L.NewTable()
			L.SetField(roomTable, "vnum", lua.LString(d.CurrentRoom.Vnum))
			L.SetField(roomTable, "title", lua.LString(d.CurrentRoom.Title))
			L.SetField(roomTable, "description", lua.LString(d.CurrentRoom.Description))

			exitsTable := L.NewTable()
			for i, ex := range d.CurrentRoom.Exits {
				exitsTable.Insert(i+1, lua.LString(ex))
			}
			L.SetField(roomTable, "exits", exitsTable)

			objectsTable := L.NewTable()
			for i, obj := range d.CurrentRoom.Objects {
				objTable := L.NewTable()
				L.SetField(objTable, "name", lua.LString(obj.Name))
				L.SetField(objTable, "quantity", lua.LNumber(obj.Quantity))
				objectsTable.Insert(i+1, objTable)
			}
			L.SetField(roomTable, "objects", objectsTable)

			mobsTable := L.NewTable()
			for i, mob := range d.CurrentRoom.Mobs {
				mobTable := L.NewTable()
				L.SetField(mobTable, "name", lua.LString(mob.Name))
				L.SetField(mobTable, "quantity", lua.LNumber(mob.Quantity))
				mobsTable.Insert(i+1, mobTable)
			}
			L.SetField(roomTable, "mobs", mobsTable)

			L.Push(roomTable)
			return 1
		}
		L.Push(lua.LNil)
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
func kallistiLineHook(s *session.Session, line string, stripped string) {
	if s == nil || s.Data == nil {
		return
	}
	if d, ok := s.Data["kallisti"].(*KallistiData); ok && d != nil {
		// Update last line timestamp
		d.LastLine = time.Now().UnixNano()
		// Update context injector to refresh Lua state
		s.UpdateContextInjector("kallisti")

		// Process triggers
		ProcessKallistiTriggers(s, line, stripped)
	}
}

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
