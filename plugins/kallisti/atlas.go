package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/jmoiron/sqlx"
	"github.com/perlsaiyan/zif/session"
)

type AtlasRoomRecord struct {
	VNUM          string  `db:"vnum"`
	Name          string  `db:"name"`
	Terrain       string  `db:"terrain_name"`
	AreaName      string  `db:"area_name"`
	RegenHP       string  `db:"regen_hp"`
	RegenMP       string  `db:"regen_mp"`
	RegenSP       string  `db:"regen_sp"`
	SetRecall     string  `db:"set_recall"`
	Peaceful      string  `db:"peaceful"`
	Deathtrap     string  `db:"deathtrap"`
	Silent        string  `db:"silent"`
	WildMagic     string  `db:"wild_magic"`
	Bank          string  `db:"bank"`
	Narrow        string  `db:"narrow"`
	NoMagic       string  `db:"no_magic"`
	NoRecall      string  `db:"no_recall"`
	LastVisited   *string `db:"last_visited"`
	LastHarvested *string `db:"last_harvested"`
	Exits         map[string]AtlasExitRecord
}

type AtlasExitRecord struct {
	FromVnum  string  `db:"from_vnum"`
	Direction string  `db:"direction"`
	ToVnum    string  `db:"to_vnum"`
	Door      string  `db:"door"`
	Closes    string  `db:"closes"`
	Locks     string  `db:"locks"`
	KeyName   string  `db:"key_name"`
	Weight    string  `db:"weight"`
	MaxLevel  string  `db:"max_level"`
	MinLevel  string  `db:"min_level"`
	Deathtrap string  `db:"deathtrap"`
	Commands  *string `db:"commands"`
}

func ConnectAtlasDB() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", "./db/world.db")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func GetRoomByVNUM(s *session.Session, vnum string) *AtlasRoomRecord {
	d := s.Data["kallisti"].(*KallistiData)
	room, ok := d.World[vnum]
	if ok {
		return &room
	}
	return nil
}

func LoadAllRooms(s *session.Session) {
	d := s.Data["kallisti"].(*KallistiData)
	query := "SELECT * FROM rooms"
	var rooms []AtlasRoomRecord
	err := d.Atlas.Select(&rooms, query)
	if err != nil {
		log.Printf("Error loading rooms: %s", err.Error())
	}

	for _, room := range rooms {
		room.Exits = make(map[string]AtlasExitRecord)
		d.World[room.VNUM] = room
	}

}

func LoadAllExits(s *session.Session) {
	d := s.Data["kallisti"].(*KallistiData)
	query := "SELECT * FROM exits"
	var exits []AtlasExitRecord
	err := d.Atlas.Select(&exits, query)
	if err != nil {
		log.Printf("Error loading exits: %s", err.Error())
	}

	for _, exit := range exits {
		room, ok := d.World[exit.FromVnum]
		if ok {
			room.Exits[exit.Direction] = exit
		}
	}

}

func CmdRoom(s *session.Session, args string) {
	if len(args) < 1 {
		s.Output("Usage: room <vnum>\n")
		return
	}

	room := GetRoomByVNUM(s, args)
	if room == nil {
		log.Printf("VNUM not in atlas: %s", args)
		s.Output("VNUM not in atlas.\n")
		return
	}

	msg := fmt.Sprintf("Room: %+v\n", room)
	s.Output(msg)
}

func FindPathBFS(s *session.Session, fromVnum string, toVnum string) ([]string, []string) {
	fromRoom := GetRoomByVNUM(s, fromVnum)
	if fromRoom == nil {
		log.Printf("VNUM not in atlas: %s", fromVnum)
		return nil, nil
	}
	toRoom := GetRoomByVNUM(s, toVnum)
	if toRoom == nil {
		log.Printf("VNUM not in atlas: %s", toVnum)
		return nil, nil
	}
	type Path struct {
		Room      *AtlasRoomRecord
		Direction string
	}
	// Do BFS
	visited := make(map[string]bool)
	queue := make([][]*Path, 0)
	start := []*Path{{Room: fromRoom}}
	queue = append(queue, start)
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		currentRoom := path[len(path)-1].Room
		if currentRoom.VNUM == toRoom.VNUM {
			// Found the destination room, store the path
			var pathVNUMs []string
			var pathDirections []string
			for _, step := range path {
				pathVNUMs = append(pathVNUMs, step.Room.VNUM)
				if step.Room.Exits[step.Direction].Commands == nil {
					pathDirections = append(pathDirections, step.Direction)
				} else {
					pathDirections = append(pathDirections, *step.Room.Exits[step.Direction].Commands)
				}
			}
			return pathVNUMs, pathDirections
		}
		if !visited[currentRoom.VNUM] {
			visited[currentRoom.VNUM] = true
			for _, exit := range currentRoom.Exits {
				nextRoom := GetRoomByVNUM(s, exit.ToVnum)
				if nextRoom != nil {
					newPath := make([]*Path, len(path))
					copy(newPath, path)
					newPath = append(newPath, &Path{Room: nextRoom, Direction: exit.Direction})
					queue = append(queue, newPath)
				}
			}
		}
	}
	// No path found
	return nil, nil
}

