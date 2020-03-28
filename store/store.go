// Package store contains methods and structures that we use to persist our data in the data store.
package store

import (
	"context"

	"cloud.google.com/go/datastore"
)

var (
	ds *datastore.Client
)

func init() {
	var err error
	ds, err = datastore.NewClient(context.Background(), "")
	if err != nil {
		panic(err)
	}
}
