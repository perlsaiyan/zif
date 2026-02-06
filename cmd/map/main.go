package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"plugin"

	"image/color"
	"image/draw"
	"image/png"

	ts "github.com/azul3d/engine/tmx"
	"github.com/jmoiron/sqlx"
	kallisti "github.com/perlsaiyan/zif/protocol"
	"github.com/perlsaiyan/zif/session"
)

const (
	Dirt_Tan = iota
	Dirt_Brown
	Dirt_Dark
	Rock_White
	Rock_Gray
	Rock_Dark
	Rock_Black
	Hole_Brown
	Hole_Black
	Mud_Brown
	Grass
	Grass_Light
	Grass_Dark
	Grass_Dead
	Soil
	Sand
	Snow_1
	Snow_2
	Gravel_1
	Dirt_Roots
	Water_Shallows_Dirt
	Water
	Water_Deep
	Water_Purple
	Water_Green
	Lava
	Water_Shallows_Sand
	Ice
	Ice_Melting
	Earth_Cracked
	Stone_White
	Stone_Tan
	Mudstone_Gray
	Mudstone_Brown
)

const mapPath = "maps/terrain-v7.tsx" // Path to your Tiled Map.
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

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func LoadAllRooms(d *sqlx.DB) *map[string]AtlasRoomRecord {
	world := make(map[string]AtlasRoomRecord)
	query := "SELECT * FROM rooms"
	var rooms []AtlasRoomRecord
	err := d.Select(&rooms, query)
	if err != nil {
		log.Printf("Error loading rooms: %s", err.Error())
	}

	for _, room := range rooms {
		room.Exits = make(map[string]AtlasExitRecord)
		world[room.VNUM] = room
	}
	return &world
}

func LoadAllExits(d *sqlx.DB, world *map[string]AtlasRoomRecord) {
	query := "SELECT * FROM exits"
	var exits []AtlasExitRecord
	err := d.Select(&exits, query)
	if err != nil {
		log.Printf("Error loading exits: %s", err.Error())
	}

	for _, exit := range exits {
		room, ok := (*world)[exit.FromVnum]
		if ok {
			room.Exits[exit.Direction] = exit
		}
	}

}

func getImageFromFilePath(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image, err
}

func main() {

	t := ts.Tileset{}
	tsx, err := os.ReadFile(mapPath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err.Error())
		os.Exit(1)
	}

	err = t.Load(tsx)
	if err != nil {
		fmt.Printf("Error loading tileset: %s\n", err.Error())
		os.Exit(1)
	}

	tiles, err := getImageFromFilePath("maps/" + t.Image.Source)
	if err != nil {
		fmt.Printf("Error reading image: %s\n", err.Error())
		os.Exit(1)
	}

	s := &session.Session{MSDP: kallisti.NewMSDP()}

	fmt.Printf("Session: %v\n", s)
	p, err := plugin.Open("./kallisti.so")
	if err != nil {
		fmt.Printf("error opening plugin: %s", err.Error())
		os.Exit(2)
	}

	f, err := p.Lookup("ConnectAtlasDB")
	if err != nil {
		fmt.Printf("error connecting DB: %s", err.Error())
		os.Exit(2)
	}

	db := f.(func() *sqlx.DB)()

	bestTerrain, err := p.Lookup("GetBestTerrainByColor")
	if err != nil {
		fmt.Printf("error looking up GetBestTerrainByColor: %s", err.Error())
		os.Exit(2)
	}

	World := LoadAllRooms(db)
	LoadAllExits(db, World)

	// create a blank image
	blankX := 3200
	blankY := 3200
	upLeft := image.Point{0, 0}
	lowRight := image.Point{blankX, blankY}

	baseimage := image.NewRGBA(image.Rectangle{upLeft, lowRight})
	white := color.RGBA{0, 0, 0, 255}
	draw.Draw(baseimage, baseimage.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	offx := 0

	grid := GetBFSGrid(World, "2347", blankX/16, blankY/16)
	//fmt.Printf("Grid: %d x %d\n", len(grid), len(grid[0]))
	for x := 0; x < len(grid); x += 2 {
		offy := 0
		for y := 0; y < len(grid); y += 2 {
			var tl, tr, bl, br string

			if grid[x][y] == nil {
				tl = ""
			} else {
				tl = bestTerrain.(func(string) string)(grid[x][y].Terrain)
			}

			if grid[x+1][y] == nil {
				tr = ""
			} else {
				tr = bestTerrain.(func(string) string)(grid[x+1][y].Terrain)
			}

			if grid[x][y+1] == nil {
				bl = ""
			} else {
				bl = bestTerrain.(func(string) string)(grid[x][y+1].Terrain)
			}

			if grid[x+1][y+1] == nil {
				br = ""
			} else {
				br = bestTerrain.(func(string) string)(grid[x+1][y+1].Terrain)
			}

			m := FindTiles(t, tl, tr, bl, br)

			if m > -1 {
				cropSize := image.Rect(0, 0, 31, 31)
				idx := m % 32
				row := m / 32
				x1 := idx * 32
				y1 := row * 32

				cropSize = cropSize.Add(image.Point{x1, y1})
				croppedImage := tiles.(SubImager).SubImage(cropSize)

				offset := image.Point{offx * 32, offy * 32}
				// Create a rectangle at the offset position with the size of the source image
				r := image.Rectangle{Min: offset, Max: offset.Add(croppedImage.Bounds().Size())}
				//fmt.Printf("Offset: %s, %s, sprite bounds %v\n", offset, r, croppedImage.Bounds())
				draw.Draw(baseimage, r, croppedImage, croppedImage.Bounds().Min, draw.Src)

			} else {
				fmt.Printf("Unknown tile: %s, %s, %s, %s\n", tl, tr, bl, br)
			}

			offy += 1
		}
		offx += 1
	}

	kallisti, err := os.Create("kallisti.png")
	if err != nil {
		panic(err)
	}

	defer kallisti.Close()
	if err := png.Encode(kallisti, baseimage); err != nil {
		panic(err)
	}

}

