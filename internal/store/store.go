// Package store keeps Localert's private runtime data in an embedded SQLite database.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound      = errors.New("app key not found")
	ErrInvalidName   = errors.New("app name must contain 1–64 characters")
	ErrDuplicateName = errors.New("an app with this name already exists")
	ErrKeyLimit      = errors.New("maximum of 100 app keys reached; revoke an unused key first")
)

//go:embed schema.sql
var schema string

type Key struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	Sent       uint64     `json:"sent"`
}

type Event struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	AppName   string    `json:"app_name"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Icon      string    `json:"icon"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	Sent   uint64 `json:"sent"`
	Failed uint64 `json:"failed"`
}
type Snapshot struct {
	Keys         []Key    `json:"keys"`
	Events       []Event  `json:"events"`
	Stats        Stats    `json:"stats"`
	Settings     Settings `json:"settings"`
	HistoryTotal int      `json:"history_total"`
}

type Store struct {
	db   *sql.DB
	path string
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path, err := filepath.Abs(filepath.Join(dir, "localert.db"))
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(path)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	query := u.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(ON)")
	u.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, path: path}
	fail := func(err error) (*Store, error) { db.Close(); return nil, err }
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version > 2 {
		return fail(errors.New("database was created by a newer Localert version"))
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return fail(err)
	}
	if _, err := db.Exec(schema); err != nil {
		return fail(fmt.Errorf("initialize SQLite: %w", err))
	}
	if err := s.migrateJSON(dir); err != nil {
		return fail(err)
	}
	if err := s.PruneHistory(); err != nil {
		return fail(err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Path() string { return s.path }

func (s *Store) Snapshot() (Snapshot, error) {
	result := Snapshot{Keys: []Key{}, Events: []Event{}}
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	rows, err := tx.Query("SELECT id, name, prefix, created_at, last_used_at, sent FROM app_keys ORDER BY rowid DESC")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			rows.Close()
			return result, err
		}
		result.Keys = append(result.Keys, key)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.Query("SELECT id, app_id, app_name, title, message, icon, status, error, created_at FROM events ORDER BY sequence DESC LIMIT 100")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var event Event
		var created string
		if err := rows.Scan(&event.ID, &event.AppID, &event.AppName, &event.Title, &event.Message, &event.Icon, &event.Status, &event.Error, &created); err != nil {
			rows.Close()
			return result, err
		}
		event.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			rows.Close()
			return result, err
		}
		result.Events = append(result.Events, event)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if err := tx.QueryRow("SELECT sent, failed FROM stats WHERE id = 1").Scan(&result.Stats.Sent, &result.Stats.Failed); err != nil {
		return result, err
	}
	result.Settings, err = readSettings(tx)
	if err != nil {
		return result, err
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM events").Scan(&result.HistoryTotal); err != nil {
		return result, err
	}
	return result, tx.Commit()
}

func (s *Store) CreateKey(name string) (Key, string, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 64 || strings.ContainsAny(name, "\r\n\x00") {
		return Key{}, "", ErrInvalidName
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Key{}, "", err
	}
	defer tx.Rollback()
	rows, err := tx.Query("SELECT name FROM app_keys")
	if err != nil {
		return Key{}, "", err
	}
	count := 0
	for rows.Next() {
		var existing string
		if err := rows.Scan(&existing); err != nil {
			rows.Close()
			return Key{}, "", err
		}
		if strings.EqualFold(existing, name) {
			rows.Close()
			return Key{}, "", ErrDuplicateName
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Key{}, "", err
	}
	if count >= 100 {
		return Key{}, "", ErrKeyLimit
	}
	secret, err := randomToken("la_", 32)
	if err != nil {
		return Key{}, "", err
	}
	id, err := randomToken("app_", 12)
	if err != nil {
		return Key{}, "", err
	}
	key := Key{ID: id, Name: name, Prefix: secret[:11] + "…" + secret[len(secret)-4:], CreatedAt: time.Now().UTC()}
	if _, err := tx.Exec("INSERT INTO app_keys (id, name, prefix, secret_hash, created_at, sent) VALUES (?, ?, ?, ?, ?, 0)", key.ID, key.Name, key.Prefix, hash(secret), timestamp(key.CreatedAt)); err != nil {
		return Key{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return Key{}, "", err
	}
	return key, secret, nil
}

func (s *Store) Authenticate(secret string) (Key, bool, error) {
	if !strings.HasPrefix(secret, "la_") {
		return Key{}, false, nil
	}
	key, err := scanKey(s.db.QueryRow("SELECT id, name, prefix, created_at, last_used_at, sent FROM app_keys WHERE secret_hash = ?", hash(secret)))
	if errors.Is(err, sql.ErrNoRows) {
		return Key{}, false, nil
	}
	return key, err == nil, err
}

func (s *Store) RevokeKey(id string) error {
	result, err := s.db.Exec("DELETE FROM app_keys WHERE id = ?", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound
	}
	return err
}

// Record commits history and counters in the same transaction. Revoking an app
// during delivery does not discard its event or recreate the revoked key.
func (s *Store) Record(event Event) error {
	if event.ID == "" {
		var err error
		event.ID, err = NewEventID()
		if err != nil {
			return err
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertEvent(tx, event); err != nil {
		return err
	}
	sent, failed := 0, 0
	if event.Status == "sent" {
		sent = 1
	} else {
		failed = 1
	}
	if _, err := tx.Exec("UPDATE stats SET sent = sent + ?, failed = failed + ? WHERE id = 1", sent, failed); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE app_keys SET last_used_at = ?, sent = sent + ? WHERE id = ?", timestamp(event.CreatedAt), sent, event.AppID); err != nil {
		return err
	}
	if err := pruneHistory(tx, time.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClearHistory() error { _, err := s.db.Exec("DELETE FROM events"); return err }
func NewEventID() (string, error)    { return randomToken("evt_", 12) }

func insertEvent(tx *sql.Tx, event Event) error {
	_, err := tx.Exec("INSERT INTO events (id, app_id, app_name, title, message, icon, status, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", event.ID, event.AppID, event.AppName, event.Title, event.Message, event.Icon, event.Status, event.Error, timestamp(event.CreatedAt))
	return err
}

type scanner interface{ Scan(...any) error }

func scanKey(row scanner) (Key, error) {
	var key Key
	var created string
	var used sql.NullString
	if err := row.Scan(&key.ID, &key.Name, &key.Prefix, &created, &used, &key.Sent); err != nil {
		return key, err
	}
	var err error
	key.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return key, err
	}
	if used.Valid {
		t, err := time.Parse(time.RFC3339Nano, used.String)
		if err != nil {
			return key, err
		}
		key.LastUsedAt = &t
	}
	return key, nil
}

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func randomToken(prefix string, n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
