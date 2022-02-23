package store

import (
	"context"

	"cloud.google.com/go/datastore"
)

type Transaction struct {
	tx *datastore.Transaction
}

func RunInTransaction(ctx context.Context, f func(tx *Transaction) error) error {
	_, err := ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		t := &Transaction{tx}
		return f(t)
	})
	return err
}
