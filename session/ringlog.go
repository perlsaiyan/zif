package session

import (
	"database/sql"
	"log"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type RingLog struct {
	Db            *sql.DB
	CurrentNumber int
}

type RingRecord struct {
	RingNumber int
	EpochNS    int64
	Context    string
	Message    string
	Stripped   string
}

func NewRingLog() RingLog {

	db, err := sql.Open("sqlite3", ":memory:")

	if err != nil {
		panic(err)
	}

	sqlStmt := `
	PRAGMA journal_mode=WAL;
	create table if not exists ring_log(ring_number not null primary key, epoch_ns, context, message, stripped);
	create index if not exists ring_log_n1 on ring_log(epoch_ns);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		panic(err)
	}

	return RingLog{Db: db}
}

func (s Session) AddRinglogEntry(ts int64, line string, stripped string) {

	// mod 10k so we ring the log
	// TODO: we could make this adjustable
	id := (s.Ringlog.GetCurrentRingNumber() + 1) % 10000
	s.Ringlog.CurrentNumber = id

	tx, err := s.Ringlog.Db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert or replace into ring_log(ring_number, epoch_ns, message, stripped) values(?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, ts, line, stripped)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

}

func (r RingLog) GetCurrentRingNumber() int {
	if r.CurrentNumber == 0 {
		stmt, err := r.Db.Prepare("select ring_number from ring_log order by epoch_ns desc limit 1")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		var id int
		err = stmt.QueryRow().Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				// no rows, return 0
				return 0
			}
		}

		return id
	}
	return r.CurrentNumber
}

func (r RingLog) GetRingEntry(id int) *RingRecord {
	stmt, err := r.Db.Prepare("select ring_number, epoch_ns, message, stripped from ring_log where ring_number = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var record RingRecord
	err = stmt.QueryRow(id).Scan(&record.RingNumber, &record.EpochNS, &record.Message, &record.Stripped)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Fatal(err)
	}

	return &record
}

func CmdRingtest(s *Session, cmd string) {
	id, err := strconv.Atoi(cmd)
	if err != nil {
		s.Output("Invalid ring number\n")
		return
	}

	stmt, err := s.Ringlog.Db.Prepare("select stripped from ring_log where ring_number = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var line string
	err = stmt.QueryRow(id).Scan(&line)
	if err != nil {
		s.Output("No record found\n")
		return
	}

	s.Output("Record: " + line + "\n")
}
