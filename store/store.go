// Package store contains methods and structures that we use to persist our data in the data store.
package store

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

var (
	db      *sql.DB
	datadir string
)

// Both sql.Row and sql.Rows implement this method, so we can use this interface to write code that works with both of them.
type Scannable interface {
	Scan(dest ...any) error
}

func Setup() error {
	var ctx = context.Background()

	dbfile := os.Getenv("DATABASE_FILE")
	dsn := "file:///" + dbfile + "?_pragma=foreign_keys(1)&_time_format=sqlite"
	log.Printf("dsn=%s", dsn)
	if newdb, err := sql.Open("sqlite", dsn); err != nil {
		return err
	} else {
		db = newdb
	}

	// Check what version of the datastore we have, and upgrade it if nessecary.
	version := GetCurrentSchemaVersion(ctx)
	log.Printf("Got schema version %d", version)
	if err := UpgradeSchema(ctx, version); err != nil {
		return err
	}

	return nil
}
