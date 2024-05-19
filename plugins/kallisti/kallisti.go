package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/perlsaiyan/zif/session"
)

var Info session.PluginInfo = session.PluginInfo{Name: "Kallisti", Version: "0.1a"}

type KallistiData struct {
	CurrentRoomRingLogID int
	LastPrompt           int
}

// RegisterSession is called when a Session activates the plugin
func RegisterSession(s *session.Session) {
	s.Output("Kallisti plugin loaded\n")
	s.Data["kallisti"] = &KallistiData{CurrentRoomRingLogID: -1, LastPrompt: -1}

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
}

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
