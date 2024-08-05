package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/admin"
	"github.com/podcreep/server/cron"
	"github.com/podcreep/server/discover"
	"github.com/podcreep/server/rss"
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
}

type episodeDetails struct {
	store.Episode
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

type subscribeDiscoveredRequest struct {
	DiscoveryID string `json:"discoveryId"`
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
		sub := subscription{podcast}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// handleSubscriptionsGet handles a GET request for /api/subscriptions, and returns all of the
// user's subscriptions.
func handleSubscriptionsGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	// Get the subscriptions for this user.
	subscriptionDetails, err := getSubscriptions(ctx, acct)
	if err != nil {
		return err
	}

	// Get the new episodes for this user. We'll grab all episodes from the last 30 days for each
	// podcast they're subscribed to, then intermix them all together.
	var newEpisodes []*episodeDetails
	var inProgress []*episodeDetails
	podcastIDs := make(map[int64]struct{})
	ne, ip, err := store.LoadEpisodesNewAndInProgress(ctx, acct, NewEpisodeDays)
	if err != nil {
		return err
	}

	for _, ep := range ne {
		podcastIDs[ep.PodcastID] = struct{}{}
		newEpisodes = append(newEpisodes, &episodeDetails{
			Episode: *ep,
		})
	}
	for _, ep := range ip {
		podcastIDs[ep.PodcastID] = struct{}{}
		inProgress = append(inProgress, &episodeDetails{
			Episode: *ep,
		})
	}
	sort.Slice(newEpisodes, func(i, j int) bool {
		return newEpisodes[i].PubDate.After(newEpisodes[j].PubDate)
	})
	// Sort in progress episodes by the last time you listened it, so most recently-listened first.
	sort.Slice(inProgress, func(i, j int) bool {
		return inProgress[i].LastListenTime.After(*inProgress[j].LastListenTime)
	})

	return json.NewEncoder(w).Encode(&subscriptionDetailsList{
		Subscriptions: subscriptionDetails,
		NewEpisodes:   newEpisodes,
		InProgress:    inProgress,
	})
}

// handleSubscriptionsPost handles a POST to /api/podcasts/{id}/subscriptions, and adds a
// subscription to the given podcast for the given user.
func handleSubscriptionsPost(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized", http.StatusUnauthorized)
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	return store.SaveSubscription(ctx, acct, podcastID)
}

// handleSubscribeDiscoveredPost handles a POST to /api/podcasts/subscribeDiscovered, and adds a
// subscription to the discovered podcast. If we do not yet track that podcast, we'll start tracking it.
func handleSubscribeDiscoveredPost(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req subscribeDiscoveredRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized", http.StatusUnauthorized)
	}

	podcast, err := store.LoadPodcastByDiscoverId(ctx, req.DiscoveryID)
	if err != nil || podcast == nil {
		// TODO: don't just assume any error is 'not found'. The problem is, row.Scan() doesn't seem to return ErrNoRows
		// as it's documented to do.

		discoverId, err := strconv.Atoi(req.DiscoveryID)
		if err != nil {
			return apiError("discover ID not an int", http.StatusInternalServerError)
		}
		discoverPodcast, _, err := discover.FetchPodcast(int64(discoverId) /*includeEpisodes*/, true)
		if err != nil {
			return err
		}

		id, err := admin.CreatePodcastFromUrl(ctx, discoverPodcast.Url)
		if err != nil {
			return err
		}

		podcast, err = store.LoadPodcast(ctx, id)
		if err != nil {
			return err
		}
		_, err = cron.UpdatePodcast(ctx, podcast, rss.ForceUpdate)
		if err != nil {
			return nil
		}
	}

	return store.SaveSubscription(ctx, acct, podcast.ID)
}

// handleSubscriptionsDelete handles a DELETE to /api/podcasts/{id}/subscriptions, and removes a
// subscription from the given podcast for the given user.
func handleSubscriptionsDelete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized", http.StatusUnauthorized)
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	return store.DeleteSubscription(ctx, acct, podcastID)
}

// handleSubscriptionsSync handles a request for /api/subscriptions/sync
func handleSubscriptionsSync(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var req subscriptionsSyncPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return err
	}

	subscriptionDetails, err := getSubscriptions(ctx, acct)
	if err != nil {
		return err
	}

	// For each podcast, grab the episodes that the client doesn't yet have.
	for i, sub := range subscriptionDetails {
		p, err := store.LoadPodcast(ctx, sub.Podcast.ID)
		if err != nil {
			return err
		}

		p.Episodes, err = store.LoadEpisodesForSubscription(ctx, acct, p)
		if err != nil {
			return err
		}

		// TODO: don't return episodes they've already got
		subscriptionDetails[i].Podcast = p
	}
	// TODO: also get the latest positions...

	return json.NewEncoder(w).Encode(&subscriptionsSyncPostResponse{
		Subscriptions: subscriptionDetails,
	})
}
