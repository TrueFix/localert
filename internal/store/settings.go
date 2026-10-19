package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidRetention = errors.New("choose count, days, both, or forever; count must be 1–100000 and days must be 1–3650")

type Retention struct {
	Mode  string `json:"mode"`
	Count int    `json:"count"`
	Days  int    `json:"days"`
}

type Settings struct {
	Retention  Retention `json:"retention"`
	UpdateFeed string    `json:"update_feed"`
}

type queryer interface{ QueryRow(string, ...any) *sql.Row }

func readSettings(q queryer) (Settings, error) {
	var result Settings
	err := q.QueryRow("SELECT retention_mode, retention_count, retention_days, update_feed FROM preferences WHERE id = 1").Scan(&result.Retention.Mode, &result.Retention.Count, &result.Retention.Days, &result.UpdateFeed)
	return result, err
}

func (s *Store) Settings() (Settings, error) { return readSettings(s.db) }

func (s *Store) SetRetention(value Retention) error {
	if (value.Mode != "count" && value.Mode != "days" && value.Mode != "both" && value.Mode != "forever") || value.Count < 1 || value.Count > 100000 || value.Days < 1 || value.Days > 3650 {
		return ErrInvalidRetention
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE preferences SET retention_mode = ?, retention_count = ?, retention_days = ? WHERE id = 1", value.Mode, value.Count, value.Days); err != nil {
		return err
	}
	if err := pruneHistory(tx, time.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetUpdateFeed(value string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE preferences SET update_feed = ? WHERE id = 1", value); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO metadata (key, value) VALUES ('update_source_chosen', '1') ON CONFLICT (key) DO NOTHING"); err != nil {
		return err
	}
	return tx.Commit()
}

// InitializeUpdateFeed lets a publisher preconfigure a feed in the distributed
// binary. A user's saved choice (including disabling updates) always wins.
func (s *Store) InitializeUpdateFeed(value string) error {
	if value == "" {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var chosen int
	if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM metadata WHERE key = 'update_source_chosen')").Scan(&chosen); err != nil {
		return err
	}
	if chosen != 0 {
		return nil
	}
	if _, err := tx.Exec("UPDATE preferences SET update_feed = ? WHERE id = 1 AND update_feed = ''", value); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO metadata (key, value) VALUES ('update_source_chosen', '1')"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) PruneHistory() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := pruneHistory(tx, time.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

func pruneHistory(tx *sql.Tx, now time.Time) error {
	settings, err := readSettings(tx)
	if err != nil {
		return err
	}
	r := settings.Retention
	if r.Mode == "days" || r.Mode == "both" {
		if _, err := tx.Exec("DELETE FROM events WHERE julianday(created_at) < julianday(?)", timestamp(now.Add(-time.Duration(r.Days)*24*time.Hour))); err != nil {
			return err
		}
	}
	if r.Mode == "count" || r.Mode == "both" {
		if _, err := tx.Exec("DELETE FROM events WHERE sequence <= (SELECT sequence FROM events ORDER BY sequence DESC LIMIT 1 OFFSET ?)", r.Count); err != nil {
			return err
		}
	}
	return nil
}

type HistoryPage struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

func (s *Store) History(limit, offset int, status, search string) (HistoryPage, error) {
	result := HistoryPage{Events: []Event{}, Limit: limit, Offset: offset}
	if limit < 1 || limit > 100 || offset < 0 || (status != "" && status != "sent" && status != "failed") {
		return result, errors.New("invalid history pagination or status")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	where := " WHERE (? = '' OR status = ?) AND (? = '' OR instr(lower(title || ' ' || message || ' ' || app_name), lower(?)) > 0)"
	if err := tx.QueryRow("SELECT COUNT(*) FROM events"+where, status, status, search, search).Scan(&result.Total); err != nil {
		return result, err
	}
	rows, err := tx.Query("SELECT id, app_id, app_name, title, message, icon, status, error, created_at FROM events"+where+" ORDER BY sequence DESC LIMIT ? OFFSET ?", status, status, search, search, limit, offset)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var e Event
		var created string
		if err := rows.Scan(&e.ID, &e.AppID, &e.AppName, &e.Title, &e.Message, &e.Icon, &e.Status, &e.Error, &created); err != nil {
			rows.Close()
			return result, err
		}
		e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			rows.Close()
			return result, err
		}
		result.Events = append(result.Events, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
