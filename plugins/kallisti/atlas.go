package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/perlsaiyan/zif/session"
)

type AtlasRoomRecord struct {
	VNUM          string  `db:"vnum"`
	Name          string  `db:"name"`
	Terrain       string  `db:"terrain_name"`
	AreaName      string  `db:"area_name"`
	RegenHP       bool    `db:"regen_hp"`
	RegenMP       bool    `db:"regen_mp"`
	RegenSP       bool    `db:"regen_sp"`
	SetRecall     bool    `db:"set_recall"`
	Peaceful      bool    `db:"peaceful"`
	Deathtrap     bool    `db:"deathtrap"`
	Silent        bool    `db:"silent"`
	WildMagic     bool    `db:"wild_magic"`
	Bank          bool    `db:"bank"`
	Narrow        bool    `db:"narrow"`
	NoMagic       bool    `db:"no_magic"`
	NoRecall      bool    `db:"no_recall"`
	LastVisited   *string `db:"last_visited"`
	LastHarvested *string `db:"last_harvested"`
}

func ConnectAtlasDB() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", "./db/world.db")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func GetRoomByVNUM(s *session.Session, vnum string) (*AtlasRoomRecord, error) {
	d := s.Data["kallisti"].(*KallistiData)
	query := "SELECT * FROM rooms WHERE vnum = ?"
	var room AtlasRoomRecord
	err := d.Atlas.Get(&room, query, vnum)
	if err != nil {
		return nil, err
	}

	return &room, nil
}

func CmdRoom(s *session.Session, args string) {
	if len(args) < 1 {
		s.Output("Usage: room <vnum>\n")
		return
	}

	room, err := GetRoomByVNUM(s, args)
	if err != nil {
		log.Printf("Error in GetRoomByVNUM: %v", err)
		s.Output("Error... \n")
		return
	}

	msg := fmt.Sprintf("Room: %+v\n", room)
	s.Output(msg)
}
