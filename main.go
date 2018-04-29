package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/podcreep/server/admin"
	"github.com/podcreep/server/api"
	"google.golang.org/appengine"
)

func handleDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}

func main() {
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
	if appengine.IsDevAppServer() {
		// Allow requests from other domains in dev mode (in particular, the angular stuff will be
		// running on a different domain in dev mode).
		handler = handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"content-type"}))(handler)
	}

	http.Handle("/", handler)
	appengine.Main()
}
