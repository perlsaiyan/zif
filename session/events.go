package session

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type EventFunction func(*Session, EventData)

type EventData struct{}

type Event struct {
	Name    string
	Event   string
	Enabled bool
	Fn      EventFunction
	Count   uint
}

type EventRegistry struct {
	Events map[string]Event
}

func NewEventRegistry() *EventRegistry {
	er := EventRegistry{Events: make(map[string]Event)}
	return &er
}

func makeEventRow(evt Event) table.Row {

	return table.NewRow(table.RowData{
		"name":    evt.Name,
		"enabled": evt.Enabled,
		"count":   evt.Count,
	})
}

func CmdEvents(s *Session, cmd string, h *SessionHandler) {
	var rows []table.Row
	for _, i := range s.Events.Events {
		rows = append(rows, makeEventRow(i))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 25).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("enabled", "Enabled", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("count", "Count", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}
