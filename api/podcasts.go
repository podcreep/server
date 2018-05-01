package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/store"
)

type podcastList struct {
	Podcasts []*store.Podcast `json:"podcasts"`
}

// handlePodcastsGet handles requests to view all the podcasts we have in our DB.
// TODO: support filtering, sorting, paging, etc etc.
func handlePodcastsGet(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if _, err := authenticate(ctx, r); err != nil {
		log.Warningf(ctx, "%v", err)
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Warningf(ctx, "Error fetching podcasts: %v", err)
		http.Error(w, "Error fetching podcasts.", http.StatusInternalServerError)
		return
	}

	list := podcastList{
		Podcasts: podcasts,
	}
	err = json.NewEncoder(w).Encode(&list)
	if err != nil {
		log.Errorf(ctx, "Error encoding podcasts: %v", err)
		http.Error(w, "Error encoding podcasts.", http.StatusInternalServerError)
		return
	}
}

// handlePodcastGet handles requests to view all the podcasts we have in our DB.
// TODO: support filtering, sorting, paging, etc etc.
func handlePodcastGet(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	vars := mux.Vars(r)

	_, err := authenticate(ctx, r)
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

	podcast, err := store.GetPodcast(ctx, podcastID)
	if err != nil {
		log.Warningf(ctx, "Error fetching podcast: %v", err)
		http.Error(w, "Error fetching podcast.", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(&podcast)
	if err != nil {
		log.Errorf(ctx, "Error encoding podcasts: %v", err)
		http.Error(w, "Error encoding podcasts.", http.StatusInternalServerError)
		return
	}
}
