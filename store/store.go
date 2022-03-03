// Package store contains methods and structures that we use to persist our data in the data store.
package store

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4"
)

var (
	conn *pgx.Conn
)

func Setup() error {
	var ctx = context.Background()
	var err error

	dburl := os.Getenv("DATABASE_URL")
	conn, err = pgx.Connect(ctx, dburl)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %s %w", dburl, err)
	}

	// Check what version of the datastore we have, and upgrade it if nessecary.
	version := GetCurrentSchemaVersion(ctx)
	log.Printf("Got schema version %d", version)
	if err := UpgradeSchema(ctx, version); err != nil {
		return err
	}

	return nil
}
