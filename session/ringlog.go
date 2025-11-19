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

func (r RingLog) GetLog(start int, end int) []RingRecord {
	var records []RingRecord

	// Handle wrapping if start > end (ring buffer)
	// But for now let's assume simple case or handle the query logic
	// Since it's a ring buffer 0-9999, if start > end, it means we wrapped around.

	var query string
	var rows *sql.Rows
	var err error

	if start <= end {
		query = "select ring_number, epoch_ns, message, stripped from ring_log where ring_number >= ? and ring_number <= ? order by ring_number asc"
		rows, err = r.Db.Query(query, start, end)
	} else {
		// Wrapped around
		// Get from start to 9999
		// Get from 0 to end
		// Actually we can just use OR
		query = "select ring_number, epoch_ns, message, stripped from ring_log where ring_number >= ? OR ring_number <= ? order by case when ring_number >= ? then 0 else 1 end, ring_number asc"
		rows, err = r.Db.Query(query, start, end, start)
	}

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var record RingRecord
		err = rows.Scan(&record.RingNumber, &record.EpochNS, &record.Message, &record.Stripped)
		if err != nil {
			log.Fatal(err)
		}
		records = append(records, record)
	}
	return records
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
