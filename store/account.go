package store

import (
	"context"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// Account ...
type Account struct {
	// A unique ID for this account.
	ID int64 `datastore:"-"`

	Cookie       string
	Username     string
	PasswordHash []byte
}

func createCookie() (string, error) {
	var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	runes := make([]rune, 20)
	for i := range bytes {
		runes[i] = alphabet[int(bytes[i])%len(alphabet)]
	}
	return string(runes), nil
}

// SaveAccount saves an account to the data store.
func SaveAccount(ctx context.Context, username, password string) (*Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}

	cookie, err := createCookie()
	if err != nil {
		return nil, fmt.Errorf("error creating cookie: %v", err)
	}

	acct := &Account{
		Cookie:       cookie,
		Username:     username,
		PasswordHash: hash,
	}
	key := datastore.NewKey(ctx, "account", "", 0, nil)
	key, err = datastore.Put(ctx, key, acct)
	if err != nil {
		return nil, fmt.Errorf("error storing account: %v", err)
	}
	return acct, nil
}

// LoadAccountByUsername loads the Account for the user with the given username. Returns nil, nil
// if no account with that username exists.
func LoadAccountByUsername(ctx context.Context, username, password string) (*Account, error) {
	q := datastore.NewQuery("account").
		Filter("Username =", username).
		Limit(1)
	for row := q.Run(ctx); ; {
		var acct Account
		key, err := row.Next(&acct)
		if err != nil {
			if err == datastore.Done {
				log.Warningf(ctx, "User does not exist %s", username)
				return nil, nil
			}
			return nil, err
		}

		// Check that the password matches as well.
		if err := bcrypt.CompareHashAndPassword(acct.PasswordHash, []byte(password)); err != nil {
			log.Warningf(ctx, "Passwords do not match for user %s: %v", username, err)
			return nil, nil
		}

		acct.ID = key.IntID()
		return &acct, nil
	}
}

// LoadAccountByCookie loads the Account for the user with the given cookie. Returns an error
// if no account with that cookie exists.
func LoadAccountByCookie(ctx context.Context, cookie string) (*Account, error) {
	q := datastore.NewQuery("account").
		Filter("Cookie =", cookie).
		Limit(1)
	for row := q.Run(ctx); ; {
		var acct Account
		key, err := row.Next(&acct)
		if err != nil {
			if err == datastore.Done {
				return nil, fmt.Errorf("user with cookie '%s' not found", cookie)
			}
			return nil, err
		}

		acct.ID = key.IntID()
		return &acct, nil
	}
}
