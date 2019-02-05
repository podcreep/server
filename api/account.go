package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/podcreep/server/store"
)

func handleAccountsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("username") == "codeka" {
		return
	}
	http.Error(w, "Something", 404)
}

type accountsPostRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type accountsPostResponse struct {
	Cookie string `json:"cookie"`
}

func handleAccountsPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req accountsPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding: %v\n", err)
		http.Error(w, "Error parsing request.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	acct, err := store.SaveAccount(ctx, req.Username, req.Password)
	if err != nil {
		log.Printf("Error saving account: %v\n", err)
		http.Error(w, "Error saving account.", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(&accountsPostResponse{Cookie: acct.Cookie})
	if err != nil {
		log.Printf("Error encoding account: %v\n", err)
		http.Error(w, "Error encoding account.", http.StatusInternalServerError)
		return
	}
}

func handleAccountsLoginPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req accountsPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding: %v\n", err)
		http.Error(w, "Error parsing request.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	acct, err := store.LoadAccountByUsername(ctx, req.Username, req.Password)
	if err != nil {
		log.Printf("Error saving account: %v\n", err)
		http.Error(w, "Error saving account.", http.StatusInternalServerError)
		return
	}

	if acct == nil {
		http.Error(w, "Invalid username/password", http.StatusUnauthorized)
		return
	}

	err = json.NewEncoder(w).Encode(&accountsPostResponse{Cookie: acct.Cookie})
	if err != nil {
		log.Printf("Error encoding account: %v\n", err)
		http.Error(w, "Error encoding account.", http.StatusInternalServerError)
		return
	}
}
