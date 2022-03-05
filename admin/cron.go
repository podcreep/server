package admin

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/podcreep/server/cron"
	"github.com/podcreep/server/store"
)

func handleCron(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cronJobs, err := store.LoadCrobJobs(ctx)
	if err != nil {
		log.Printf("error: %v\n", err)
		// TODO: handle error
	}

	log.Printf("Got %d jobs", len(cronJobs))
	render(w, "cron/list.html", map[string]interface{}{
		"CronJobs": cronJobs,
	})
}

func renderEditPage(w http.ResponseWriter, cronJob *store.CronJob) {
	render(w, "cron/edit.html", map[string]interface{}{
		"AvailableJobs": cron.GetCronJobNames(),
		"CronJob":       cronJob,
	})
}

func handleCronAdd(w http.ResponseWriter, r *http.Request) {
	renderEditPage(w, &store.CronJob{})
}

func handleCronEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("r.Method=%s", r.Method)
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			log.Printf("Error parsing form: %v", err)
			http.Error(w, "Error parsing form", 400)
			return
		}

		cronJob := &store.CronJob{}
		sid := r.Form.Get("id")
		if sid != "" {
			// TODO: load existing?
			// cron = store.LoadCronJob()
		}

		if err := schema.NewDecoder().Decode(cronJob, r.PostForm); err != nil {
			log.Printf("Error parsing form: %v", err)
			http.Error(w, "Error parsing form", 400)
			return
		}

		log.Printf("Saving: %v\n", cronJob)
		err := store.SaveCronJob(ctx, cronJob)
		if err != nil {
			log.Printf("Error saving podcast: %v", err)
			http.Error(w, "Error saving podcast", 500)
			return
		}

		http.Redirect(w, r, "/admin/cron", 302)
		return
	}

	renderEditPage(w, &store.CronJob{})
}

func handleCronDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	cronID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cronJob, err := store.LoadCrobJob(ctx, cronID)
	if err != nil {
		log.Printf("error: %v\n", err)
		// TODO: handle error
	}

	if r.Method == "POST" {
		if err := store.DeleteCronJob(ctx, cronID); err != nil {
			log.Printf("error: %v", err)
			// TODO: handle error
		}

		http.Redirect(w, r, "/admin/cron", 302)
		return
	}

	render(w, "cron/delete.html", map[string]interface{}{
		"CronJob": cronJob,
	})
}
