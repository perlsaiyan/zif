package session

import (
	"time"
)

// AddLuaTimer adds a timer that can be managed by the Lua timer system
func (s *Session) AddLuaTimer(ticker *TickerRecord) {
	if s.Tickers == nil {
		NewTickerRegistry(s.Context, s)
	}
	
	// Set initial fire time if not already set
	if ticker.NextFire.Before(s.Birth) {
		ticker.NextFire = time.Now().Add(time.Duration(ticker.Interval) * time.Millisecond)
	}
	
	s.Tickers.Entries[ticker.Name] = ticker
}

// RemoveLuaTimer removes a Lua timer
func (s *Session) RemoveLuaTimer(name string) {
	if s.Tickers != nil {
		delete(s.Tickers.Entries, name)
	}
}

