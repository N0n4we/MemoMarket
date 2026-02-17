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

	CREATE TABLE IF NOT EXISTS memo_packs (
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

	CREATE INDEX IF NOT EXISTS idx_memo_packs_author ON memo_packs(author_id);
	CREATE INDEX IF NOT EXISTS idx_memo_packs_published ON memo_packs(published);
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

// ---- MemoPack DB operations ----

func InsertMemoPack(mp *MemoPack) error {
	_, err := db.Exec(
		`INSERT INTO memo_packs (id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mp.ID, mp.Name, mp.Description, mp.AuthorID, mp.AuthorName, mp.Version,
		mp.SystemPrompt, MarshalRules(mp.Rules), MarshalMemos(mp.Memos), MarshalTags(mp.Tags),
		mp.Downloads, boolToInt(mp.Published), mp.CreatedAt, mp.UpdatedAt,
	)
	return err
}

func UpdateMemoPack(mp *MemoPack) error {
	_, err := db.Exec(
		`UPDATE memo_packs SET name=?, description=?, version=?, system_prompt=?, rules=?, memos=?, tags=?, published=?, updated_at=?
		 WHERE id=? AND author_id=?`,
		mp.Name, mp.Description, mp.Version, mp.SystemPrompt,
		MarshalRules(mp.Rules), MarshalMemos(mp.Memos), MarshalTags(mp.Tags), boolToInt(mp.Published), nowISO(),
		mp.ID, mp.AuthorID,
	)
	return err
}

func DeleteMemoPack(id, authorID string) error {
	_, err := db.Exec(`DELETE FROM memo_packs WHERE id=? AND author_id=?`, id, authorID)
	return err
}

func GetMemoPack(id string) (*MemoPack, error) {
	var mp MemoPack
	var rulesJSON, memosJSON, tagsJSON string
	var published int
	err := db.QueryRow(
		`SELECT id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at
		 FROM memo_packs WHERE id=?`, id,
	).Scan(&mp.ID, &mp.Name, &mp.Description, &mp.AuthorID, &mp.AuthorName, &mp.Version,
		&mp.SystemPrompt, &rulesJSON, &memosJSON, &tagsJSON, &mp.Downloads, &published, &mp.CreatedAt, &mp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	mp.Rules = UnmarshalRules(rulesJSON)
	mp.Memos = UnmarshalMemos(memosJSON)
	mp.Tags = UnmarshalTags(tagsJSON)
	mp.Published = published == 1
	return &mp, nil
}

func ListMemoPacks(q ListQuery) ([]MemoPack, int, error) {
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
	err := db.QueryRow("SELECT COUNT(*) FROM memo_packs WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (q.Page - 1) * q.Limit
	rows, err := db.Query(
		"SELECT id, name, description, author_id, author_name, version, system_prompt, rules, memos, tags, downloads, published, created_at, updated_at FROM memo_packs WHERE "+whereClause+" ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		append(args, q.Limit, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packs []MemoPack
	for rows.Next() {
		var mp MemoPack
		var rulesJSON, memosJSON, tagsJSON string
		var published int
		rows.Scan(&mp.ID, &mp.Name, &mp.Description, &mp.AuthorID, &mp.AuthorName, &mp.Version,
			&mp.SystemPrompt, &rulesJSON, &memosJSON, &tagsJSON, &mp.Downloads, &published, &mp.CreatedAt, &mp.UpdatedAt)
		mp.Rules = UnmarshalRules(rulesJSON)
		mp.Memos = UnmarshalMemos(memosJSON)
		mp.Tags = UnmarshalTags(tagsJSON)
		mp.Published = published == 1
		packs = append(packs, mp)
	}
	if packs == nil {
		packs = []MemoPack{}
	}
	return packs, total, nil
}

func IncrementMemoPackDownloads(id string) error {
	_, err := db.Exec(`UPDATE memo_packs SET downloads = downloads + 1 WHERE id = ?`, id)
	return err
}

// ---- helpers ----

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
