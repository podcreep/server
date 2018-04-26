package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/admin"
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

	r.HandleFunc("/", handleDefault)

	http.Handle("/", r)
	appengine.Main()
}
