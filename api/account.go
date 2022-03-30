package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/podcreep/server/store"
)

func handleAccountsGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	username := r.URL.Query().Get("username")
	exists, err := store.VerifyUsernameExists(ctx, username)
	if err != nil {
		return fmt.Errorf("error querying for username: %v", err)
	}

	if !exists {
		return apiError("Username does not exist", http.StatusNotFound)
	}
	return nil
}

type accountsPostRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type accountsPostResponse struct {
	Cookie string `json:"cookie"`
}

func handleAccountsPost(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var req accountsPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	acct, err := store.SaveAccount(ctx, req.Username, req.Password)
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(&accountsPostResponse{Cookie: acct.Cookie})
	if err != nil {
		return err
	}

	return nil
}

func handleAccountsLoginPost(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	log.Printf("Request: %s", string(b))

	var req accountsPostRequest
	err = json.NewDecoder(bytes.NewReader(b)).Decode(&req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	acct, err := store.LoadAccountByUsername(ctx, req.Username, req.Password)
	if err != nil {
		log.Printf("Error loading account for %s: %v", req.Username, err)
		return apiError("Invalid username/password", http.StatusUnauthorized)
	}

	if acct == nil {
		return apiError("Invalid username/password", http.StatusUnauthorized)
	}

	err = json.NewEncoder(w).Encode(&accountsPostResponse{Cookie: acct.Cookie})
	if err != nil {
		return err
	}

	return nil
}
