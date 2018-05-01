package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"google.golang.org/appengine/datastore"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/store"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

// handleSubscriptionsPost handles a POST to /api/podcasts/{id}/subscriptions, and adds a
// subscription to the given podcast for the given user.
func handleSubscriptionsPost(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Errorf(ctx, "Error authenticating: %v", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Errorf(ctx, "Error parsing ID: %s", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err = datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		p, err := store.GetPodcast(ctx, podcastID)
		if err != nil {
			return fmt.Errorf("error loading podcast: %v", err)
		}

		p.Subscribers = append(p.Subscribers, acct.ID)
		_, err = store.SavePodcast(ctx, p)
		if err != nil {
			return fmt.Errorf("error saving podcast: %v", err)
		}

		return nil
	}, nil)
	if err != nil {
		log.Errorf(ctx, "Error updating podcast: %s", err)
		http.Error(w, "Error setting up subscription.", http.StatusInternalServerError)
		return
	}

	s, err := store.SaveSubscription(ctx, acct, podcastID)
	if err != nil {
		// TODO: remove the subscription from the podcast
		log.Errorf(ctx, "Error saving subscription (TODO: remove subscription from podcast): %v", err)
		http.Error(w, "Error saving subscription", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		log.Errorf(ctx, "Error encoding account: %v", err)
		http.Error(w, "Error encoding account.", http.StatusInternalServerError)
		return
	}
}
