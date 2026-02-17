package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Server metadata — configurable via env vars.
var serverName = "MemoMarket"
var serverDescription = ""

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	if n := os.Getenv("SERVER_NAME"); n != "" {
		serverName = n
	}
	if d := os.Getenv("SERVER_DESC"); d != "" {
		serverDescription = d
	}

	InitDB(dataDir)
	log.Printf("MemoMarket backend starting on :%s (data: %s)", port, dataDir)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Server info — each backend node is a channel
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, ServerInfo{Name: serverName, Description: serverDescription})
	})

	// Auth
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/me", authMiddleware(handleMe))

	// Rule Packs — route by method
	mux.HandleFunc("/api/rule-packs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListRulePacks(w, r)
		case http.MethodPost:
			authMiddleware(handlePublishRulePack)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})
	mux.HandleFunc("/api/rule-packs/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/download") {
			handleDownloadRulePack(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleGetRulePack(w, r)
		case http.MethodPut:
			authMiddleware(handleUpdateRulePack)(w, r)
		case http.MethodDelete:
			authMiddleware(handleDeleteRulePack)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})

	// Server info update (admin-like, auth required)
	mux.HandleFunc("/api/info/update", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
			return
		}
		var info ServerInfo
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
			return
		}
		if info.Name != "" {
			serverName = info.Name
		}
		serverDescription = info.Description
		writeJSON(w, http.StatusOK, ServerInfo{Name: serverName, Description: serverDescription})
	}))

	handler := corsMiddleware(mux)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
