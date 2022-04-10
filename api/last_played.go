package api

import (
	"encoding/json"
	"net/http"

	"github.com/podcreep/server/store"
)

// LastPlayedResponse is the response we give for a last played request.
type LastPlayedResponse struct {
	Podcast *store.Podcast `json:"podcast"`
	Episode *store.Episode `json:"episode"`
}

// handleLastPlayedGet handles requests to get the last played episode for a user.
func handleLastPlayedGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Not authorized", http.StatusUnauthorized)
	}

	ep, err := store.GetMostRecentPlaybackState(ctx, acct)
	if err != nil {
		// You're not subscribed to this episode. We don't save the state if you're not subbed.
		return apiError("No recently-played", http.StatusNotFound)
	}

	podcast, err := store.LoadPodcast(ctx, ep.PodcastID)
	if err != nil {
		// Shouldn't happen, but you never know.
		return apiError("Error fetching podcast", http.StatusInternalServerError)
	}

	resp := LastPlayedResponse{
		Podcast: podcast,
		Episode: ep,
	}
	return json.NewEncoder(w).Encode(&resp)
}
