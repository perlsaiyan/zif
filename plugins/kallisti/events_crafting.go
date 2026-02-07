package main

import (
	"github.com/perlsaiyan/zif/session"
)

type KallistiCraftEvent struct {
	session.BaseEvent
	Method        string // e.g. "tanning", "crafting", "carving", "brewing"
	SourceType    string // e.g. "hide", "bone", "herb"
	OutputType    string // e.g. "leather", "weapon", "potion"
	InputQuality  string // e.g. "pristine", "good", "poor"
	OutputQuality string // e.g. "pristine", "exquisite", "clear"
	OutputName    string // e.g. "bloodlust", "vigor", "divine armor", "dagger"
	Source        string // e.g. "an armored warhorse"
}

func NewKallistiCraftEvent(method, sourceType, outputType, inputQuality, outputQuality, outputName, source string) KallistiCraftEvent {
	return KallistiCraftEvent{
		BaseEvent:     session.NewBaseEvent(),
		Method:        method,
		SourceType:    sourceType,
		OutputType:    outputType,
		InputQuality:  inputQuality,
		OutputQuality: outputQuality,
		OutputName:    outputName,
		Source:        source,
	}
}
