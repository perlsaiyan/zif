package session

import (
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type EventFunction func(*Session, EventData)

type EventData interface {
	Timestamp() time.Time
}

type BaseEvent struct {
	ts time.Time
}

func (b BaseEvent) Timestamp() time.Time {
	return b.ts
}

func NewBaseEvent() BaseEvent {
	return BaseEvent{ts: time.Now()}
}

type Event struct {
	Name    string
	Event   string
	Enabled bool
	Fn      EventFunction
	Count   uint
}

type EventRegistry struct {
	Events map[string][]Event
	mu     sync.Mutex
}

func NewEventRegistry() *EventRegistry {

	er := EventRegistry{Events: make(map[string][]Event)}
	return &er
}

func makeEventRow(e string, evt Event) table.Row {

	return table.NewRow(table.RowData{
		"event":   e,
		"name":    evt.Name,
		"enabled": evt.Enabled,
		"count":   evt.Count,
	})
}

func CmdEvents(s *Session, cmd string) {
	var rows []table.Row
	for e, i := range s.Events.Events {
		for _, j := range i {
			rows = append(rows, makeEventRow(e, j))
		}
	}

	t := table.New([]table.Column{
		table.NewColumn("event", "Event", 15).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("name", "Name", 25).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("enabled", "Enabled", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("count", "Count", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}

func (s *Session) AddEvent(hook string, evt Event) {

	s.Events.mu.Lock()
	defer s.Events.mu.Unlock()

	s.Events.Events[hook] = append(s.Events.Events[hook], evt)
}

func (s *Session) FireEvent(name string, evt EventData) {

	s.Events.mu.Lock()
	defer s.Events.mu.Unlock()

	for _, i := range s.Events.Events[name] {
		i.Fn(s, evt)
	}
}
