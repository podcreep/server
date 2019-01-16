package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/store"
)

// authenticate checks that the given request includes an Authorization header and returns the
// account assosicated with the cookie if it does, or an error if it does not.
func authenticate(ctx context.Context, r *http.Request) (*store.Account, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("no Authorization header")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, fmt.Errorf("Authorization header is not Bearer header")
	}
	auth = auth[7:]

	return store.LoadAccountByCookie(ctx, auth)
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/api/accounts", handleAccountsGet).Methods("GET")
	r.HandleFunc("/api/accounts", handleAccountsPost).Methods("POST")
	r.HandleFunc("/api/accounts/login", handleAccountsLoginPost).Methods("POST")
	r.HandleFunc("/api/podcasts", handlePodcastsGet).Methods("GET")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}", handlePodcastGet).Methods("GET")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}/subscriptions", handleSubscriptionsPost).Methods("POST")
	r.HandleFunc("/api/subscriptions", handleSubscriptionsGet).Methods("GET")

	return nil
}
