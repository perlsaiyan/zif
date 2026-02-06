package session

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// Mock event data for testing
type TestEventData struct {
	BaseEvent
	Message string
	Count   int
}

func TestLuaRegisterEvent(t *testing.T) {
	// Setup session
	s := &Session{
		Name:     "test",
		Events:   NewEventRegistry(),
		LuaState: lua.NewState(),
		Modules:  NewModuleRegistry(),
	}

	// Register Lua API
	s.RegisterLuaAPI()

	// Mock module context
	SetCurrentModule(s.LuaState, "test_module")
	s.Modules.Modules["test_module"] = &Module{Name: "test_module"}

	// Lua script to register event and capture data
	script := `
		captured_data = nil
		session.register_event("test.event", function(evt)
			captured_data = evt
		end)
	`
	if err := s.LuaState.DoString(script); err != nil {
		t.Fatalf("Failed to run Lua script: %v", err)
	}

	// Fire event
	evt := TestEventData{
		BaseEvent: NewBaseEvent(),
		Message:   "Hello Lua",
		Count:     42,
	}
	s.FireEvent("test.event", evt)

	// Check captured data in Lua
	val := s.LuaState.GetGlobal("captured_data")
	if val == lua.LNil {
		t.Fatal("Lua event handler was not called (captured_data is nil)")
	}

	tbl, ok := val.(*lua.LTable)
	if !ok {
		t.Fatalf("captured_data is not a table, got %T", val)
	}

	// Verify fields
	msg := tbl.RawGetString("Message")
	if msg.String() != "Hello Lua" {
		t.Errorf("Message: expected 'Hello Lua', got %q", msg.String())
	}

	count := tbl.RawGetString("Count")
	if count.String() != "42" { // Lua numbers are converted to string when using RawGetString? No, RawGetString expects key string.
		// Let's check type properly
		num, ok := count.(lua.LNumber)
		if !ok {
			// Try getting it as a number directly if RawGetString failed to return the value we expected (wait, RawGetString takes key string, returns LValue)
			// Re-read docs/code: LTable.RawGetString(key string) -> LValue
		}
		if float64(num) != 42 {
			t.Errorf("Count: expected 42, got %v", count)
		}
	}
}
