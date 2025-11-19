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
	reANSI          = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
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
	ScanRoom(s, d)
	d.CurrentRoomRingLogID = -1
	d.LastPrompt = s.Ringlog.GetCurrentRingNumber()

	HandleTravel(s, d)

}

func ScanRoom(s *session.Session, d *KallistiData) {
	start := d.CurrentRoomRingLogID
	end := s.Ringlog.GetCurrentRingNumber()

	d.CurrentRoom = CurrentRoom{
		Vnum:    s.MSDP.GetString("ROOM_VNUM"),
		Exits:   make([]string, 0),
		Objects: make([]string, 0),
		Mobs:    make([]string, 0),
	}

	lines := s.Ringlog.GetLog(start, end)
	if len(lines) == 0 {
		return
	}

	// Title is the first line
	// Strip ANSI for title
	d.CurrentRoom.Title = reANSI.ReplaceAllString(lines[0].Message, "")

	inDescription := true

	for i, line := range lines {
		if i == 0 {
			continue // Skip title
		}

		// Check for mobs
		if (len(line.Message) > 7 && line.Message[0:7] == "\x1b[1;37m") || (len(line.Message) > 14 && line.Message[0:14] == "\x1b[0;37m\x1b[1;37m") {
			inDescription = false
			d.CurrentRoom.Mobs = append(d.CurrentRoom.Mobs, reANSI.ReplaceAllString(line.Message, ""))
			continue
		}

		// Check for objects
		if len(line.Message) > 7 && line.Message[0:7] == "\x1b[0;37m" {
			// Check if it's not a description line that happens to start with white
			// Objects usually don't start with "Here -" or "Inside" or "The" (if it's a title, but we passed title)
			// Actually, based on logs, objects start with \x1b[0;37m and are single lines usually.
			// But description lines can also be white.
			// However, description lines usually follow the title block.
			// Let's assume once we hit an object/mob, we are out of description.

			// Heuristic: Objects often start with "A " or "An " or "Some "
			// But so can descriptions.
			// The log shows objects come after the description block.
			// Let's look for the specific object color code at start of line.
			inDescription = false
			d.CurrentRoom.Objects = append(d.CurrentRoom.Objects, reANSI.ReplaceAllString(line.Message, ""))
			continue
		}

		if inDescription {
			// Filter out the "Inside" / "City Path" / Exits lines which are usually lines 1-2
			// They contain the compass.
			if reRoomCompass.MatchString(line.Message) || reRoomNoCompass.MatchString(line.Message) {
				continue
			}

			cleanLine := reANSI.ReplaceAllString(line.Message, "")
			if cleanLine != "" {
				if d.CurrentRoom.Description != "" {
					d.CurrentRoom.Description += " "
				}
				d.CurrentRoom.Description += cleanLine
			}
		}
	}
}

func HandleTravel(s *session.Session, d *KallistiData) {
	if !d.Travel.On {
		return
	}

	// We're in travel mode, so we need to check if we've arrived
	if d.Travel.To == s.MSDP.GetString("ROOM_VNUM") {
		s.Output("Arrived at destination!\n")
		d.Travel.On = false
		return
	}

	// We're not there yet, so we need to keep moving
	path, directions := FindPathBFS(s, s.MSDP.GetString("ROOM_VNUM"), d.Travel.To)
	if path == nil {
		s.Output("No path found!\n")
		d.Travel.On = false
		return
	}

	// Move to the next room, which is the second room in the path
	d.Travel.Distance = len(path) - 1

	// Check if we have a valid path with at least one direction
	if len(directions) == 0 {
		s.Output("No valid path found\n")
		d.Travel.On = false
		return
	}

	room := GetRoomByVNUM(s, path[0])
	// Use the first direction (index 0) since directions contains directions between rooms
	var method string
	if room.Exits[directions[0]].Commands != nil && *room.Exits[directions[0]].Commands != "" {
		method = *room.Exits[directions[0]].Commands
	} else {
		method = directions[0]
	}
	s.Output(fmt.Sprintf("Moving to %s\n", method))
	s.Socket.Write([]byte(method + "\n"))
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
