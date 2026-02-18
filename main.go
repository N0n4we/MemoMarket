package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Server metadata — loaded from config file, env vars as fallback.
var serverName = "MemoMarket"
var serverDescription = ""

func loadServerConfig(dataDir string) {
	configPath := filepath.Join(dataDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// No config file yet — create one from env vars / defaults
		saveServerConfig(dataDir)
		return
	}
	var info ServerInfo
	if err := json.Unmarshal(data, &info); err == nil {
		if info.Name != "" {
			serverName = info.Name
		}
		serverDescription = info.Description
	}
}

func saveServerConfig(dataDir string) {
	configPath := filepath.Join(dataDir, "config.json")
	data, _ := json.MarshalIndent(ServerInfo{Name: serverName, Description: serverDescription}, "", "  ")
	os.WriteFile(configPath, data, 0644)
}

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

	os.MkdirAll(dataDir, 0755)
	loadServerConfig(dataDir)
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
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/me", authMiddleware(handleMe))

	// Memo Packs — route by method
	mux.HandleFunc("/api/memo-packs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListMemoPacks(w, r)
		case http.MethodPost:
			authMiddleware(handlePublishMemoPack)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})
	mux.HandleFunc("/api/memo-packs/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/download") {
			handleDownloadMemoPack(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleGetMemoPack(w, r)
		case http.MethodPut:
			authMiddleware(handleUpdateMemoPack)(w, r)
		case http.MethodDelete:
			authMiddleware(handleDeleteMemoPack)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})

	handler := corsMiddleware(mux)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
