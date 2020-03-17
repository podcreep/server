package cron

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newCloudTasksClient(ctx context.Context) (*cloudtasks.Client, error) {
	cloudTasksHost := os.Getenv("CLOUDTASKS_HOST")
	if cloudTasksHost != "" {
		conn, err := grpc.Dial(cloudTasksHost, grpc.WithInsecure())
		if err != nil {
			return nil, fmt.Errorf("cannot grpc.Dial: %v", err)
		}

		clientOpt := option.WithGRPCConn(conn)
		return cloudtasks.NewClient(ctx, clientOpt)
	} else {
		return cloudtasks.NewClient(ctx)
	}
}

func handleCronCheckUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error loading podcasts: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	client, err := newCloudTasksClient(ctx)
	if err != nil {
		log.Printf("Error creating CloudTask client: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	parent := "projects/podcreep/locations/us-central1"
	createQueueRequest := taskspb.CreateQueueRequest{
		Parent: parent,
		Queue: &taskspb.Queue{
			Name: parent + "/queues/podcast-updater",
		},
	}
	queue, err := client.CreateQueue(ctx, &createQueueRequest)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			// Queue already exists, this is OK.
		} else {
			log.Printf("Error creating queue: %v\n", err)

			// TODO: something else?
		}
	}

	log.Printf("Got %d podcasts.\n", len(podcasts))
	for _, p := range podcasts {
		req := &taskspb.CreateTaskRequest{
			Parent: queue.GetName(),
			Task: &taskspb.Task{
				MessageType: &taskspb.Task_HttpRequest{
					HttpRequest: &taskspb.HttpRequest{
						HttpMethod: taskspb.HttpMethod_GET,
						Url:        fmt.Sprintf("%s/cron/tasks/update-podcast/%d", os.Getenv("BASE_URL"), p.ID),
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
