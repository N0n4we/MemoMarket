package main

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// POST /api/register — create a new user with username/password, returns token.
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
	if req.Password == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "password is required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to hash password"})
		return
	}

	user, err := CreateUser(req.Username, string(hash))
	if err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// POST /api/login — authenticate with username/password, returns user with token.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	var req LoginReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username and password are required"})
		return
	}

	user, err := GetUserByUsername(req.Username)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid username or password"})
		return
	}

	// Clear hash before responding
	user.PasswordHash = ""
	writeJSON(w, http.StatusOK, user)
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
