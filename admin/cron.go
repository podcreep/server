package admin

import (
	"log"
	"net/http"

	"github.com/gorilla/schema"
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

func handleCronAdd(w http.ResponseWriter, r *http.Request) {
	render(w, "cron/edit.html", map[string]interface{}{
		"CronJob": store.CronJob{},
	})
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

	render(w, "cron/edit.html", map[string]interface{}{
		"CronJob": store.CronJob{},
	})
}
