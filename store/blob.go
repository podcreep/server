package store

import (
	"fmt"
	"os"
	"path"
)

// GetBlobStorePath gets the path where we store blobs with the given name.
func GetBlobStorePath(name string) (string, error) {
	basePath := os.Getenv("BLOB_STORE_PATH")
	path := path.Join(basePath, name)
	if _, err := os.Stat(path); err != nil {
		// Try to create it (the error could be anything, but let's see if just making the directory
		// is enough to fix it).
		err := os.MkdirAll(path, os.ModeDir)
		if err != nil {
			return "", fmt.Errorf("Could not create blob store path: %w", err)
		}
	}

	return path, nil
}
