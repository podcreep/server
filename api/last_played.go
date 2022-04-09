package api

import (
	"encoding/json"
	"net/http"

	"github.com/podcreep/server/store"
)

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

	return json.NewEncoder(w).Encode(ep)
}
