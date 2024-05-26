package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	"github.com/perlsaiyan/zif/session"
)

var Info session.PluginInfo = session.PluginInfo{Name: "Kallisti", Version: "0.1a"}

type KallistiData struct {
	CurrentRoomRingLogID int
	LastPrompt           int
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
	s.Output("Kallisti plugin loaded\n")
	s.Data["kallisti"] = &KallistiData{CurrentRoomRingLogID: -1,
		LastPrompt: -1,
		World:      make(map[string]AtlasRoomRecord),
	}
	d := s.Data["kallisti"].(*KallistiData)

	// Connect to our world.db
	d.Atlas = ConnectAtlasDB()
	LoadAllRooms(s)
	LoadAllExits(s)

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

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
