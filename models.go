package main

import "encoding/json"

// MemoRule represents a single rule within a Pack.
type MemoRule struct {
	Title      string `json:"title"`
	UpdateRule string `json:"update_rule"`
}

// Memo represents a single memo entry.
type Memo struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// RulePack is a unified publishable pack containing both rules and memos.
// (Renamed from RulePack but now includes memos too)
type RulePack struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	AuthorID     string     `json:"author_id"`
	AuthorName   string     `json:"author_name"`
	Version      string     `json:"version"`
	SystemPrompt string     `json:"system_prompt"`
	Rules        []MemoRule `json:"rules"`
	Memos        []Memo     `json:"memos"`
	Tags         []string   `json:"tags"`
	Downloads    int        `json:"downloads"`
	Published    bool       `json:"published"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// User represents a registered publisher.
type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Token       string `json:"token,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// ServerInfo describes this backend node (each node = one channel).
type ServerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// --- Request / Response types ---

type PublishRulePackReq struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Version      string     `json:"version"`
	SystemPrompt string     `json:"system_prompt"`
	Rules        []MemoRule `json:"rules"`
	Memos        []Memo     `json:"memos"`
	Tags         []string   `json:"tags"`
}

type RegisterReq struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type ListQuery struct {
	Search string
	Tag    string
	Author string
	Page   int
	Limit  int
}

type ListResponse struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- JSON marshal helpers for DB storage ---

func MarshalRules(rules []MemoRule) string {
	b, _ := json.Marshal(rules)
	return string(b)
}

func UnmarshalRules(s string) []MemoRule {
	var rules []MemoRule
	json.Unmarshal([]byte(s), &rules)
	if rules == nil {
		rules = []MemoRule{}
	}
	return rules
}

func MarshalMemos(memos []Memo) string {
	b, _ := json.Marshal(memos)
	return string(b)
}

func UnmarshalMemos(s string) []Memo {
	var memos []Memo
	json.Unmarshal([]byte(s), &memos)
	if memos == nil {
		memos = []Memo{}
	}
	return memos
}

func MarshalTags(tags []string) string {
	if tags == nil {
		tags = []string{}
	}
	b, _ := json.Marshal(tags)
	return string(b)
}

func UnmarshalTags(s string) []string {
	var tags []string
	json.Unmarshal([]byte(s), &tags)
	if tags == nil {
		tags = []string{}
	}
	return tags
}
