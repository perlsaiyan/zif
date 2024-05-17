package session

import (
	"plugin"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/perlsaiyan/zif/config"
)

type PluginRegistry struct {
	Plugins map[string]PluginInfo
}

type PluginInfo struct {
	Plugin      *plugin.Plugin
	Name        string
	Version     string
	Description string
}

func LoadPlugin(path string, config *config.Config) (*plugin.Plugin, error) {

	// In order to load a plugin, do something like this:

	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}

func NewPluginRegistry() *PluginRegistry {
	pr := PluginRegistry{Plugins: make(map[string]PluginInfo)}
	return &pr
}

func makePluginRow(p PluginInfo) table.Row {

	return table.NewRow(table.RowData{
		"name":        p.Name,
		"version":     p.Version,
		"description": p.Description,
	})
}

func CmdPlugins(s *Session, cmd string, h *SessionHandler) {
	var rows []table.Row
	for _, i := range h.Plugins.Plugins {
		rows = append(rows, makePluginRow(i))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 25).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("version", "Version", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("description", "Description", 40).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}
