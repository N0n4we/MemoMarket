package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB(dataDir string) {
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "memomarket.db")

	var err error
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	migrate()
}

func migrate() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		token TEXT UNIQUE NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS rule_packs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		author_id TEXT NOT NULL,
		author_name TEXT NOT NULL DEFAULT '',
		version TEXT NOT NULL DEFAULT '1.0.0',
		system_prompt TEXT NOT NULL DEFAULT '',
		rules TEXT NOT NULL DEFAULT '[]',
		memos TEXT NOT NULL DEFAULT '[]',
		tags TEXT NOT NULL DEFAULT '[]',
		downloads INTEGER NOT NULL DEFAULT 0,
		published INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (author_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_rule_packs_author ON rule_packs(author_id);
	CREATE INDEX IF NOT EXISTS idx_rule_packs_published ON rule_packs(published);
	`
	_, err := db.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05")
}

func newID() string {
	return uuid.New().String()
}

// ---- User DB operations ----

func CreateUser(username, displayName string) (*User, error) {
	id := newID()
	token := uuid.New().String()
	now := nowISO()

	_, err := db.Exec(
		`INSERT INTO users (id, username, display_name, token, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, username, displayName, token, now,
	)
	if err != nil {
		return nil, fmt.Errorf("username already taken")
	}
	return &User{ID: id, Username: username, DisplayName: displayName, Token: token, CreatedAt: now}, nil
}

func GetUserByToken(token string) (*User, error) {
	var u User
	err := db.QueryRow(
		`SELECT id, username, display_name, token, created_at FROM users WHERE token = ?`, token,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Token, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(id string) (*User, error) {
	var u User
	err := db.QueryRow(
		`SELECT id, username, display_name, '', created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Token, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ---- RulePack DB operations ----

func InsertRulePack(rp *RulePack) error {
	_, err := db.Exec(
		`INSERT INTO rule_packs (id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rp.ID, rp.Name, rp.Description, rp.AuthorID, rp.AuthorName, rp.Version,
		rp.SystemPrompt, MarshalRules(rp.Rules), MarshalMemos(rp.Memos), MarshalTags(rp.Tags),
		rp.Downloads, boolToInt(rp.Published), rp.CreatedAt, rp.UpdatedAt,
	)
	return err
}

func UpdateRulePack(rp *RulePack) error {
	_, err := db.Exec(
		`UPDATE rule_packs SET name=?, description=?, version=?, system_prompt=?, rules=?, memos=?, tags=?, published=?, updated_at=?
		 WHERE id=? AND author_id=?`,
		rp.Name, rp.Description, rp.Version, rp.SystemPrompt,
		MarshalRules(rp.Rules), MarshalMemos(rp.Memos), MarshalTags(rp.Tags), boolToInt(rp.Published), nowISO(),
		rp.ID, rp.AuthorID,
	)
	return err
}

func DeleteRulePack(id, authorID string) error {
	_, err := db.Exec(`DELETE FROM rule_packs WHERE id=? AND author_id=?`, id, authorID)
	return err
}

func GetRulePack(id string) (*RulePack, error) {
	var rp RulePack
	var rulesJSON, memosJSON, tagsJSON string
	var published int
	err := db.QueryRow(
		`SELECT id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at
		 FROM rule_packs WHERE id=?`, id,
	).Scan(&rp.ID, &rp.Name, &rp.Description, &rp.AuthorID, &rp.AuthorName, &rp.Version,
		&rp.SystemPrompt, &rulesJSON, &memosJSON, &tagsJSON, &rp.Downloads, &published, &rp.CreatedAt, &rp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rp.Rules = UnmarshalRules(rulesJSON)
	rp.Memos = UnmarshalMemos(memosJSON)
	rp.Tags = UnmarshalTags(tagsJSON)
	rp.Published = published == 1
	return &rp, nil
}

func ListRulePacks(q ListQuery) ([]RulePack, int, error) {
	where := []string{"published = 1"}
	args := []any{}

	if q.Search != "" {
		where = append(where, "(name LIKE ? OR description LIKE ? OR author_name LIKE ?)")
		s := "%" + q.Search + "%"
		args = append(args, s, s, s)
	}
	if q.Tag != "" {
		where = append(where, "tags LIKE ?")
		args = append(args, "%\""+q.Tag+"\"%")
	}
	if q.Author != "" {
		where = append(where, "author_id = ?")
		args = append(args, q.Author)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM rule_packs WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (q.Page - 1) * q.Limit
	rows, err := db.Query(
		"SELECT id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at FROM rule_packs WHERE "+whereClause+" ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		append(args, q.Limit, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packs []RulePack
	for rows.Next() {
		var rp RulePack
		var rulesJSON, memosJSON, tagsJSON string
		var published int
		rows.Scan(&rp.ID, &rp.Name, &rp.Description, &rp.AuthorID, &rp.AuthorName, &rp.Version,
			&rp.SystemPrompt, &rulesJSON, &memosJSON, &tagsJSON, &rp.Downloads, &published, &rp.CreatedAt, &rp.UpdatedAt)
		rp.Rules = UnmarshalRules(rulesJSON)
		rp.Memos = UnmarshalMemos(memosJSON)
		rp.Tags = UnmarshalTags(tagsJSON)
		rp.Published = published == 1
		packs = append(packs, rp)
	}
	if packs == nil {
		packs = []RulePack{}
	}
	return packs, total, nil
}

func IncrementRulePackDownloads(id string) error {
	_, err := db.Exec(`UPDATE rule_packs SET downloads = downloads + 1 WHERE id = ?`, id)
	return err
}

// ---- helpers ----

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
