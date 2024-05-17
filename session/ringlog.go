package session

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type RingLog struct {
	Db *sql.DB
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
	//log.Printf("Inserting row %d at %d", id, ts)

	tx, err := s.Ringlog.Db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert or replace into ring_log(ring_number, epoch_ns, message, stripped) values(?, ?,?,?)")
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
	stmt, err := r.Db.Prepare("select ifnull(max(ring_number), 0) from ring_log where epoch_ns = (select max(epoch_ns) from ring_log)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var id int
	err = stmt.QueryRow().Scan(&id)
	if err != nil {
		log.Fatal(err)
	}

	return id
}
