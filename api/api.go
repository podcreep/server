package api

import "github.com/gorilla/mux"

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/api/accounts", handleAccountsGet).Methods("GET")
	r.HandleFunc("/api/accounts", handleAccountsPost).Methods("POST")
	r.HandleFunc("/api/accounts/login", handleAccountsLoginPost).Methods("POST")

	return nil
}
