package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/podcreep/server/admin"
	"github.com/podcreep/server/api"
	"github.com/podcreep/server/cron"
	"github.com/podcreep/server/discover"
	"github.com/podcreep/server/store"
)

func setupStaticFiles(r *mux.Router) {
	// The static files under the /admin directory.
	r.PathPrefix("/admin/static").Handler(http.StripPrefix("/admin/static", http.FileServer(http.Dir("./admin/static"))))

	// All the static files are stored under /dist but we want them to map to /
	r.Path("/{file:.*}.{ext:js|css|ico|html}").Handler(http.FileServer(http.Dir("./dist")))

	// Every other path, including bare /, maps to the index.html file.
	r.Path("/{url:.*}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./dist/index.html")
	})
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := store.Setup(); err != nil {
		panic(err)
	}
	r := mux.NewRouter()
	if err := admin.Setup(r); err != nil {
		panic(err)
	}
	if err := api.Setup(r); err != nil {
		panic(err)
	}
	if err := discover.Setup(); err != nil {
		panic(err)
	}
	if err := cron.Setup(r); err != nil {
		panic(err)
	}
	setupStaticFiles(r)

	var handler http.Handler
	handler = r
	if os.Getenv("DEBUG") != "" {
		// Allow requests from other domains in debug mode (in particular, the angular stuff will be
		// running on a different domain in debug mode).
		handler = handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"content-type", "authorization"}))(handler)
	}

	// Add logging to stdout.
	handler = handlers.LoggingHandler(os.Stdout, handler)

	http.Handle("/", handler)

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil))
}