func CmdBFSRoomToRoom(s *session.Session, arg string) {
	d := s.Data["kallisti"].(*KallistiData)
	if len(arg) == 0 {
		s.Output("Usage: #path <to vnum>\n")
		return
	}

	fromVnum := s.MSDP.RoomVnum
	toVnum := arg
	pathVNUMs, pathDirections := FindPathBFS(s, fromVnum, toVnum)
	if pathVNUMs == nil {
		s.Output(fmt.Sprintf("No path found from %s to %s\n", fromVnum, toVnum))
		return
	}

	d.Travel.On = true
	d.Travel.To = toVnum
	d.Travel.Length = len(pathVNUMs) - 1 // don't count the room we're in
	d.Travel.Distance = len(pathVNUMs) - 1
	// Index 1 is the first direction since we're in room 0?
	s.Output("Traveling to " + toVnum + ", sending " + pathDirections[1] + "\n")
	s.Socket.Write([]byte(pathDirections[1] + "\n"))

}

func TravelProgress(s *session.Session) string {
	d := s.Data["kallisti"].(*KallistiData)

	if !d.Travel.On {
		return ""
	}

	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	prog.Width = 20
	prog.ShowPercentage = false
	return prog.ViewAs(float64(d.Travel.Length-d.Travel.Distance) / float64(d.Travel.Length))
}

func CmdShowMap(s *session.Session, arg string) {
	// make a string containing the map
	mapString := MakeMap(s, 50, 25)

	lines := strings.Split(mapString, "\n")
	for _, line := range lines {
		s.Output(line + "\n")
	}
}

func MakeMap(s *session.Session, x int, y int) string {
	mapgrid := GetBFSGrid(s, x, y)

	// make a string containing the map
	var mapString string
	for _, row := range mapgrid {
		for _, room := range row {
			if room != nil {
				terrains := strings.Split(room.Terrain, " ")
				t := GetTerrainByName(terrains[0])
				glyph := GetTerrainMapSymbol(room.Terrain)
				if room.VNUM == s.MSDP.RoomVnum {
					t := GetTerrainByName("You")
					style := GetStyleByTerrain(t)
					mapString += style.Render("@")
				} else if t != nil {
					style := GetStyleByTerrain(t)
					//color := GetTerrainMapColor(room.Terrain)
					//log.Printf("Room VNUM: %s, Terrain: %s, Symbol: %s, Color: %s\n", room.VNUM, room.Terrain, glyph, color)
					mapString += style.Render(glyph)
				} else {
					mapString += ":"
				}
			} else {
				mapString += " "
			}
		}
		mapString += "\n"
	}

	return mapString
}

func GetBFSGrid(s *session.Session, x int, y int) [][]*AtlasRoomRecord {
	//d := s.Data["kallisti"].(*KallistiData)
	fromRoom := GetRoomByVNUM(s, s.MSDP.RoomVnum)
	if fromRoom == nil {
		log.Printf("VNUM not in atlas: %s", s.MSDP.RoomVnum)
		return nil
	}

	width := x
	height := y
	overscan := 0
	cenH := height / 2
	cenW := width / 2
	matrix := make([][]*AtlasRoomRecord, height+overscan)
	for i := range matrix {
		matrix[i] = make([]*AtlasRoomRecord, width+overscan)
	}

	type MapPoint struct {
		Room *AtlasRoomRecord
		X    int
		Y    int
	}

	queue := make([]MapPoint, 0)
	visited := make(map[string]bool)
	queue = append(queue, MapPoint{Room: fromRoom, X: cenW, Y: cenH})
	for len(queue) > 0 {
		here := queue[0]
		queue = queue[1:]
		if _, ok := visited[here.Room.VNUM]; ok {
			continue
		}
		visited[here.Room.VNUM] = true
		if matrix[here.Y][here.X] == nil {
			matrix[here.Y][here.X] = here.Room
			for _, exit := range here.Room.Exits {
				nextRoom := GetRoomByVNUM(s, exit.ToVnum)
				if nextRoom != nil {
					if exit.Direction == "north" && here.Y-1 >= 0 {
						queue = append(queue, MapPoint{Room: nextRoom, X: here.X, Y: here.Y - 1})
					}
					if exit.Direction == "south" && here.Y+1 < len(matrix) {
						queue = append(queue, MapPoint{Room: nextRoom, X: here.X, Y: here.Y + 1})
					}
					if exit.Direction == "east" && here.X+1 < len(matrix[here.Y]) {
						queue = append(queue, MapPoint{Room: nextRoom, X: here.X + 1, Y: here.Y})
					}
					if exit.Direction == "west" && here.X-1 >= 0 {
						queue = append(queue, MapPoint{Room: nextRoom, X: here.X - 1, Y: here.Y})
					}
				}
			}
		}
	}
	return matrix
}
