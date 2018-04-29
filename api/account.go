package api

import "net/http"

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("username") == "codeka" {
		return
	}
	http.Error(w, "Something", 404)
}
