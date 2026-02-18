package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Auth middleware — extracts Bearer token and attaches user to context.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing or invalid token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		user, err := GetUserByToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// optionalAuth attaches user if token present, but doesn't require it.
func optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if user, err := GetUserByToken(token); err == nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	}
}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userContextKey).(*User)
	return u
}

// ---- JSON helpers ----

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func parseListQuery(r *http.Request) ListQuery {
	q := ListQuery{
		Search: r.URL.Query().Get("search"),
		Author: r.URL.Query().Get("author"),
		Page:   1,
		Limit:  20,
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		q.Page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		q.Limit = l
	}
	return q
}
