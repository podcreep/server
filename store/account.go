package store

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"

	"github.com/podcreep/server/util"
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

	// DoneCutoffDate is the date after which we assume all episodes are "done". When you mark an
	// episode done, if there's nothing else after that episode before this date, we'll simply adjust
	// the cutoff date to include the new episode. That way, we can keep the episode position list
	// relatively short.
	DoneCutoffDate int64 `json:"doneCutoffDate"`

	// Positions is an array of episodeID,offset integer. The first integer is the identifier of the
	// episide that is being played. The second integer is the offset (in seconds) that playback is
	// up to for the given user. If the second integer is negative, then the episode has been fully
	// played.
	Positions []int64 `json:"-"`

	// PositionsMap is a nicer encoding of Positions for JSON. The key is the episode ID (as a
	// string, because that's what JSON requires), and the value is the offset in seconds that you're
	// up to (again, negative for completely-played episodes).
	PositionsMap map[string]int32 `datastore:"-" json:"positions"`

	// ignored, do not use.
	Ignored int64 `datastore:"OldestUnlistenedEpisodeID"`
}

// SaveAccount saves an account to the data store.
func SaveAccount(ctx context.Context, username, password string) (*Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}

	cookie, err := util.CreateCookie()
	if err != nil {
		return nil, fmt.Errorf("error creating cookie: %v", err)
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

	key, err := ds.Put(ctx, key, sub)
	if err != nil {
		return nil, fmt.Errorf("error storing subscription: %v", err)
	}

	sub.ID = key.ID
	return sub, nil
}

// DeleteSubscription deletes a subscription for the given podcast.
func DeleteSubscription(ctx context.Context, acct *Account, subscriptionID int64) error {
	acctKey := datastore.IDKey("account", acct.ID, nil)
	key := datastore.IDKey("subscription", subscriptionID, acctKey)
	return ds.Delete(ctx, key)
}

func populateSubscription(sub *Subscription) {
	sub.PositionsMap = make(map[string]int32)
	for i := 0; i < len(sub.Positions); i += 2 {
		s := strconv.FormatInt(sub.Positions[i], 10)
		sub.PositionsMap[s] = int32(sub.Positions[i+1])
	}
}

// GetSubscriptions return all of the subscriptions owned by the given account.
func GetSubscriptions(ctx context.Context, acct *Account) ([]*Subscription, error) {
	var subscriptions []*Subscription

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
		populateSubscription(&subscription)

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

// GetSubscription returns the single subscription with the given ID from the given acccount.
func GetSubscription(ctx context.Context, acct *Account, subID int64) (*Subscription, error) {
	acctKey := datastore.IDKey("account", acct.ID, nil)
	subKey := datastore.IDKey("subscription", subID, acctKey)

	sub := new(Subscription)
	err := ds.Get(ctx, subKey, sub)
	if err != nil {
		return nil, fmt.Errorf("error loading subscription: %v", err)
	}

	log.Printf("fetched a subscription: %v", sub)
	populateSubscription(sub)
	return sub, nil
}

// LoadAccountByUsername loads the Account for the user with the given username. Returns nil, nil
// if no account with that username exists.
func LoadAccountByUsername(ctx context.Context, username, password string) (*Account, error) {
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
