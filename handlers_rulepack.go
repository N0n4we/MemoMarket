package main

import (
	"net/http"
	"strings"
)

// GET /api/rule-packs — list published rule packs (public).
func handleListRulePacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	q := parseListQuery(r)
	packs, total, err := ListRulePacks(q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list packs"})
		return
	}
	writeJSON(w, http.StatusOK, ListResponse{Items: packs, Total: total, Page: q.Page, Limit: q.Limit})
}

// GET /api/rule-packs/{id} — get a single rule pack (public).
func handleGetRulePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	id := extractID(r.URL.Path, "/api/rule-packs/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing pack id"})
		return
	}
	pack, err := GetRulePack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

// GET /api/rule-packs/{id}/download — download (increment counter + return pack).
func handleDownloadRulePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	// path: /api/rule-packs/{id}/download
	path := strings.TrimPrefix(r.URL.Path, "/api/rule-packs/")
	id := strings.TrimSuffix(path, "/download")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing pack id"})
		return
	}
	pack, err := GetRulePack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	IncrementRulePackDownloads(id)
	pack.Downloads++
	writeJSON(w, http.StatusOK, pack)
}

// POST /api/rule-packs — publish a new rule pack (auth required).
func handlePublishRulePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	var req PublishRulePackReq
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
	pack := &RulePack{
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

	if err := InsertRulePack(pack); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to publish"})
		return
	}
	writeJSON(w, http.StatusCreated, pack)
}

// PUT /api/rule-packs/{id} — update own rule pack (auth required).
func handleUpdateRulePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	id := extractID(r.URL.Path, "/api/rule-packs/")
	existing, err := GetRulePack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	if existing.AuthorID != user.ID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "not your pack"})
		return
	}

	var req PublishRulePackReq
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

	if err := UpdateRulePack(existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update"})
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// DELETE /api/rule-packs/{id} — delete own rule pack (auth required).
func handleDeleteRulePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	user := currentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
		return
	}

	id := extractID(r.URL.Path, "/api/rule-packs/")
	existing, err := GetRulePack(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "pack not found"})
		return
	}
	if existing.AuthorID != user.ID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "not your pack"})
		return
	}

	if err := DeleteRulePack(id, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func extractID(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	// remove trailing slash or sub-paths
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}
