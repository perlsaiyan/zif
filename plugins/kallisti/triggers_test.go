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

	// Register triggers
	AddKallistiTrigger(s, "Tanning", `^You tan (.+) hide \(from (.+)\) into (.+) leather \(from .+\)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 4 {
			return
		}
		evt := NewKallistiCraftEvent("tanning", "hide", "leather", matches[1], matches[3], "", matches[2])
		s.FireEvent("kallisti.craft", evt)
	})
	AddKallistiTrigger(s, "Bonecrafting", `^You craft an? (.+?) (.+) made from (.+) bone\.`, func(s *session.Session, matches []string) {
		if len(matches) < 4 {
			return
		}
		evt := NewKallistiCraftEvent("bonecrafting", "bone", matches[2], "", matches[1], matches[2], matches[3])
		s.FireEvent("kallisti.craft", evt)
	})
	AddKallistiTrigger(s, "Carving", `^You carve some (.+) bone \(from (.+)\) into some (.+) bone \(from .+\)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 4 {
			return
		}
		evt := NewKallistiCraftEvent("carving", "bone", "bone", matches[1], matches[3], "", matches[2])
		s.FireEvent("kallisti.craft", evt)
	})
	AddKallistiTrigger(s, "Brewing", `^You brew an? (.+) potion of (.+)\.`, func(s *session.Session, matches []string) {
		if len(matches) < 3 {
			return
		}
		evt := NewKallistiCraftEvent("brewing", "herb", "potion", "", matches[1], matches[2], "")
		s.FireEvent("kallisti.craft", evt)
	})
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
		expected interface{}
	}{
		{
			name:  "Tan Hide",
			input: "You tan pristine hide (from an armored warhorse) into pristine leather (from an armored warhorse).",
			expected: KallistiCraftEvent{
				Method:        "tanning",
				SourceType:    "hide",
				OutputType:    "leather",
				InputQuality:  "pristine",
				OutputQuality: "pristine",
				Source:        "an armored warhorse",
			},
		},
		{
			name:  "Bonecraft Dagger",
			input: "You craft a pristine dagger made from a skirmisher bone.",
			expected: KallistiCraftEvent{
				Method:        "bonecrafting",
				SourceType:    "bone",
				OutputType:    "dagger",
				OutputQuality: "pristine",
				OutputName:    "dagger",
				Source:        "a skirmisher",
			},
		},
		{
			name:  "Carve Bone",
			input: "You carve some pristine bone (from a skirmisher) into some exquisite bone (from a skirmisher).",
			expected: KallistiCraftEvent{
				Method:        "carving",
				SourceType:    "bone",
				OutputType:    "bone",
				InputQuality:  "pristine",
				OutputQuality: "exquisite",
				Source:        "a skirmisher",
			},
		},
		{
			name:  "Brew Potion",
			input: "You brew a clear potion of divine armor.",
			expected: KallistiCraftEvent{
				Method:        "brewing",
				SourceType:    "herb",
				OutputType:    "potion",
				OutputQuality: "clear",
				OutputName:    "divine armor",
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
			capturedEvents = nil
			ProcessKallistiTriggers(s, tc.input, tc.input)

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
				if actual.Method != expected.Method {
					t.Errorf("Method: expected %q, got %q", expected.Method, actual.Method)
				}
				if actual.SourceType != expected.SourceType {
					t.Errorf("SourceType: expected %q, got %q", expected.SourceType, actual.SourceType)
				}
				if actual.OutputType != expected.OutputType {
					t.Errorf("OutputType: expected %q, got %q", expected.OutputType, actual.OutputType)
				}
				if actual.InputQuality != expected.InputQuality {
					t.Errorf("InputQuality: expected %q, got %q", expected.InputQuality, actual.InputQuality)
				}
				if actual.OutputQuality != expected.OutputQuality {
					t.Errorf("OutputQuality: expected %q, got %q", expected.OutputQuality, actual.OutputQuality)
				}
				if actual.OutputName != expected.OutputName {
					t.Errorf("OutputName: expected %q, got %q", expected.OutputName, actual.OutputName)
				}
				if actual.Source != expected.Source {
					t.Errorf("Source: expected %q, got %q", expected.Source, actual.Source)
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
