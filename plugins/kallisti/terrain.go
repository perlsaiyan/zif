package main

import (
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Terrain struct {
	Name       string
	Symbol     string
	SymbolSort int
	Color      string
	ColorSort  int
	Weight     int
	BgColor    string
	Impassable bool
	Style      lipgloss.Style
}

var TerrainList = []Terrain{
	{
		Name:       "",
		Symbol:     " ",
		SymbolSort: 1,
		Color:      "black",
		ColorSort:  1,
		Weight:     10,
		BgColor:    "black",
	},
	{
		Name:       " ",
		Symbol:     " ",
		SymbolSort: 1,
		Color:      "black",
		ColorSort:  1,
		Weight:     10,
		BgColor:    "black",
	},
	{
		Name:       "You",
		Symbol:     "@",
		SymbolSort: 1,
		Color:      "#FFaa33",
		ColorSort:  1,
		Weight:     0,
		BgColor:    "#000000",
	},
	{
		Name:       "Unknown",
		Symbol:     "?",
		SymbolSort: 1,
		Color:      "red",
		ColorSort:  1,
		Weight:     4,
		BgColor:    "black",
	},
	{
		Name:       "?",
		Symbol:     "?",
		SymbolSort: 1,
		Color:      "black",
		ColorSort:  1,
		Weight:     4,
	},
	{
		Name:       "Air",
		Symbol:     ".",
		SymbolSort: 8,
		Color:      "#888888",
		ColorSort:  5,
		Weight:     1,
	},
	{
		Name:       "Arctic",
		Symbol:     "_",
		SymbolSort: 8,
		Color:      "#aaaaaa",
		ColorSort:  3,
		Weight:     7,
		BgColor:    "#FFFFFF",
	},
	{
		Name:       "Astral",
		Symbol:     ".",
		SymbolSort: 9,
		Color:      "white",
		ColorSort:  5,
		Weight:     3,
	},
	{
		Name:       "Beach",
		Symbol:     "~",
		SymbolSort: 4,
		Color:      "#FFEA00",
		ColorSort:  3,
		Weight:     4,
		BgColor:    "#00FFFF",
	},
	{
		Name:       "Bridge",
		Symbol:     "=",
		SymbolSort: 1,
		Color:      "bright_yellow",
		ColorSort:  2,
		Weight:     1,
	},
	{
		Name:       "City",
		Symbol:     "+",
		SymbolSort: 10,
		Color:      "white",
		ColorSort:  3,
		Weight:     2,
	},
	{
		Name:       "Deep",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "blue",
		ColorSort:  3,
		Weight:     6,
		BgColor:    "cyan",
	},
	{
		Name:       "Desert",
		Symbol:     ".",
		SymbolSort: 1,
		Color:      "bright_yellow",
		ColorSort:  9,
		Weight:     4,
		BgColor:    "yellow",
	},
	{
		Name:       "Fence",
		Symbol:     "|",
		SymbolSort: 5,
		Color:      "green",
		ColorSort:  3,
		Weight:     3,
	},
	{
		Name:       "Field",
		Symbol:     ".",
		SymbolSort: 7,
		Color:      "#00FF00",
		ColorSort:  4,
		Weight:     3,
		BgColor:    "#00aa00",
	},
	{
		Name:       "Forest",
		Symbol:     "*",
		SymbolSort: 5,
		//Color:      "green",
		Color:     "#228B22",
		ColorSort: 3,
		Weight:    3,
		//BgColor:    "green",
		BgColor: "#00aa00",
	},
	{
		Name:       "ForestJungle",
		Symbol:     "x",
		SymbolSort: 3,
		Color:      "bright_green",
		ColorSort:  7,
		Weight:     4,
	},
	{
		Name:       "Hills",
		Symbol:     ")",
		SymbolSort: 6,
		Color:      "yellow",
		ColorSort:  3,
		Weight:     5,
		BgColor:    "yellow",
	},
	{
		Name:       "Inside",
		Symbol:     "o",
		SymbolSort: 7,
		Color:      "bright_white",
		ColorSort:  5,
		Weight:     1,
	},
	{
		Name:       "Jungle",
		Symbol:     "x",
		SymbolSort: 4,
		Color:      "bright_green",
		ColorSort:  6,
		Weight:     7,
		BgColor:    "green",
	},
	{
		Name:       "Lava",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "bright_red",
		ColorSort:  9,
		Weight:     99,
		Impassable: true,
		BgColor:    "red",
	},
	{
		Name:       "Lush",
		Symbol:     "x",
		SymbolSort: 1,
		Color:      "bright_green",
		ColorSort:  9,
		Weight:     3,
	},
	{
		Name:       "Mountains",
		Symbol:     "^",
		SymbolSort: 7,
		Color:      "white",
		ColorSort:  3,
		Weight:     9,
		BgColor:    "yellow",
	},
	{
		Name:       "Ocean",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "bright_blue",
		ColorSort:  9,
		Weight:     99,
		Impassable: true,
		BgColor:    "blue",
	},
	{
		Name:       "Pasture",
		Symbol:     ".",
		SymbolSort: 4,
		Color:      "green",
		ColorSort:  6,
		Weight:     3,
	},
	{
		Name:       "Path",
		Symbol:     "-",
		SymbolSort: 2,
		Color:      "yellow",
		ColorSort:  10,
		Weight:     1,
	},
	{
		Name:       "Peak",
		Symbol:     "^",
		SymbolSort: 1,
		Color:      "bright_white",
		ColorSort:  1,
		Weight:     99,
		Impassable: true,
		BgColor:    "yellow",
	},
	{
		Name:       "Planar",
		Symbol:     ".",
		SymbolSort: 9,
		Color:      "white",
		ColorSort:  3,
		Weight:     1,
	},
	{
		Name:       "Portal",
		Symbol:     "&",
		SymbolSort: 2,
		Color:      "white",
		ColorSort:  8,
		Weight:     1,
	},
	{
		Name:       "Shallow",
		Symbol:     "~",
		SymbolSort: 2,
		Color:      "bright_cyan",
		ColorSort:  8,
		Weight:     6,
	},
	{
		Name:       "Snow",
		Symbol:     "_",
		SymbolSort: 1,
		Color:      "bright_white",
		ColorSort:  9,
		Weight:     5,
		BgColor:    "white",
	},
	{
		Name:       "Stairs",
		Symbol:     "v",
		SymbolSort: 0,
		Color:      "white",
		ColorSort:  5,
		Weight:     4,
	},
	{
		Name:       "Swamp",
		Symbol:     "~",
		SymbolSort: 4,
		Color:      "black",
		ColorSort:  6,
		Weight:     8,
		BgColor:    "green",
	},
	{
		Name:       "Tundra",
		Symbol:     ".",
		SymbolSort: 4,
		Color:      "bright_white",
		ColorSort:  4,
		Weight:     5,
		BgColor:    "white",
	},
	{
		Name:       "Underground",
		Symbol:     "o",
		SymbolSort: 11,
		Color:      "bright_white",
		ColorSort:  3,
		Weight:     3,
		BgColor:    "black",
	},
	{
		Name:       "Underwater",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "blue",
		ColorSort:  9,
		Weight:     7,
	},
	{
		Name:       "Water",
		Symbol:     "~",
		SymbolSort: 3,
		//Color:      "bright_cyan",
		Color:     "96",
		ColorSort: 3,
		Weight:    6,
		//BgColor:   "cyan",
		BgColor: "36",
	},
}

func GetTerrainByName(name string) *Terrain {
	for _, terrain := range TerrainList {
		if terrain.Name == name {
			return &terrain
		}
	}
	return nil
}

func GetStyleByTerrain(t *Terrain) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Color)).
		Background(lipgloss.Color(t.BgColor))
}

func GetTerrainMapSymbol(terrain string) string {
	terrainWords := strings.Split(terrain, " ")
	lowestSortValue := -1
	symbol := " "
	for _, word := range terrainWords {
		terrain := GetTerrainByName(word)
		if terrain != nil && (lowestSortValue == -1 || terrain.SymbolSort < lowestSortValue) {
			lowestSortValue = terrain.SymbolSort
			symbol = terrain.Symbol
		}
	}
	log.Printf("Terrain: %s, Symbol: %s", terrain, symbol)
	return symbol
}

func GetTerrainMapColor(terrain string) string {
	terrainWords := strings.Split(terrain, " ")
	lowestSortValue := -1
	color := "black"
	for _, word := range terrainWords {
		terrain := GetTerrainByName(word)
		if terrain != nil && (lowestSortValue == -1 || terrain.ColorSort < lowestSortValue) {
			lowestSortValue = terrain.ColorSort
			color = terrain.Color
		}
	}
	log.Printf("Terrain: %s, Color: %s", terrain, color)
	return color
}
