package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/store"
)

const (
	// NewEpisodeDays is the number of days worth of episodes we'll fetch
	NewEpisodeDays = 30
)

// subscription represents a subscription to a podcast. It is a child entity of the account.
type subscription struct {
	// Podcast is the podcast this subscription is for.
	Podcast *store.Podcast `json:"podcast"`

	// PositionsMap is a nicer encoding of Positions for JSON. The key is the episode ID (as a
	// string, because that's what JSON requires), and the value is the offset in seconds that you're
	// up to (again, negative for completely-played episodes).
	Positions map[string]int32 `json:"positions"`
}

type episodeDetails struct {
	store.Episode

	// Position is the progress you've made into this episode.
	Position int32 `json:"position"`
}

// This is returned to the client when it requests the users subscriptions.
type subscriptionDetailsList struct {
	Subscriptions []subscription    `json:"subscriptions"`
	NewEpisodes   []*episodeDetails `json:"newEpisodes"`
	InProgress    []*episodeDetails `json:"inProgress"`
}

type subscriptionsSyncPostRequest struct {
	// TODO: stuff here...
}

type subscriptionsSyncPostResponse struct {
	Subscriptions []subscription `json:"subscriptions"`
}

func getSubscriptions(ctx context.Context, acct *store.Account) ([]subscription, error) {
	subs, err := store.GetSubscriptions(ctx, acct)
	if err != nil {
		return nil, err
	}
	log.Printf("Got %d subscription(s) for %s\n", len(subs), acct.Username)

	podcasts, err := store.GetSubscriptions(ctx, acct)
	if err != nil {
		return nil, err
	}

	var subscriptions []subscription
	for _, podcast := range podcasts {
		sub := subscription{podcast, make(map[string]int32)}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
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

	// Get the subscriptions for this user.
	subscriptionDetails, err := getSubscriptions(ctx, acct)
	if err != nil {
		log.Printf("Error getting subscriptions: %v\n", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}

	// Get the new episodes for this user. We'll grab all episodes from the last 30 days for each
	// podcast they're subscribed to, then intermix them all together.
	var newEpisodes []*episodeDetails
	var inProgress []*episodeDetails
	podcastIDs := make(map[int64]struct{})
	ne, ip, err := store.GetEpisodesNewAndInProgress(ctx, acct, NewEpisodeDays)
	if err != nil {
		log.Printf("Error getting episodes: %v\n", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}

	for _, ep := range ne {
		podcastIDs[ep.PodcastID] = struct{}{}
		newEpisodes = append(newEpisodes, &episodeDetails{
			Episode:  *ep,
			Position: 0,
		})
	}
	for _, ep := range ip {
		podcastIDs[ep.PodcastID] = struct{}{}
		inProgress = append(inProgress, &episodeDetails{
			Episode:  ep.Episode,
			Position: ep.Position,
		})
	}
	sort.Slice(newEpisodes, func(i, j int) bool {
		return newEpisodes[i].PubDate.After(newEpisodes[j].PubDate)
	})

	err = json.NewEncoder(w).Encode(&subscriptionDetailsList{
		Subscriptions: subscriptionDetails,
		NewEpisodes:   newEpisodes,
		InProgress:    inProgress,
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

	if err := store.SaveSubscription(ctx, acct, podcastID); err != nil {
		log.Printf("Error saving subscription: %v\n", err)
		http.Error(w, "Error saving subscription", http.StatusInternalServerError)
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

	err = store.DeleteSubscription(ctx, acct, podcastID)
	if err != nil {
		log.Printf("Error deleting subscription: %v\n", err)
		http.Error(w, "Error deleting subscription", http.StatusInternalServerError)
		return
	}
}

// handleSubscriptionsSync handles a request for /api/subscriptions/sync
func handleSubscriptionsSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req subscriptionsSyncPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error parsing request: %v\n", err)
		http.Error(w, "Error parsing request.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Printf("Error authenticating: %v\n", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	subscriptionDetails, err := getSubscriptions(ctx, acct)
	if err != nil {
		log.Printf("Error getting subscriptions: %v\n", err)
		http.Error(w, "Unexpected error.", http.StatusInternalServerError)
		return
	}

	// For each podcast, grab the episodes that the client doesn't yet have.
	for i, sub := range subscriptionDetails {
		p, err := store.GetPodcast(ctx, sub.Podcast.ID)
		if err != nil {
			log.Printf("Error fetching podcast: %v\n", err)
			http.Error(w, "Error fetching podcast.", http.StatusInternalServerError)
			return
		}

		p.Episodes, err = store.GetEpisodesForSubscription(ctx, acct, p)
		if err != nil {
			log.Printf("Error fetching episodes: %v\n", err)
			http.Error(w, "Error fetching episodes.", http.StatusInternalServerError)
			return
		}

		// TODO: don't return episodes they've already got
		subscriptionDetails[i].Podcast = p
	}
	// TODO: also get the latest positions...

	err = json.NewEncoder(w).Encode(&subscriptionsSyncPostResponse{
		Subscriptions: subscriptionDetails,
	})
	if err != nil {
		log.Printf("Error encoding subscriptions: %v\n", err)
		http.Error(w, "Error encoding subscriptions.", http.StatusInternalServerError)
		return
	}
}
