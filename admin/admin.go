// Package admin contains the backend-management features we use to manage feeds, etc.
package admin

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/util"
)

type sessionInfo struct{}

var (
	sessions = make(map[string]sessionInfo)
)

func handleHome(w http.ResponseWriter, r *http.Request) error {
	data := struct {
	}{}

	return render(w, "index.html", data)
}

func handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return render(w, "login.html", nil)
	}

	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("error parsing form: %w", err)
	}

	password := r.Form.Get("password")
	if os.Getenv("ADMIN_PASSWORD") == "" || password != os.Getenv("ADMIN_PASSWORD") {
		log.Printf("Admin password does not match ADMIN_PASSWORD environment variable")
		http.Error(w, "Invalid password", 400)
		return httpError("Password does not match", http.StatusForbidden)
	}

	cookieValue, err := util.CreateCookie()
	if err != nil {
		return fmt.Errorf("error creating cookie: %w", err)
	}

	sessions[cookieValue] = sessionInfo{}

	expire := time.Now().AddDate(0, 0, 1)
	cookie := http.Cookie{
		Name:    "sess",
		Value:   cookieValue,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)

	redirectUrl := r.URL.Query().Get("from")
	log.Printf("redirect: %s", redirectUrl)
	if redirectUrl == "" {
		log.Print("it's empty")
		redirectUrl = "/admin"
	}
	http.Redirect(w, r, redirectUrl, 302)
	return nil
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

		loginUrl := "/admin/login?from=" + url.QueryEscape(r.URL.Path)

		cookie, err := r.Cookie("sess")
		if err != nil {
			log.Printf("No cookie, redirecting to login.")
			http.Redirect(w, r, loginUrl, 302)
			return
		}

		if _, ok := sessions[cookie.Value]; !ok {
			log.Printf("Invalid cookie value, redirecting to login.")
			http.Redirect(w, r, loginUrl, 302)
			return
		}

		// All good, off you go.
		h.ServeHTTP(w, r)
	})
}

type adminRequestError struct {
	Err     error
	Message string
	Code    int
}

func (requestErr adminRequestError) Error() string {
	return fmt.Sprintf("%v [%s] %d", requestErr.Err, requestErr.Message, requestErr.Code)
}

func httpError(msg string, code int) *adminRequestError {
	return &adminRequestError{nil, msg, code}
}

type wrappedRequest func(http.ResponseWriter, *http.Request) error

func wrap(fn wrappedRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			requestErr, ok := err.(adminRequestError)
			if !ok {
				requestErr = adminRequestError{err, err.Error(), 500}
			}
			log.Printf("Error in request: %v", requestErr.Error())
			http.Error(w, requestErr.Message, requestErr.Code)
		}
	}
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

	subr.HandleFunc("/", wrap(handleHome)).Methods("GET")
	subr.HandleFunc("/login", wrap(handleLogin)).Methods("GET", "POST")
	subr.HandleFunc("/podcasts", wrap(handlePodcastsList)).Methods("GET")
	subr.HandleFunc("/podcasts/add", wrap(handlePodcastsAdd)).Methods("GET", "POST")
	subr.HandleFunc("/podcasts/edit", wrap(handlePodcastsEditPost)).Methods("POST")
	subr.HandleFunc("/cron", wrap(handleCron)).Methods("GET")
	subr.HandleFunc("/cron/add", wrap(handleCronAdd)).Methods("GET")
	subr.HandleFunc("/cron/edit", wrap(handleCronEdit)).Methods("GET", "POST")
	subr.HandleFunc("/cron/{id:[0-9]+}/delete", wrap(handleCronDelete)).Methods("GET", "POST")
	subr.HandleFunc("/cron/validate-schedule", wrap(handleCronValidateSchedule)).Methods("GET")

	return nil
}
