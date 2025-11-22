package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/perlsaiyan/zif/session"
)

var (
	reRoomNoCompass    = regexp.MustCompile(`^.* (\[ [ NSWEUD<>v^\|\(\)\[\]]* \] *$)`)
	reRoomCompass      = regexp.MustCompile(`^.* \|`)
	reRoomHere         = regexp.MustCompile(`^Here +- `)
	reRoomNoExits      = regexp.MustCompile(`^.* \[ No exits! \]`)
	reANSI             = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	reQuantityParens   = regexp.MustCompile(`\s*\((\d+)\)$`)
	reQuantityBrackets = regexp.MustCompile(`\s*\[(\d+)\]$`)
)

func ParseRoom(s *session.Session, evt session.EventData) {
	d := s.Data["kallisti"].(*KallistiData)

	// We're not looking for a room off this prompt
	if d.CurrentRoomRingLogID < 0 {
		d.LastPrompt = s.Ringlog.GetCurrentRingNumber()
		return
	}

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
		Objects: make([]RoomEntity, 0),
		Mobs:    make([]RoomEntity, 0),
	}

	// Populate exits from Atlas
	room := GetRoomByVNUM(s, d.CurrentRoom.Vnum)
	if room != nil {
		for dir := range room.Exits {
			d.CurrentRoom.Exits = append(d.CurrentRoom.Exits, dir)
		}
	}

	lines := s.Ringlog.GetLog(start, end)
	if len(lines) == 0 {
		return
	}

	// Title is the first line
	d.CurrentRoom.Title = reANSI.ReplaceAllString(lines[0].Message, "")

	// State machine constants
	const (
		ModeDescription = iota
		ModeObjects
		ModeMobs
	)

	mode := ModeDescription

	for i, line := range lines {
		// Skip Title (0), Terrain (1), Compass (2)
		if i < 3 {
			continue
		}

		msg := line.Message

		// Check for state transitions
		if mode == ModeDescription {
			if strings.HasPrefix(msg, "\x1b[0;37m") {
				mode = ModeObjects
			} else if strings.HasPrefix(msg, "\x1b[1;37m") {
				mode = ModeMobs
			}
		} else if mode == ModeObjects {
			if strings.HasPrefix(msg, "\x1b[1;37m") {
				mode = ModeMobs
			}
		}

		// Process based on current mode
		switch mode {
		case ModeDescription:
			cleanLine := reANSI.ReplaceAllString(msg, "")
			if cleanLine != "" {
				if d.CurrentRoom.Description != "" {
					d.CurrentRoom.Description += " "
				}
				d.CurrentRoom.Description += cleanLine
			}
		case ModeObjects:
			// If we hit a blank line or just a color reset, we might be done with objects/mobs
			// Check for end of block (blank line or just color code)
			if strings.TrimSpace(reANSI.ReplaceAllString(msg, "")) == "" {
				return
			}

			d.CurrentRoom.Objects = append(d.CurrentRoom.Objects, ParseEntity(reANSI.ReplaceAllString(msg, ""), reQuantityParens))
		case ModeMobs:
			// Check for end of block
			if strings.TrimSpace(reANSI.ReplaceAllString(msg, "")) == "" {
				return
			}
			d.CurrentRoom.Mobs = append(d.CurrentRoom.Mobs, ParseEntity(reANSI.ReplaceAllString(msg, ""), reQuantityBrackets))
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

func ParseEntity(line string, re *regexp.Regexp) RoomEntity {
	line = strings.TrimSpace(line)
	// Check for quantity
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		qty, err := strconv.Atoi(matches[1])
		if err == nil {
			return RoomEntity{
				Name:     strings.TrimSpace(re.ReplaceAllString(line, "")),
				Quantity: qty,
			}
		}
	}

	return RoomEntity{
		Name:     strings.TrimSpace(line),
		Quantity: 1,
	}
}
