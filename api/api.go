package api

import (
	"context"
	"fmt"
	"log"
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

type apierr struct {
	Err     error
	Message string
	Code    int
}

func (err apierr) Error() string {
	return fmt.Sprintf("%v [%s] %d", err.Err, err.Message, err.Code)
}

func apiError(msg string, code int) apierr {
	return apierr{nil, msg, code}
}

type wrappedRequest func(http.ResponseWriter, *http.Request) error

func wrap(fn wrappedRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			requestErr, ok := err.(apierr)
			if !ok {
				requestErr = apierr{err, err.Error(), 500}
			}
			log.Printf("Error in request %s: %v", r.URL, requestErr.Error())

			// TODO: hide errors from clients?
			http.Error(w, requestErr.Message, requestErr.Code)
		}
	}
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/api/accounts", wrap(handleAccountsGet)).Methods("GET")
	r.HandleFunc("/api/accounts", wrap(handleAccountsPost)).Methods("POST")
	r.HandleFunc("/api/accounts/login", wrap(handleAccountsLoginPost)).Methods("POST")
	r.HandleFunc("/api/podcasts", wrap(handlePodcastsGet)).Methods("GET")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}", wrap(handlePodcastGet)).Methods("GET")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}", wrap(handleSubscriptionsDelete)).Methods("DELETE")
	r.HandleFunc("/blobs/podcasts/{id:[0-9]+}/icon/{sha1:.+}.png", wrap(handlePodcastIconGet)).Methods("GET")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}/subscriptions", wrap(handleSubscriptionsPost)).Methods("POST")
	r.HandleFunc("/api/podcasts/{id:[0-9]+}/episodes/{ep:[0-9]+}/playback-state", wrap(handlePlaybackStatePut)).Methods("PUT")
	r.HandleFunc("/api/subscriptions", wrap(handleSubscriptionsGet)).Methods("GET")
	r.HandleFunc("/api/subscriptions/sync", wrap(handleSubscriptionsSync)).Methods("POST")
	r.HandleFunc("/api/last-played", wrap(handleLastPlayedGet)).Methods("GET")

	return nil
}
