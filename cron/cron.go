package cron

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
	"github.com/podcreep/server/util"
)

var (
	Jobs map[string]func(context.Context) error
)

// cronCheckUpdates checks for updates to our podcasts. We only do one podcast per call to this
// method, so it should be called relatively frequenctly.
//
// To decide which podcast to update, we look at how long it has been since the last update: we
// pick the podcast with the oldest update, as long as it's been more than one hour.
func cronCheckUpdates(ctx context.Context) error {
	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		return err
	}

	if len(podcasts) == 0 {
		log.Printf("No podcasts.")
		return nil
	}

	// Sort the podcasts by LastFetchTime, so that the first podcast in the list is the one that
	// we haven't fetched for the longer time.
	sort.Slice(podcasts, func(i, j int) bool {
		return podcasts[i].LastFetchTime.Before(podcasts[j].LastFetchTime)
	})

	p := podcasts[0]
	if p.LastFetchTime.After(time.Now().Add(-1 * time.Hour)) {
		log.Printf("Oldest podcast ('%s') was only updated at %v, not updating again.", p.Title, p.LastFetchTime)
		return nil
	}

	log.Printf("Updating podcast %s, LastFetchTime = %v", p.Title, p.LastFetchTime)
	numUpdated, err := updatePodcast(ctx, p, false)
	if err != nil {
		return fmt.Errorf("Error updating podcast: %w", err)
	}
	log.Printf(" - updated %d episodes", numUpdated)
	return nil
}

// handleCronForceUpdate does a "force" update on a podcast, including re-downloading and storing
// all episodes. This is useful if we change our parsing or storing logic or something and we need
// to refresh the whole thing.
func handleCronForceUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		io.WriteString(w, fmt.Sprintf("Error parsing ID: %s\n", vars["id"]))
		return
	}

	p, err := store.GetPodcast(ctx, podcastID)
	if err != nil {
		log.Printf("Error fetching podcast: %v", err)
		io.WriteString(w, fmt.Sprintf("Error fetching podcast: %v", err))
		return
	}

	p.Episodes, err = store.LoadEpisodes(ctx, p.ID, 20)
	if err != nil {
		log.Printf("Error fetching recent episodes: %v", err)
		io.WriteString(w, fmt.Sprintf("Error fetching episodes: %v", err))
	}

	numUpdated, err := updatePodcast(ctx, p, true)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("Error updating podcast: %v", err))
	} else {
		io.WriteString(w, fmt.Sprintf("Updated: %s (%d episodes)", p.Title, numUpdated))
	}
}

// handleClearEpisodes clears all of the episodes for a podcast. This is mostly just used for
// debugging/testing.
func handleClearEpisodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s", vars["id"])
		io.WriteString(w, fmt.Sprintf("Error parsing ID: %s", vars["id"]))
		return
	}

	err = store.ClearEpisodes(ctx, podcastID)
	if err != nil {
		log.Printf("Error clearing episodes: %v", err)
		io.WriteString(w, fmt.Sprintf("Error clearing episodes: %v", err))
		return
	}
}

func updatePodcast(ctx context.Context, podcast *store.Podcast, force bool) (int, error) {
	// The podcast we get here will not have the episodes populated, as it comes from the list.
	// So fetch the episodes manually. We just get the latest 10 episodes. Anything older than this
	// we will ignore entirely.
	episodes, err := store.LoadEpisodes(ctx, podcast.ID, 10)
	if err != nil {
		log.Printf("Error fetching podcast: %v", err)
		return 0, err
	}
	podcast.Episodes = episodes

	// Actually do the update.
	numUpdated, error := rss.UpdatePodcast(ctx, podcast, force)

	// Update the last fetch time.
	podcast.LastFetchTime = time.Now()
	_, err = store.SavePodcast(ctx, podcast)

	return numUpdated, error
}

func runJob(ctx context.Context, now time.Time, job *store.CronJob) error {
	found := false
	for n, fn := range Jobs {
		if n == job.Name {
			found = true

			err := fn(ctx)
			if err != nil {
				return fmt.Errorf("Error running job: %v", err)
			}
		}
	}

	if !found {
		job.Enabled = false
		return fmt.Errorf("Job does not exist: %s", job.Name)
	}

	sched, err := util.ParseSchedule(job.Schedule)
	if err != nil {
		job.Enabled = false
		return fmt.Errorf("Job has invalid schedule, cannot reschedule: %w", err)
	}
	nextRun := sched.NextTime(now)
	job.NextRun = &nextRun
	return nil
}

// cronIterate is run in a goroutine to actually execute the cron tasks.
func cronIterate() error {
	ctx := context.Background()

	now := time.Now()
	jobs, err := store.LoadPendingCronJobs(ctx, now)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		err := runJob(ctx, now, job)
		if err != nil {
			return err
		} else {
			err := store.SaveCronJob(ctx, job)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// runCronIterate is a helper that runs cronIterate and then schedules itself to run again.
func runCronIterate() {
	ctx := context.Background()
	now := time.Now()
	timeToWait := store.GetTimeToNextCronJob(ctx, now)

	log.Printf("Waiting %v to next cron job", timeToWait)
	time.Sleep(timeToWait)

	err := cronIterate()
	if err != nil {
		log.Printf("Error running cronIterate: %v", err)
		// Keep going, schedule again.
	}

	// Schedule to run again.
	go runCronIterate()
}

// Gets a list of the cron job names.
func GetCronJobNames() []string {
	var names []string
	for k := range Jobs {
		names = append(names, k)
	}
	return names
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	Jobs = make(map[string]func(context.Context) error)
	Jobs["check-updates"] = cronCheckUpdates
	//	r.HandleFunc("/cron/force-update/{id:[0-9]+}", handleCronForceUpdate).Methods("GET")
	//	r.HandleFunc("/cron/clear-episodes/{id:[0-9]+}", handleClearEpisodes).Methods("GET")

	// Run the cron goroutine start away.
	go runCronIterate()

	return nil
}
