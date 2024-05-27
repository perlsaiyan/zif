package main

import (
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
		Color:      "#000000",
		ColorSort:  1,
		Weight:     10,
		BgColor:    "#000000",
	},
	{
		Name:       " ",
		Symbol:     " ",
		SymbolSort: 1,
		Color:      "#000000",
		ColorSort:  1,
		Weight:     10,
		BgColor:    "#000000",
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
		Color:      "#ff0000",
		ColorSort:  1,
		Weight:     4,
		BgColor:    "#000000",
	},
	{
		Name:       "?",
		Symbol:     "?",
		SymbolSort: 1,
		Color:      "#000000",
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
		BgColor:    "#000000",
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
		Color:      "#FFFFFF",
		ColorSort:  5,
		Weight:     3,
		BgColor:    "#000000",
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
		Color:      "#FFEA00",
		ColorSort:  2,
		Weight:     1,
		BgColor:    "#00FFFF",
	},
	{
		Name:       "City",
		Symbol:     "+",
		SymbolSort: 10,
		Color:      "#FFFFFF",
		ColorSort:  3,
		Weight:     2,
		BgColor:    "#000000",
	},
	{
		Name:       "Deep",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "#0000FF",
		ColorSort:  3,
		Weight:     6,
		BgColor:    "#00FFFF",
	},
	{
		Name:       "Desert",
		Symbol:     ".",
		SymbolSort: 1,
		Color:      "#FFEA00",
		ColorSort:  9,
		Weight:     4,
		BgColor:    "#FFFF00",
	},
	{
		Name:       "Fence",
		Symbol:     "|",
		SymbolSort: 5,
		Color:      "#00FF00",
		ColorSort:  3,
		Weight:     3,
		BgColor:    "#000000",
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
		Color:      "#228B22",
		ColorSort:  3,
		Weight:     3,
		BgColor:    "#00aa00",
	},
	{
		Name:       "ForestJungle",
		Symbol:     "x",
		SymbolSort: 3,
		Color:      "#228B22",
		ColorSort:  7,
		Weight:     4,
		BgColor:    "#00aa00",
	},
	{
		Name:       "Hills",
		Symbol:     ")",
		SymbolSort: 6,
		Color:      "#FFFF00",
		ColorSort:  3,
		Weight:     5,
		BgColor:    "#66ff00",
	},
	{
		Name:       "Inside",
		Symbol:     "o",
		SymbolSort: 7,
		Color:      "#FFFFFF",
		ColorSort:  5,
		Weight:     1,
		BgColor:    "#000000",
	},
	{
		Name:       "Jungle",
		Symbol:     "x",
		SymbolSort: 4,
		Color:      "#00ff00",
		ColorSort:  6,
		Weight:     7,
		BgColor:    "#228b22",
	},
	{
		Name:       "Lava",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "#ff0000",
		ColorSort:  9,
		Weight:     99,
		Impassable: true,
		BgColor:    "#660000",
	},
	{
		Name:       "Lush",
		Symbol:     "x",
		SymbolSort: 1,
		Color:      "#33ff33",
		ColorSort:  9,
		Weight:     3,
		BgColor:    "#228b22",
	},
	{
		Name:       "Mountains",
		Symbol:     "^",
		SymbolSort: 7,
		Color:      "#aaaaaa",
		ColorSort:  3,
		Weight:     9,
		BgColor:    "#000000",
	},
	{
		Name:       "Ocean",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "#00FFFF",
		ColorSort:  9,
		Weight:     99,
		Impassable: true,
		BgColor:    "#0000FF",
	},
	{
		Name:       "Pasture",
		Symbol:     ".",
		SymbolSort: 4,
		Color:      "#00FF00",
		ColorSort:  6,
		Weight:     3,
	},
	{
		Name:       "Path",
		Symbol:     "-",
		SymbolSort: 2,
		Color:      "#FFEA00",
		ColorSort:  10,
		Weight:     1,
	},
	{
		Name:       "Peak",
		Symbol:     "^",
		SymbolSort: 1,
		Color:      "#ffffff",
		ColorSort:  1,
		Weight:     99,
		Impassable: true,
		BgColor:    "#ffea00",
	},
	{
		Name:       "Planar",
		Symbol:     ".",
		SymbolSort: 9,
		Color:      "#aaaaaa",
		ColorSort:  3,
		Weight:     1,
		BgColor:    "#000000",
	},
	{
		Name:       "Portal",
		Symbol:     "&",
		SymbolSort: 2,
		Color:      "#aaaaaa",
		ColorSort:  8,
		Weight:     1,
		BgColor:    "#000000",
	},
	{
		Name:       "Shallow",
		Symbol:     "~",
		SymbolSort: 2,
		Color:      "#00ffff",
		ColorSort:  8,
		Weight:     6,
		BgColor:    "#0000ff",
	},
	{
		Name:       "Snow",
		Symbol:     "_",
		SymbolSort: 1,
		Color:      "#ffffff",
		ColorSort:  9,
		Weight:     5,
		BgColor:    "#aaaaaa",
	},
	{
		Name:       "Stairs",
		Symbol:     "v",
		SymbolSort: 0,
		Color:      "#aaaaaa",
		ColorSort:  5,
		Weight:     4,
		BgColor:    "#000000",
	},
	{
		Name:       "Swamp",
		Symbol:     "~",
		SymbolSort: 4,
		Color:      "#000000",
		ColorSort:  6,
		Weight:     8,
		BgColor:    "#00aa00",
	},
	{
		Name:       "Tundra",
		Symbol:     ".",
		SymbolSort: 4,
		Color:      "#ffffff",
		ColorSort:  4,
		Weight:     5,
		BgColor:    "#aaaaaa",
	},
	{
		Name:       "Underground",
		Symbol:     "o",
		SymbolSort: 11,
		Color:      "#ffffff",
		ColorSort:  3,
		Weight:     3,
		BgColor:    "#000000",
	},
	{
		Name:       "Underwater",
		Symbol:     "~",
		SymbolSort: 1,
		Color:      "#0000aa",
		ColorSort:  9,
		Weight:     7,
		BgColor:    "#0000ff",
	},
	{
		Name:       "Water",
		Symbol:     "~",
		SymbolSort: 3,
		//Color:      "bright_cyan",
		Color:     "#00ffff",
		ColorSort: 3,
		Weight:    6,
		//BgColor:   "cyan",
		BgColor: "#00aaaa",
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
	//log.Printf("Terrain: %s, Symbol: %s", terrain, symbol)
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
	//log.Printf("Terrain: %s, Color: %s", terrain, color)
	return color
}
