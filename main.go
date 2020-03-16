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
)

func handleDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()
	if err := admin.Setup(r); err != nil {
		panic(err)
	}
	if err := api.Setup(r); err != nil {
		panic(err)
	}
	if err := cron.Setup(r); err != nil {
		panic(err)
	}

	r.HandleFunc("/", handleDefault)

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
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
