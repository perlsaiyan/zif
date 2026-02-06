package main

import (
	"testing"

	"github.com/perlsaiyan/zif/session"
)

func TestKallistiTriggers(t *testing.T) {
	// Setup session
	s := &session.Session{
		Data:   make(map[string]interface{}),
		Events: session.NewEventRegistry(),
	}

	// Initialize KallistiData
	s.Data["kallisti"] = &KallistiData{
		Triggers: make([]KallistiTrigger, 0),
	}

	// Register triggers (copy logic from RegisterSession for testing)
	// Crafting
	AddKallistiTrigger(s, "Crafting", `^You craft (.+) made from (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 3 {
			return
		}
		evt := NewKallistiCraftEvent("bone", "weapon", matches[2], matches[1])
		s.FireEvent("kallisti.craft", evt)
	})
	// Carving
	AddKallistiTrigger(s, "Carving", `^You carve (.+) into (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 3 {
			return
		}
		evt := NewKallistiCraftEvent("bone", "bone", matches[1], matches[2])
		s.FireEvent("kallisti.craft", evt)
	})
	// Brewing
	AddKallistiTrigger(s, "Brewing", `^You brew (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 2 {
			return
		}
		evt := NewKallistiCraftEvent("herb", "potion", "herbs", matches[1])
		s.FireEvent("kallisti.craft", evt)
	})
	// Death
	AddKallistiTrigger(s, "Death", `^(.+?)(?: \(your follower\))? is dead!  R.I.P\.`, func(s *session.Session, matches []string) {
		if len(matches) < 2 {
			return
		}
		evt := NewKallistiDeathEvent(matches[1])
		s.FireEvent("kallisti.death", evt)
	})

	// Capture events
	var capturedEvents []session.EventData
	s.AddEvent("kallisti.craft", session.Event{
		Name: "TestCaptureCraft",
		Fn: func(s *session.Session, data session.EventData) {
			capturedEvents = append(capturedEvents, data)
		},
	})
	s.AddEvent("kallisti.death", session.Event{
		Name: "TestCaptureDeath",
		Fn: func(s *session.Session, data session.EventData) {
			capturedEvents = append(capturedEvents, data)
		},
	})

	// Test cases
	tests := []struct {
		name     string
		input    string
		expected interface{} // Expected event data
	}{
		{
			name:  "Craft Dagger",
			input: "You craft a superior dagger made from an armored warhorse bone.",
			expected: KallistiCraftEvent{
				SourceType: "bone",
				OutputType: "weapon",
				Input:      "an armored warhorse bone",
				Output:     "a superior dagger",
			},
		},
		{
			name:  "Carve Bone",
			input: "You carve some good bone from a pikeman into some superior bone from a pikeman.",
			expected: KallistiCraftEvent{
				SourceType: "bone",
				OutputType: "bone",
				Input:      "some good bone from a pikeman",
				Output:     "some superior bone from a pikeman",
			},
		},
		{
			name:  "Brew Potion",
			input: "You brew a clear potion of divine armor.",
			expected: KallistiCraftEvent{
				SourceType: "herb",
				OutputType: "potion",
				Input:      "herbs",
				Output:     "a clear potion of divine armor",
			},
		},
		{
			name:  "Death Follower",
			input: "An armored warhorse (your follower) is dead!  R.I.P.",
			expected: KallistiDeathEvent{
				Name: "An armored warhorse",
			},
		},
		{
			name:  "Death Mob",
			input: "A pikeman is dead!  R.I.P.",
			expected: KallistiDeathEvent{
				Name: "A pikeman",
			},
		},
		{
			name:  "Death Player",
			input: "Spencer is dead!  R.I.P.",
			expected: KallistiDeathEvent{
				Name: "Spencer",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capturedEvents = nil                           // Reset
			ProcessKallistiTriggers(s, tc.input, tc.input) // Pass same string as stripped for testing

			if len(capturedEvents) != 1 {
				t.Errorf("Expected 1 event, got %d", len(capturedEvents))
				return
			}

			got := capturedEvents[0]

			switch expected := tc.expected.(type) {
			case KallistiCraftEvent:
				actual, ok := got.(KallistiCraftEvent)
				if !ok {
					t.Errorf("Expected KallistiCraftEvent, got %T", got)
					return
				}
				if actual.SourceType != expected.SourceType {
					t.Errorf("SourceType: expected %q, got %q", expected.SourceType, actual.SourceType)
				}
				if actual.OutputType != expected.OutputType {
					t.Errorf("OutputType: expected %q, got %q", expected.OutputType, actual.OutputType)
				}
				if actual.Input != expected.Input {
					t.Errorf("Input: expected %q, got %q", expected.Input, actual.Input)
				}
				if actual.Output != expected.Output {
					t.Errorf("Output: expected %q, got %q", expected.Output, actual.Output)
				}
			case KallistiDeathEvent:
				actual, ok := got.(KallistiDeathEvent)
				if !ok {
					t.Errorf("Expected KallistiDeathEvent, got %T", got)
					return
				}
				if actual.Name != expected.Name {
					t.Errorf("Name: expected %q, got %q", expected.Name, actual.Name)
				}
			}
		})
	}
}
