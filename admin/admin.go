// Package admin contains the backend-management features we use to manage feeds, etc.
package admin

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/util"

	oauth2 "google.golang.org/api/oauth2/v2"
)

const (
	clientID = "683097828984-0bsih3puj8t271s3igc97spje3igr1v7.apps.googleusercontent.com"
)

var (
	// sessions is the in-memory map we keep our session info in.
	// TODO(dean): Make this stored in the database or something
	sessions = make(map[string]sessionInfo)
)

type sessionInfo struct {
	email string
}

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

	svc, err := oauth2.New(&http.Client{})
	if err != nil {
		log.Printf("Error getting oauth2 service: %v", err)
		http.Error(w, "Error getting oauth2 service", 500)
		return
	}

	idToken := r.Form.Get("idToken")
	tokenInfo, err := svc.Tokeninfo().IdToken(idToken).Do()
	if err != nil {
		log.Printf("Error getting tokeninfo: %v", err)
		http.Error(w, "Error getting tokeninfo", 403)
		return
	}

	if tokenInfo.Email != r.Form.Get("email") {
		log.Printf("Associated email does not match (given %s, wanted %s)", r.Form.Get("email"), tokenInfo.Email)
		http.Error(w, "Forbidden", 403)
		return
	}

	if tokenInfo.Email != "dean@codeka.com.au" {
		log.Printf("Someone, not me, has tried to log in: %s", tokenInfo.Email)
		http.Error(w, "Forbidden", 403)
		return
	}

	if tokenInfo.Audience != clientID {
		log.Printf("Audience does not match (given %s, wanted %s)", tokenInfo.Audience, clientID)
		http.Error(w, "Forbidden", 403)
		return
	}

	cookieValue, err := util.CreateCookie()
	if err != nil {
		log.Printf("Error creating cookie: %v", err)
		http.Error(w, "Error creating cookie", 500)
		return
	}

	sessions[cookieValue] = sessionInfo{
		email: tokenInfo.Email,
	}

	expire := time.Now().AddDate(0, 0, 1)
	cookie := http.Cookie{
		Name:    "sess",
		Value:   cookieValue,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)
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

		sess := sessions[cookie.Value]
		if sess.email != "dean@codeka.com.au" {
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

	return nil
}
