package cron

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

func handleCronCheckUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error loading podcasts: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		log.Printf("Error creating CloudTask client: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	log.Printf("Got %d podcasts.\n", len(podcasts))
	for _, p := range podcasts {
		req := &taskspb.CreateTaskRequest{
			Parent: "projects/podcreep/locations/us-central1/queues/podcast-updater",
			Task: &taskspb.Task{
				// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
				MessageType: &taskspb.Task_HttpRequest{
					HttpRequest: &taskspb.HttpRequest{
						HttpMethod: taskspb.HttpMethod_GET,
						Url:        fmt.Sprintf("/cron/tasks/update-podcast/%d", p.ID),
					},
				},
			},
		}

		_, err := client.CreateTask(ctx, req)
		if err != nil {
			log.Printf("Error creating CloudTask task: %v\n", err)
		}
	}
}

func handleCronTaskUpdatePodcast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	podcast, err := store.GetPodcast(ctx, podcastID)
	if err != nil {
		log.Printf("Error loading podcasts: %v\n", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// if the latest episode from this podcast is < 6 hours old, we won't try to re-fetch it.
	var newestEpisodeDate time.Time
	for _, ep := range podcast.Episodes {
		if ep.PubDate.After(newestEpisodeDate) {
			newestEpisodeDate = ep.PubDate
		}
	}
	log.Printf("Newest episode was last updated: %v", newestEpisodeDate)

	log.Printf("Updating podcast: %v", podcast)
	rss.UpdatePodcast(ctx, podcast)
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/cron/check-updates", handleCronCheckUpdates).Methods("GET")
	r.HandleFunc("/cron/tasks/update-podcast/{id:[0-9]+}", handleCronTaskUpdatePodcast).Methods("GET")

	return nil
}