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

	r.HandleFunc("/", handleDefault)

	var handler http.Handler
	handler = r
	if os.Getenv("RUN_WITH_DEVAPPSERVER") != "" {
		// Allow requests from other domains in dev mode (in particular, the angular stuff will be
		// running on a different domain in dev mode).
		handler = handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"content-type", "authorization"}))(handler)
	}

	http.Handle("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
