package session

import (
	"context"
	"time"
)

type Ticker struct {
	Context context.Context
}

func NewSessionTicker(ctx context.Context, s *Session) {
	s.Ticker = &Ticker{Context: ctx}
	go SessionTicker(s)
}

func SessionTicker(s *Session) {

	s.Output("Launching ticker!!\n")
	for {
		select {
		case <-s.Context.Done():
			s.Output("KILLING TICKER!!!\n")
			return

		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

}

// test context cancel
func CmdCancelTicker(s *SessionHandler, cmd string) {
	if s.ActiveSession().Cancel != nil {
		s.ActiveSession().Cancel()
	}
}
