package session

import (
	"context"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type TickerRegistry struct {
	Context context.Context
	Entries map[string]*TickerRecord
}

type TickerRecord struct {
	Name       string
	Interval   int
	Fn         func(*Session)
	Command    string
	LastFire   time.Time
	NextFire   time.Time
	Count      uint
	Iterations uint
}

func NewTickerRegistry(ctx context.Context, s *Session) {
	s.Tickers = &TickerRegistry{Context: ctx, Entries: make(map[string]*TickerRecord)}
	go SessionTicker(s)
}

func (s *Session) AddTicker(ticker *TickerRecord) {
	s.Tickers.Entries[ticker.Name] = ticker
}

func SessionTicker(s *Session) {

	s.Output("Launching ticker!!\n")
	for {
		select {
		case <-s.Context.Done():
			s.Output("KILLING TICKER!!!\n")
			return

		default:

			for k, v := range s.Tickers.Entries {
				if v.NextFire.Before(time.Now()) {
					v.LastFire = time.Now()
					//log.Printf("Firing ticker " + v.Name + "\n")
					if v.Fn != nil {
						v.Fn(s)
					} else if len(v.Command) > 0 {
						s.Socket.Write([]byte(v.Command + "\n"))
					}
					v.NextFire = time.Now().Add(time.Duration(v.Interval) * time.Millisecond)
					s.Tickers.Entries[k] = v
				}
			}

			time.Sleep(50 * time.Millisecond)
		}

	}

}

func makeTickerRow(name string, last time.Time, next time.Time) table.Row {

	return table.NewRow(table.RowData{
		"name":      name,
		"last fire": time.Since(last).Round(time.Second).String() + " ago",
		"next fire": time.Until(next).Round(time.Second),
	})
}

func CmdTickers(s *Session, cmd string) {
	var rows []table.Row
	for _, i := range s.Tickers.Entries {
		rows = append(rows, makeTickerRow(i.Name, i.LastFire, i.NextFire))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 25).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("last fire", "Last Fire", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("next fire", "Next Fire", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}

// test context cancel
func CmdCancelTicker(s *Session, cmd string) {
	if s.Cancel != nil {
		s.Cancel()
	}
}

func CmdTestTicker(s *Session, cmd string) {
	s.Tickers.Entries["test1"] = &TickerRecord{
		Name:     "test1",
		Interval: 5000,
		Command:  "smile",
		NextFire: time.Now().Add(5000 * time.Millisecond),
		LastFire: time.Now(),
	}
}
