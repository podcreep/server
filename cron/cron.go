package cron

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
	"github.com/podcreep/server/util"
)

var (
	Jobs map[string]func(context.Context) error
)

// cronCheckUpdates checks for updates to our podcasts. To decide which podcast to update, we look
// at how long it has been since the last update: we update all podcasts that have not been updated
// in at last the last hour.
// TODO: allow us to configure the refresh frequency on a per-podcast basis.
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

	// Loop through all the podcasts, and stop when we get one that was updated in the last
	// hour.
	for _, p := range podcasts {
		if p.LastFetchTime.After(time.Now().Add(-1 * time.Hour)) {
			log.Printf("This podcast ('%s') was only updated at %v, not updating again.", p.Title, p.LastFetchTime)
			return nil
		}

		log.Printf("Updating podcast %s, LastFetchTime = %v", p.Title, p.LastFetchTime)
		numUpdated, err := UpdatePodcast(ctx, p, 0 /*flags*/)
		if err != nil {
			return fmt.Errorf("Error updating podcast: %w", err)
		}
		log.Printf(" - updated %d episodes", numUpdated)
	}
	return nil
}

func UpdatePodcast(ctx context.Context, podcast *store.Podcast, flags rss.UpdatePodcastFlags) (int, error) {
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
	numUpdated, error := rss.UpdatePodcast(ctx, podcast, flags)

	// Update the last fetch time.
	podcast.LastFetchTime = time.Now()
	_, err = store.SavePodcast(ctx, podcast)

	return numUpdated, error
}

func RunCronJob(ctx context.Context, now time.Time, job *store.CronJob) error {
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
		err := RunCronJob(ctx, now, job)
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

	// Run the cron goroutine start away.
	go runCronIterate()

	return nil
}
