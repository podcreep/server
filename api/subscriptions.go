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

// subscriptionDetails contains some additional details we'll include in the subscriptions we return
// to the client (for example, we'll include the details of the podcast itself).
type subscriptionDetails struct {
	store.Subscription

	Podcast *store.Podcast `json:"podcast"`
}

type subscriptionDetailsList struct {
	Subscriptions []subscriptionDetails `json:"subscriptions"`
}

// handleSubscriptionsGet handles a GET request for /api/subscriptions, and returns all of the
// user's subscriptions.
func handleSubscriptionsGet(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Errorf(ctx, "Error authenticating: %v", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	subs, err := store.GetSubscriptions(ctx, acct)
	if err != nil {
		log.Errorf(ctx, "Error fetching subscriptions: %v", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}
	log.Infof(ctx, "Got %d subscription(s) for %s", len(subs), acct.Username)

	// TODO: don't fetch all the podcasts just to filter out a few...
	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Errorf(ctx, "Error fetching podcasts: %v", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}

	var details []subscriptionDetails
	for _, s := range subs {
		d := subscriptionDetails{*s, nil}
		for _, p := range podcasts {
			if p.ID == d.Subscription.PodcastID {
				d.Podcast = p
			}
		}
		details = append(details, d)
	}

	err = json.NewEncoder(w).Encode(&subscriptionDetailsList{
		Subscriptions: details,
	})
	if err != nil {
		log.Errorf(ctx, "Error encoding subscriptions: %v", err)
		http.Error(w, "Error encoding subscriptions.", http.StatusInternalServerError)
		return
	}
}

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
