package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

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

	details := &podcastDetails{}

	// First, see if we have this podcast already stored in our data store.
	podcast, err := store.LoadPodcastByDiscoverId(ctx, vars["id"])
	if podcast != nil && err == nil {
		log.Println("got an existing podcast")
		details.Podcast = *podcast

		episodes, err := store.LoadEpisodes(ctx, podcast.ID, 10)
		if err != nil {
			return err
		}
		details.Episodes = episodes
	} else {
		log.Println("existing podcast doesn't exist")
		podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
		if err != nil {
			return err
		}

		podcast, episodes, err := discover.FetchPodcast(podcastID /*includeEpisodes*/, true)
		if err != nil {
			// I'm not sure what the best HTTP status to return here is?
			return err
		}

		// Translate to our podcastDetails that the client understands.
		details.Podcast = store.Podcast{
			ID:              podcast.ID,
			DiscoverID:      strconv.FormatInt(podcast.ID, 10),
			Title:           podcast.Title,
			Description:     podcast.Description,
			ImageURL:        podcast.ImageUrl,
			IsImageExternal: true, // Discover podcasts haven't been saved yet, so the icons are external.
			FeedURL:         podcast.Url,
			// TODO: link?
		}

		for _, episode := range episodes {
			details.Episodes = append(details.Episodes, &store.Episode{
				ID:          episode.ID,
				PodcastID:   podcast.ID,
				Title:       episode.Title,
				Description: episode.Description,
				PubDate:     time.Unix(episode.DatePublished, 0),
				// TODO: Duration?
			})
		}
	}

	err = json.NewEncoder(w).Encode(&details)
	if err != nil {
		return err
	}

	return nil
}
