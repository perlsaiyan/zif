package main

import (
	"fmt"
	"regexp"

	"github.com/perlsaiyan/zif/session"
)

var (
	reRoomNoCompass = regexp.MustCompile(`^.* (\[ [ NSWEUD<>v^\|\(\)\[\]]* \] *$)`)
	reRoomCompass   = regexp.MustCompile(`^.* \|`)
	reRoomHere      = regexp.MustCompile(`^Here +- `)
	reRoomNoExits   = regexp.MustCompile(`^.* \[ No exits! \]`)
)

func ParseRoom(s *session.Session, evt session.EventData) {
	d := s.Data["kallisti"].(*KallistiData)

	// We're not looking for a room off this prompt
	if d.CurrentRoomRingLogID < 0 {
		d.LastPrompt = s.Ringlog.GetCurrentRingNumber()
		return
	}

	msg := fmt.Sprintf("Probable room from %d to %d, Room title @ %d\n", d.LastPrompt, s.Ringlog.GetCurrentRingNumber(), d.CurrentRoomRingLogID)
	s.Output(msg)
	d.CurrentRoomRingLogID = -1
	d.LastPrompt = s.Ringlog.GetCurrentRingNumber()

}

func PossibleRoomScanner(s *session.Session, matches session.ActionMatches) {
	d := s.Data["kallisti"].(*KallistiData)

	regexps := []*regexp.Regexp{
		reRoomCompass,
		reRoomNoCompass,
		reRoomHere,
		reRoomNoExits,
	}

	for _, re := range regexps {
		if re.MatchString(matches.Line) {
			d.CurrentRoomRingLogID = s.Ringlog.GetCurrentRingNumber()
			break
		}
	}
}
