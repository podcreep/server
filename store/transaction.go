package store

import (
	"context"

	"cloud.google.com/go/datastore"
)

type Transaction struct {
	tx *datastore.Transaction
}

func RunInTransaction(ctx context.Context, f func(tx *Transaction) error) error {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return err
	}

	_, err = ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		t := &Transaction{tx}
		return f(t)
	})
	return err
}
