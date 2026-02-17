package main

import "net/http"

// POST /api/register — create a new user, returns token.
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	var req RegisterReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}
	if req.Username == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username is required"})
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	user, err := CreateUser(req.Username, req.DisplayName)
	if err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// GET /api/me — get current user info.
func handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}
