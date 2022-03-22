package store

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4"
	"github.com/podcreep/server/util"
	"golang.org/x/crypto/bcrypt"
)

// Account ...
type Account struct {
	// A unique ID for this account.
	ID           int64
	Cookie       string
	Username     string
	PasswordHash []byte
}

// SaveAccount saves an account to the data store.
func SaveAccount(ctx context.Context, username, password string) (*Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	cookie, err := util.CreateCookie()
	if err != nil {
		return nil, fmt.Errorf("error creating cookie: %w", err)
	}

	sql := "INSERT INTO accounts (cookie, username, password_hash) VALUES($1, $2, $3) RETURNING id"
	row := pool.QueryRow(ctx, sql, cookie, username, hash)

	var id int64
	row.Scan(&id)

	acct := &Account{
		ID:           id,
		Cookie:       cookie,
		Username:     username,
		PasswordHash: hash,
	}
	return acct, nil
}

// SaveSubscription saves a new subscription to the data store.
func SaveSubscription(ctx context.Context, acct *Account, podcastID int64) error {
	sql := "INSERT INTO subscriptions (podcast_id, account_id) VALUES ($1, $2)"
	_, err := pool.Exec(ctx, sql, podcastID, acct.ID)
	return err
}

// DeleteSubscription deletes a subscription for the given podcast.
func DeleteSubscription(ctx context.Context, acct *Account, podcastID int64) error {
	sql := "DELETE FROM subscriptions WHERE account_id=$1 AND podcast_id=$2"
	_, err := pool.Exec(ctx, sql, acct.ID, podcastID)
	return err
}

// GetSubscriptions return the Podcasts that this account is subscribed to.
func GetSubscriptions(ctx context.Context, acct *Account) ([]*Podcast, error) {
	sql := `SELECT
			id, title, description, image_url, image_path, feed_url, last_fetch_time
		FROM podcasts
		  INNER JOIN subscriptions ON podcasts.id = subscriptions.podcast_id
		WHERE subscriptions.account_id = $1`
	rows, _ := pool.Query(ctx, sql, acct.ID)
	defer rows.Close()

	return populatePodcasts(rows)
}

// LoadSubscriptionIDs gets the ID of all the podcasts the given account is subscribed to.
func LoadSubscriptionIDs(ctx context.Context, acct *Account) (map[int64]struct{}, error) {
	sql := "SELECT podcast_id FROM subscriptions WHERE account_id = $1"
	rows, _ := pool.Query(ctx, sql, acct.ID)
	defer rows.Close()

	ids := make(map[int64]struct{})
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Error scanning podcast: %w", err)
		}

		ids[id] = struct{}{}
	}

	return ids, nil
}

// IsSubscribed returns true if the given account is subscribed to the given podcast or not.
func IsSubscribed(ctx context.Context, acct *Account, podcastID int64) bool {
	sql := "SELECT * FROM subscriptions WHERE account_id=$1 AND podcast_id=$2"
	rows, _ := pool.Query(ctx, sql, acct.ID, podcastID)
	defer rows.Close()
	return rows.Next()
}

// VerifyUsernameExists returns true if the given username exists or false if it does not exist.
// An error is returned if there is an error talking to the database.
func VerifyUsernameExists(ctx context.Context, username string) (bool, error) {
	rows, _ := pool.Query(ctx, "SELECT id, username FROM accounts WHERE username=$1", username)
	defer rows.Close()

	return rows.Next(), nil
}

func getAccountFromRow(row pgx.Row) (*Account, error) {
	var acct Account
	if err := row.Scan(&acct.ID, &acct.Username, &acct.Cookie, &acct.PasswordHash); err != nil {
		return nil, fmt.Errorf("Error scanning row: %w", err)
	}

	return &acct, nil
}

// LoadAccountByUsername loads the Account for the user with the given username. Returns nil, nil
// if no account with that username exists.
func LoadAccountByUsername(ctx context.Context, username, password string) (*Account, error) {
	sql := "SELECT id, username, cookie, password_hash FROM accounts WHERE username=$1"
	row := pool.QueryRow(ctx, sql, username)

	acct, err := getAccountFromRow(row)
	if err != nil {
		return nil, err
	}

	// Check that the password matches as well.
	if err := bcrypt.CompareHashAndPassword(acct.PasswordHash, []byte(password)); err != nil {
		log.Printf("Passwords do not match for user %s: %v\n", username, err)
		return nil, nil
	}

	return acct, nil
}

// LoadAccountByCookie loads the Account for the user with the given cookie. Returns an error
// if no account with that cookie exists.
func LoadAccountByCookie(ctx context.Context, cookie string) (*Account, error) {
	sql := "SELECT id, username, cookie, password_hash FROM accounts WHERE cookie=$1"
	row := pool.QueryRow(ctx, sql, cookie)
	return getAccountFromRow(row)
}
