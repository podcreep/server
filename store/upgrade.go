package store

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// GetCurrentSchemaVersion gets the current version of the database schema. A completely fresh
// database will have version of 0.
func GetCurrentSchemaVersion(ctx context.Context) int {
	row := conn.QueryRow(ctx, "SELECT version FROM schema_version")
	var version int
	if err := row.Scan(&version); err != nil {
		// The error could be anything, but we'll assume it's just that the table doesn't exist. That
		// is, we need to start from scratch and re-create everything.
		return 0
	}
	return version
}

// UpgradeSchema upgrades the current database schema to the latest version, starting from the
// given current version.
func UpgradeSchema(ctx context.Context, currentVersion int) error {
	for {
		script, err := loadSchemaUpgradeScript(ctx, currentVersion+1)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Schema up-to-date at version %d", currentVersion)

				_, err := conn.Exec(ctx, "UPDATE schema_version SET version=$1", currentVersion)
				return err
			}
			return fmt.Errorf("Error loading schema script: %w", err)
		}

		_, err = conn.Exec(ctx, script)
		if err != nil {
			return fmt.Errorf("Error executing command: %w", err)
		}

		currentVersion += 1
	}
}

// loadSchemaUpgradeScript loads the script to upgrade us to the given version. If the script does
// not exist, returns an error that you can check for with os.IsNotExist -- any other errors are
// real.
func loadSchemaUpgradeScript(ctx context.Context, version int) (string, error) {
	filename := fmt.Sprintf("store/schema/schema-%03d.sql", version)
	f, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			// We'll return the error anyway, but if it's something other than a not exists error, then
			// it's probably bad...
			log.Printf("Got something other than NotExist: %s", err.Error())
		}
		return "", err
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	log.Printf("Loaded upgrade script %s", filename)
	return string(bytes), nil
}
