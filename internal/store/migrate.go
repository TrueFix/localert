package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// migrateJSON runs exactly once, inside a transaction. The original file remains
// untouched as a backup; a completed migration is never replayed after revocation.
func (s *Store) migrateJSON(dir string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var checked string
	err = tx.QueryRow("SELECT value FROM metadata WHERE key = 'legacy_import_checked'").Scan(&checked)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	path := filepath.Join(dir, "state.json")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		var old struct {
			Version    int    `json:"version"`
			AdminToken string `json:"admin_token"`
			Keys       []struct {
				Key
				Hash string `json:"hash"`
			} `json:"keys"`
			Events []Event `json:"events"`
			Stats  Stats   `json:"stats"`
		}
		if err := json.Unmarshal(data, &old); err != nil {
			return fmt.Errorf("read legacy state (original file left untouched): %w", err)
		}
		if old.Version != 1 || old.AdminToken == "" {
			return errors.New("unsupported legacy state (original file left untouched)")
		}
		for _, key := range old.Keys {
			var used any
			if key.LastUsedAt != nil {
				used = timestamp(*key.LastUsedAt)
			}
			if _, err := tx.Exec("INSERT INTO app_keys (id, name, prefix, secret_hash, created_at, last_used_at, sent) VALUES (?, ?, ?, ?, ?, ?, ?)", key.ID, key.Name, key.Prefix, key.Hash, timestamp(key.CreatedAt), used, key.Sent); err != nil {
				return fmt.Errorf("migrate app keys: %w", err)
			}
		}
		limit := min(len(old.Events), 100)
		for i := limit - 1; i >= 0; i-- {
			if err := insertEvent(tx, old.Events[i]); err != nil {
				return fmt.Errorf("migrate history: %w", err)
			}
		}
		if _, err := tx.Exec("UPDATE stats SET sent = ?, failed = ? WHERE id = 1", old.Stats.Sent, old.Stats.Failed); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO metadata (key, value) VALUES ('legacy_admin_hash', ?)", hash(old.AdminToken)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("INSERT INTO metadata (key, value) VALUES ('legacy_import_checked', '1')"); err != nil {
		return err
	}
	return tx.Commit()
}
