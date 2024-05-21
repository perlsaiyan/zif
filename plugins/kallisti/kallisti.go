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
}

// RegisterSession is called when a Session activates the plugin
func RegisterSession(s *session.Session) {
	s.Output("Kallisti plugin loaded\n")
	s.Data["kallisti"] = &KallistiData{CurrentRoomRingLogID: -1, LastPrompt: -1}
	d := s.Data["kallisti"].(*KallistiData)

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

	// Connect to our world.db
	d.Atlas = ConnectAtlasDB()

	// some junk
	q := &session.QueueItem{Priority: 1, Name: "prio1", Command: "prio1"}
	s.Queue.Add(q)

	q = &session.QueueItem{Priority: 6, Name: "prio6", Command: "prio6"}
	s.Queue.Add(q)

	q = &session.QueueItem{Priority: 3, Name: "prio3", Command: "prio3"}
	s.Queue.Add(q)

	q = &session.QueueItem{Priority: 4, Name: "prio4", Command: "prio4"}
	s.Queue.Add(q)

	q = &session.QueueItem{Priority: 2, Name: "prio2", Command: "prio2"}
	s.Queue.Add(q)

}

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
