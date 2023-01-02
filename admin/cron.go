package admin

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/podcreep/server/cron"
	"github.com/podcreep/server/store"
	"github.com/podcreep/server/util"
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

	schedule, err := util.ParseSchedule(cronJob.Schedule)
	if err != nil {
		return httpError(fmt.Sprintf("invalid schedule: %v", err), http.StatusBadRequest)
	}
	nextRun := schedule.NextTime(time.Now())
	if cronJob.Enabled {
		cronJob.NextRun = &nextRun
	} else {
		cronJob.NextRun = nil
	}

	log.Printf("Saving: %v\n", cronJob)
	err = store.SaveCronJob(r.Context(), cronJob)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/admin/cron", http.StatusFound)
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

		http.Redirect(w, r, "/admin/cron", http.StatusFound)
		return nil
	}

	return render(w, "cron/delete.html", map[string]interface{}{
		"CronJob": cronJob,
	})
}

func handleCronRunNow(w http.ResponseWriter, r *http.Request) error {
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

	return cron.RunCronJob(ctx, time.Now(), cronJob)
}

func handleCronValidateSchedule(w http.ResponseWriter, r *http.Request) error {
	s, err := util.ParseSchedule(r.URL.Query().Get("schedule"))
	if err != nil {
		// We don't return the error because we don't want to go through the normal processing, we are
		// expected to be called from AJAX.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	fmt.Fprintf(w, "%s", s)
	return nil
}
