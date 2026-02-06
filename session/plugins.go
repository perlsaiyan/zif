package session

import (
	"plugin"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
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

// NewPluginRegistry creates and initializes a new PluginRegistry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		Plugins: make(map[string]PluginInfo),
	}
}

// makePluginRow creates a table row for the given PluginInfo.
func makePluginRow(p PluginInfo) table.Row {
	return table.NewRow(table.RowData{
		"name":        p.Name,
		"version":     p.Version,
		"description": p.Description,
	})
}

// CmdPlugins generates a table view of all loaded plugins and outputs it to the session.
func CmdPlugins(s *Session, cmd string) {
	h := s.Handler
	var rows []table.Row
	for _, pluginInfo := range h.Plugins.Plugins {
		rows = append(rows, makePluginRow(pluginInfo))
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
