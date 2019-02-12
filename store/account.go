package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"strconv"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// Account ...
type Account struct {
	// A unique ID for this account.
	ID int64 `datastore:"-"`

	Cookie       string
	Username     string
	PasswordHash []byte
}

// Subscription represents a subscription to a podcast. It is a child entity of the account.
type Subscription struct {
	ID int64 `datastore:"-" json:"id"`

	// PodcastID is the identified of the podcast that you're subscribed to.
	PodcastID int64 `json:"podcastID"`

	// OldestUnlistenedEpisodeID is the oldest episode ID that hasn't been listened to. This allows us
	// to keep Positions short, by truncating it to only include episodes after this one. An episode
	// can be explicitly in Positions even if it's older than this.
	OldestUnlistenedEpisodeID int64 `json:"oldestUnlistenedEpisodeID"`

	// Positions is an array of episodeID,offset integer. The first integer is the identifier of the
	// episide that is being played. The second integer is the offset (in seconds) that playback is
	// up to for the given user. If the second integer is negative, then the episode has been fully
	// played.
	Positions []int64 `json:"-"`

	// JSONPositions is a nicer encoding of Positions for JSON. The key is the episode ID (as a
	// string, because that's what JSON requires), and the value is the offset in seconds that you're
	// up to (again, negative for completely-played episodes).
	JSONPositions map[string]int32 `datastore:"-" json:"positions"`
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

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	acct := &Account{
		Cookie:       cookie,
		Username:     username,
		PasswordHash: hash,
	}
	key := datastore.IDKey("account", 0, nil)
	key, err = ds.Put(ctx, key, acct)
	if err != nil {
		return nil, fmt.Errorf("error storing account: %v", err)
	}

	acct.ID = key.ID
	return acct, nil
}

// SaveSubscription saves a new subscription to the data store.
func SaveSubscription(ctx context.Context, acct *Account, sub *Subscription) (*Subscription, error) {
	acctKey := datastore.IDKey("account", acct.ID, nil)
	key := datastore.IDKey("subscription", sub.ID, acctKey)

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	key, err = ds.Put(ctx, key, sub)
	if err != nil {
		return nil, fmt.Errorf("error storing subscription: %v", err)
	}

	sub.ID = key.ID
	return sub, nil
}

// DeleteSubscription deletes a subscription for the given podcast.
func DeleteSubscription(ctx context.Context, acct *Account, subscriptionID int64) error {
	acctKey := datastore.IDKey("account", acct.ID, nil)

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return err
	}

	key := datastore.IDKey("subscription", subscriptionID, acctKey)
	return ds.Delete(ctx, key)
}

// GetSubscriptions return all of the subscriptions owned by the given account.
func GetSubscriptions(ctx context.Context, acct *Account) ([]*Subscription, error) {
	var subscriptions []*Subscription

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	acctKey := datastore.IDKey("account", acct.ID, nil)
	q := datastore.NewQuery("subscription").Ancestor(acctKey)
	for row := ds.Run(ctx, q); ; {
		var subscription Subscription
		key, err := row.Next(&subscription)
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, err
		}

		subscription.ID = key.ID
		subscription.JSONPositions = make(map[string]int32)
		for i := 0; i < len(subscription.Positions); i += 2 {
			s := strconv.FormatInt(subscription.Positions[i], 10)
			if err == nil {
				subscription.JSONPositions[s] = int32(subscription.Positions[i+1])
			}
		}

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

// LoadAccountByUsername loads the Account for the user with the given username. Returns nil, nil
// if no account with that username exists.
func LoadAccountByUsername(ctx context.Context, username, password string) (*Account, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery("account").
		Filter("Username =", username).
		Limit(1)
	for row := ds.Run(ctx, q); ; {
		var acct Account
		key, err := row.Next(&acct)
		if err != nil {
			if err == iterator.Done {
				log.Printf("User does not exist %s\n", username)
				return nil, nil
			}
			return nil, err
		}

		// Check that the password matches as well.
		if err := bcrypt.CompareHashAndPassword(acct.PasswordHash, []byte(password)); err != nil {
			log.Printf("Passwords do not match for user %s: %v\n", username, err)
			return nil, nil
		}

		acct.ID = key.ID
		return &acct, nil
	}
}

// LoadAccountByCookie loads the Account for the user with the given cookie. Returns an error
// if no account with that cookie exists.
func LoadAccountByCookie(ctx context.Context, cookie string) (*Account, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery("account").
		Filter("Cookie =", cookie).
		Limit(1)
	for row := ds.Run(ctx, q); ; {
		var acct Account
		key, err := row.Next(&acct)

		if err != nil {
			if err == iterator.Done {
				return nil, fmt.Errorf("user with cookie '%s' not found", cookie)
			}
			return nil, err
		}

		acct.ID = key.ID
		return &acct, nil
	}
}
