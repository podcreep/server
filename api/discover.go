package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/discover"
	"github.com/podcreep/server/store"
)

func convert(inp []discover.Podcast) podcastList {
	outp := podcastList{}
	for _, podcast := range inp {
		outp.Podcasts = append(outp.Podcasts, &podcastDetails{
			Podcast: store.Podcast{
				ID:              podcast.ID,
				Title:           podcast.Title,
				Description:     podcast.Description,
				ImageURL:        podcast.ImageUrl,
				IsImageExternal: true, // Discover podcasts haven't been saved yet, so the icons are external.
				FeedURL:         podcast.Url,
			},
		})
	}
	return outp
}

func convertToJson(inp []discover.Podcast, w http.ResponseWriter) error {
	outp := convert(inp)
	return json.NewEncoder(w).Encode(&outp)
}

// handleDiscoverTrendingGet handles GET requests for /api/discover/trending. It returns podcasts in "trending" order.
func handleDiscoverTrendingGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	_, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcasts, err := discover.FetchTrending()
	if err != nil {
		// I'm not sure what the best HTTP status to return here is?
		return err
	}

	// Translate to our podcastDetails that the client understands.
	return convertToJson(podcasts, w)
}

// handleDiscoverSearchGet handles GET requests for /api/discover/search. It allows clients to search for postcasts.
func handleDiscoverSearchGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	query := r.URL.Query().Get("q")
	if query == "" {
		return handleDiscoverTrendingGet(w, r)
	}

	_, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcasts, err := discover.Search(query)
	if err != nil {
		// I'm not sure what the best HTTP status to return here is?
		return err
	}

	// Translate to our podcastDetails that the client understands.
	return convertToJson(podcasts, w)
}

func handleDiscoverPodcastGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	_, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	podcast, err := discover.FetchPodcast(podcastID)
	if err != nil {
		// I'm not sure what the best HTTP status to return here is?
		return err
	}

	// Translate to our podcastDetails that the client understands.
	details := &podcastDetails{
		Podcast: store.Podcast{
			ID:              podcast.ID,
			Title:           podcast.Title,
			Description:     podcast.Description,
			ImageURL:        podcast.ImageUrl,
			IsImageExternal: true, // Discover podcasts haven't been saved yet, so the icons are external.
			FeedURL:         podcast.Url,
			// TODO: link?
		},
	}
	err = json.NewEncoder(w).Encode(&details)
	if err != nil {
		return err
	}

	return nil
}