func FindTiles(t ts.Tileset, tl string, tr string, bl string, br string) int {
	q1 := getTerrainNumber(tl)
	q2 := getTerrainNumber(tr)
	q3 := getTerrainNumber(bl)
	q4 := getTerrainNumber(br)
	if q1 == -1 && (q2 == q3 && q2 == q4) {
		q1 = q2
	}
	if q2 == -1 && (q1 == q3 && q1 == q4) {
		q2 = q1
	}
	if q3 == -1 && (q2 == q1 && q2 == q4) {
		q3 = q2
	}
	if q4 == -1 && (q2 == q3 && q2 == q1) {
		q4 = q2
	}
	if q1 == -1 && q2 == -1 && q3 == q4 {
		q1 = q3
		q2 = q3
	}

	if q1 == -1 && q3 == -1 && q2 == q4 {
		q1 = q2
		q3 = q2
	}

	if q1 == Water && q3 == Water {
		q2 = Water
		q4 = Water
	}

	if q1 == Water && q2 == Water {
		q3 = Water
		q4 = Water
	}

	if q1 == q2 && q3 == -1 && q4 == -1 {
		q3 = q1
		q4 = q1
	}

	if q1 == q3 && q2 == -1 && q4 == -1 {
		q2 = q1
		q4 = q1
	}

	for _, tile := range t.Tiles {
		if tile.Terrain[0] == q1 && tile.Terrain[1] == q2 && tile.Terrain[2] == q3 && tile.Terrain[3] == q4 {
			return tile.ID
		}
	}
	return -1
}

func getTerrainNumber(t string) int {
	switch t {
	case "Water":
		return Water
	case "Beach":
		return Water_Shallows_Sand
	case "Deep":
		return Water_Deep
	case "Field":
		return Grass
	case "Desert":
		return Sand
	case "City":
		return Earth_Cracked
	case "Swamp":
		return Mud_Brown
	case "Pasture":
		return Grass_Dark
	case "Hills":
		return Grass_Light
	case "Path":
		return Gravel_1
	case "Mountains":
		return Rock_Gray
	case "Inside":
		return Earth_Cracked
	case "Bridge":
		return Earth_Cracked
	case "Forest":
		return Grass
	case "Jungle":
		return Grass
	case "Stairs":
		return Grass_Dead

	default:
		fmt.Printf("Unknown terrain %s\n", t)
		return -1
	}

}

func GetBFSGrid(world *map[string]AtlasRoomRecord, start string, x int, y int) [][]*AtlasRoomRecord {
	//d := s.Data["kallisti"].(*KallistiData)
	fromRoom := (*world)[start]

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
	queue = append(queue, MapPoint{Room: &fromRoom, X: cenW, Y: cenH})
	for len(queue) > 0 {
		here := queue[0]
		queue = queue[1:]
		if _, ok := visited[here.Room.VNUM]; ok {
			continue
		}
		visited[here.Room.VNUM] = true
		if matrix[here.Y][here.X] == nil {
			matrix[here.Y][here.X] = here.Room
			var exits []AtlasExitRecord
			for _, k := range []string{"north", "east", "south", "west"} {
				if e, ok := here.Room.Exits[k]; ok {
					exits = append(exits, e)
				}
			}

			for _, exit := range exits {
				nextRoom := (*world)[exit.ToVnum]

				if exit.Direction == "north" && here.Y-1 >= 0 {
					queue = append(queue, MapPoint{Room: &nextRoom, X: here.X, Y: here.Y - 1})
				} else if exit.Direction == "south" && here.Y+1 < len(matrix) {
					queue = append(queue, MapPoint{Room: &nextRoom, X: here.X, Y: here.Y + 1})
				} else if exit.Direction == "east" && here.X+1 < len(matrix[here.Y]) {
					queue = append(queue, MapPoint{Room: &nextRoom, X: here.X + 1, Y: here.Y})
				} else if exit.Direction == "west" && here.X-1 >= 0 {
					queue = append(queue, MapPoint{Room: &nextRoom, X: here.X - 1, Y: here.Y})
				}
			}
		}
	}
	return matrix
}
