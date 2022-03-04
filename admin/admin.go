// Package admin contains the backend-management features we use to manage feeds, etc.
package admin

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/util"
)

type sessionInfo struct{}

var (
	sessions = make(map[string]sessionInfo)
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	data := struct {
	}{}

	render(w, "index.html", data)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		render(w, "login.html", nil)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Error parsing form", 400)
		return
	}

	password := r.Form.Get("password")
	if os.Getenv("ADMIN_PASSWORD") == "" || password != os.Getenv("ADMIN_PASSWORD") {
		log.Printf("Admin password does not match ADMIN_PASSWORD environment variable")
		http.Error(w, "Invalid password", 400)
		return
	}

	cookieValue, err := util.CreateCookie()
	if err != nil {
		log.Printf("Error creating cookie: %v", err)
		http.Error(w, "Error creating cookie", 500)
		return
	}

	sessions[cookieValue] = sessionInfo{}

	expire := time.Now().AddDate(0, 0, 1)
	cookie := http.Cookie{
		Name:    "sess",
		Value:   cookieValue,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/admin", 302)
}

// authMiddleware is some middleware that ensures the user is authenticated before allowing them
// access to any pages under /admin (the only exception is you can access /admin/login).
func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/login" {
			// Anybody can access login, otherwise how can you log in??
			h.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("sess")
		if err != nil {
			log.Printf("No cookie, redirecting to login.")
			http.Redirect(w, r, "/admin/login", 302)
			return
		}

		if _, ok := sessions[cookie.Value]; !ok {
			log.Printf("Invalid cookie value, redirecting to login.")
			http.Redirect(w, r, "/admin/login", 302)
			return
		}

		// All good, off you go.
		h.ServeHTTP(w, r)
	})
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	if err := initTemplates(); err != nil {
		return err
	}

	// Requests to /admin (no trailing slash) redirect to /admin/ (with trailing slash)
	r.Path("/admin").Handler(http.RedirectHandler("/admin/", http.StatusMovedPermanently))

	subr := r.PathPrefix("/admin").Subrouter()
	subr.Use(authMiddleware)

	subr.HandleFunc("/", handleHome).Methods("GET")
	subr.HandleFunc("/login", handleLogin).Methods("GET", "POST")
	subr.HandleFunc("/podcasts", handlePodcastsList).Methods("GET")
	subr.HandleFunc("/podcasts/add", handlePodcastsAdd).Methods("GET", "POST")
	subr.HandleFunc("/podcasts/edit", handlePodcastsEditPost).Methods("POST")
	subr.HandleFunc("/cron", handleCron).Methods("GET")
	subr.HandleFunc("/cron/add", handleCronAdd).Methods("GET")
	subr.HandleFunc("/cron/edit", handleCronEdit).Methods("GET", "POST")

	return nil
}
