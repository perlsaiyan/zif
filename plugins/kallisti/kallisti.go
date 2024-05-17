package main

import (
	"log"

	"github.com/charmbracelet/lipgloss"
	"github.com/perlsaiyan/zif/session"
)

func F() {
	log.Printf("Plugin executed")
}

var Info session.PluginInfo = session.PluginInfo{Name: "Kallisti", Version: "0.1a"}

// RegisterSession is called when a Session activates the plugin
func RegisterSession() {}

func MOTD() string {
	return lipgloss.NewStyle().Bold(true).Render("Kallisti enhancement package version " + Info.Version)

}
