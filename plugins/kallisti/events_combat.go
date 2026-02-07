package main

import (
	"github.com/perlsaiyan/zif/session"
)

type KallistiDeathEvent struct {
	session.BaseEvent
	Name string // Name of the mob or player
}

func NewKallistiDeathEvent(name string) KallistiDeathEvent {
	return KallistiDeathEvent{
		BaseEvent: session.NewBaseEvent(),
		Name:      name,
	}
}
