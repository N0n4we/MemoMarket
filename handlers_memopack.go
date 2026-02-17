package main

import (
	"net/http"
	"strings"
)

// GET /api/memo-packs — list published memo packs (public).
func handleListMemoPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	q := parseListQuery(r)
	packs, total, err := ListMemoPacks(q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list packs"})
		return
	}
	writeJSON(w, http.StatusOK, ListResponse{Items: packs, Total: total, Page: q.Page, Limit: q.Limit})
}

// GET /api/memo-packs/{id} — get a single memo pack (public).
func handleGetMemoPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	id := extractID(r.URL.Path, "/api/memo-packs/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing pack id"})
		return
	}
	pack, err := GetMemoPack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

// GET /api/memo-packs/{id}/download — download (increment counter + return pack).
func handleDownloadMemoPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/memo-packs/")
	id := strings.TrimSuffix(path, "/download")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing pack id"})
		return
	}
	pack, err := GetMemoPack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	IncrementMemoPackDownloads(id)
	pack.Downloads++
	writeJSON(w, http.StatusOK, pack)
}

// POST /api/memo-packs — publish a new memo pack (auth required).
func handlePublishMemoPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	var req PublishMemoPackReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}
	if req.Version == "" {
		req.Version = "1.0.0"
	}

	now := nowISO()
	pack := &MemoPack{
		ID:           newID(),
		Name:         req.Name,
		Description:  req.Description,
		AuthorID:     user.ID,
		AuthorName:   user.DisplayName,
		Version:      req.Version,
		SystemPrompt: req.SystemPrompt,
		Rules:        req.Rules,
		Memos:        req.Memos,
		Tags:         req.Tags,
		Downloads:    0,
		Published:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if pack.Rules == nil {
		pack.Rules = []MemoRule{}
	}
	if pack.Memos == nil {
		pack.Memos = []Memo{}
	}
	if pack.Tags == nil {
		pack.Tags = []string{}
	}

	if err := InsertMemoPack(pack); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to publish"})
		return
	}
	writeJSON(w, http.StatusCreated, pack)
}

// PUT /api/memo-packs/{id} — update own memo pack (auth required).
func handleUpdateMemoPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	id := extractID(r.URL.Path, "/api/memo-packs/")
	existing, err := GetMemoPack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	if existing.AuthorID != user.ID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "not your pack"})
		return
	}

	var req PublishMemoPackReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Version = req.Version
	existing.SystemPrompt = req.SystemPrompt
	existing.Rules = req.Rules
	existing.Memos = req.Memos
	existing.Tags = req.Tags
	if existing.Rules == nil {
		existing.Rules = []MemoRule{}
	}
	if existing.Memos == nil {
		existing.Memos = []Memo{}
	}
	if existing.Tags == nil {
		existing.Tags = []string{}
	}

	if err := UpdateMemoPack(existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update"})
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// DELETE /api/memo-packs/{id} — delete own memo pack (auth required).
func handleDeleteMemoPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	id := extractID(r.URL.Path, "/api/memo-packs/")
	existing, err := GetMemoPack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	if existing.AuthorID != user.ID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "not your pack"})
		return
	}

	if err := DeleteMemoPack(id, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func extractID(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}
