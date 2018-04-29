package api

import "github.com/gorilla/mux"

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/api/accounts", handleAccounts).Methods("GET")

	return nil
}
