package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"cloud.google.com/go/datastore"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/store"
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
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Printf("Error authenticating: %v\n", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	subs, err := store.GetSubscriptions(ctx, acct)
	if err != nil {
		log.Printf("Error fetching subscriptions: %v\n", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}
	log.Printf("Got %d subscription(s) for %s\n", len(subs), acct.Username)

	// TODO: don't fetch all the podcasts just to filter out a few...
	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error fetching podcasts: %v\n", err)
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
		log.Printf("Error encoding subscriptions: %v\n", err)
		http.Error(w, "Error encoding subscriptions.", http.StatusInternalServerError)
		return
	}
}

// handleSubscriptionsPost handles a POST to /api/podcasts/{id}/subscriptions, and adds a
// subscription to the given podcast for the given user.
func handleSubscriptionsPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Printf("Error authenticating: %v\n", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	s, err := store.SaveSubscription(ctx, acct, podcastID)
	if err != nil {
		log.Printf("Error saving subscription: %v\n", err)
		http.Error(w, "Error saving subscription", http.StatusInternalServerError)
		return
	}

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		log.Printf("Error creating datastore client: %v\n", err)
		http.Error(w, "Error creating datastore client", http.StatusInternalServerError)
		return
	}
	_, err = ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		p, err := store.GetPodcast(ctx, podcastID)
		if err != nil {
			return fmt.Errorf("error loading podcast: %v", err)
		}

		p.Subscribers = append(p.Subscribers, acct.ID, s.ID)
		_, err = store.SavePodcast(ctx, p)
		if err != nil {
			// Ignoring this error.
			_ = store.DeleteSubscription(ctx, acct, s.ID)
			return fmt.Errorf("error saving podcast: %v", err)
		}

		return nil
	})
	if err != nil {
		log.Printf("Error updating podcast: %s\n", err)
		http.Error(w, "Error setting up subscription.", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		log.Printf("Error encoding account: %v\n", err)
		http.Error(w, "Error encoding account.", http.StatusInternalServerError)
		return
	}
}

// handleSubscriptionsDelete handles a DELETE to /api/podcasts/{id}/subscriptions, and removes a
// subscription from the given podcast for the given user.
func handleSubscriptionsDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Printf("Error authenticating: %v\n", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	subscriptionID, err := strconv.ParseInt(vars["sub"], 10, 0)
	if err != nil {
		log.Printf("Error parsing sub ID: %s\n", vars["sub"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err = store.DeleteSubscription(ctx, acct, subscriptionID)
	if err != nil {
		log.Printf("Error deleting subscription: %v\n", err)
		http.Error(w, "Error deleting subscription", http.StatusInternalServerError)
		return
	}

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		log.Printf("Error creating datastore client: %v\n", err)
		http.Error(w, "Error creating datastore client", http.StatusInternalServerError)
		return
	}
	_, err = ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		p, err := store.GetPodcast(ctx, podcastID)
		if err != nil {
			return fmt.Errorf("error loading podcast: %v", err)
		}

		for i := 0; i < len(p.Subscribers); i += 2 {
			if p.Subscribers[i] == acct.ID {
				p.Subscribers = append(p.Subscribers[:i], p.Subscribers[i+2:]...)
				break
			}
		}
		_, err = store.SavePodcast(ctx, p)
		if err != nil {
			return fmt.Errorf("error saving podcast: %v", err)
		}

		return nil
	})
	if err != nil {
		log.Printf("Error updating podcast: %s\n", err)
		http.Error(w, "Error setting up subscription.", http.StatusInternalServerError)
		return
	}

	// TODO?
	//err = json.NewEncoder(w).Encode(p)
	//if err != nil {
	//	log.Printf("Error encoding account: %v\n", err)
	//	http.Error(w, "Error encoding account.", http.StatusInternalServerError)
	//	return
	//}
}