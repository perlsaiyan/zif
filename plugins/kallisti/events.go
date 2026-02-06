package main

import (
	"github.com/perlsaiyan/zif/session"
)

type KallistiCraftEvent struct {
	session.BaseEvent
	SourceType string // e.g. "bone", "herb"
	OutputType string // e.g. "potion", "refined bone"
	Input      string // e.g. "some good bone"
	Output     string // e.g. "an exquisite knife"
}

type KallistiDeathEvent struct {
	session.BaseEvent
	Name string // Name of the mob or player
}

func NewKallistiCraftEvent(sourceType, outputType, input, output string) KallistiCraftEvent {
	return KallistiCraftEvent{
		BaseEvent:  session.NewBaseEvent(),
		SourceType: sourceType,
		OutputType: outputType,
		Input:      input,
		Output:     output,
	}
}

func NewKallistiDeathEvent(name string) KallistiDeathEvent {
	return KallistiDeathEvent{
		BaseEvent: session.NewBaseEvent(),
		Name:      name,
	}
}
