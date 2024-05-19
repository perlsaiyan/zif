package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/perlsaiyan/zif/session"
)

type AtlasRoomRecord struct {
	VNUM     string `db:"vnum"`
	Name     string `db:"name"`
	Terrain  string `db:"terrain_name"`
	AreaName string `db:"area_name"`
	RegenHP  bool   `db:"regen_hp"`
	//,regen_mp,regen_sp,set_recall,peaceful,deathtrap,silent,wild_magic,bank,narrow,no_magic,no_recall, last_visited, last_harvested
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
	query := "SELECT vnum, name, terrain_name, area_name, regen_hp FROM rooms WHERE vnum = ?"
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
