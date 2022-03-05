package admin

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/podcreep/server/cron"
	"github.com/podcreep/server/store"
)

func handleCron(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	cronJobs, err := store.LoadCrobJobs(ctx)
	if err != nil {
		return err
	}

	return render(w, "cron/list.html", map[string]interface{}{
		"CronJobs": cronJobs,
	})
}

func renderEditPage(w http.ResponseWriter, cronJob *store.CronJob) error {
	return render(w, "cron/edit.html", map[string]interface{}{
		"AvailableJobs": cron.GetCronJobNames(),
		"CronJob":       cronJob,
	})
}

func handleCronAdd(w http.ResponseWriter, r *http.Request) error {
	return renderEditPage(w, &store.CronJob{})
}

func handleCronEdit(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return renderEditPage(w, &store.CronJob{})
	}

	if err := r.ParseForm(); err != nil {
		return httpError(fmt.Sprintf("error parsing form: %v", err), http.StatusBadRequest)
	}

	cronJob := &store.CronJob{}
	sid := r.Form.Get("id")
	if sid != "" {
		// TODO: load existing?
		// cron = store.LoadCronJob()
	}

	if err := schema.NewDecoder().Decode(cronJob, r.PostForm); err != nil {
		return httpError(fmt.Sprintf("error parsing form: %v", err), http.StatusBadRequest)
	}

	log.Printf("Saving: %v\n", cronJob)
	err := store.SaveCronJob(r.Context(), cronJob)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/admin/cron", 302)
	return nil
}

func handleCronDelete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	cronID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return httpError(err.Error(), http.StatusBadRequest)
	}

	cronJob, err := store.LoadCrobJob(ctx, cronID)
	if err != nil {
		return err
	}

	if r.Method == "POST" {
		if err := store.DeleteCronJob(ctx, cronID); err != nil {
			return err
		}

		http.Redirect(w, r, "/admin/cron", 302)
		return nil
	}

	return render(w, "cron/delete.html", map[string]interface{}{
		"CronJob": cronJob,
	})
}
