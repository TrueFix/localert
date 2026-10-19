package store

import (
	"errors"
	"testing"
	"time"
)

func TestRetentionModesPruneWithoutChangingAllTimeCounters(t *testing.T) {
	dir := t.TempDir()
	s := openTestStore(t, dir)
	if err := s.SetRetention(Retention{Mode: "forever", Count: 3, Days: 7}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		when := time.Now().Add(-time.Duration(i) * 48 * time.Hour)
		if err := s.Record(Event{Title: "Memory", Status: "sent", CreatedAt: when}); err != nil {
			t.Fatal(err)
		}
	}
	if got := snapshot(t, s); got.HistoryTotal != 8 || got.Stats.Sent != 8 {
		t.Fatalf("forever did not retain all events: %+v", got.Stats)
	}
	if err := s.SetRetention(Retention{Mode: "days", Count: 3, Days: 7}); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, s); got.HistoryTotal != 4 || got.Stats.Sent != 8 {
		t.Fatalf("age retention: %d events, %+v", got.HistoryTotal, got.Stats)
	}
	if err := s.SetRetention(Retention{Mode: "both", Count: 2, Days: 7}); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, s); got.HistoryTotal != 2 || got.Stats.Sent != 8 {
		t.Fatalf("combined retention: %d events", got.HistoryTotal)
	}
	if err := s.SetUpdateFeed("https://releases.example/update-manifest.json"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s = openTestStore(t, dir)
	settings, err := s.Settings()
	if err != nil || settings.Retention.Mode != "both" || settings.Retention.Count != 2 || settings.UpdateFeed == "" {
		t.Fatalf("settings did not persist: %+v, %v", settings, err)
	}
	if err := s.SetRetention(Retention{Mode: "unknown", Count: 2, Days: 7}); !errors.Is(err, ErrInvalidRetention) {
		t.Fatal("invalid settings were accepted")
	}
}

func TestHistoryPaginationSearchAndCountLimit(t *testing.T) {
	s := openTestStore(t, t.TempDir())
	if err := s.SetRetention(Retention{Mode: "count", Count: 130, Days: 30}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 135; i++ {
		if err := s.Record(Event{Title: "Build ready", AppName: "My builder", Status: "sent", CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if got := snapshot(t, s); len(got.Events) != 100 || got.HistoryTotal != 130 {
		t.Fatalf("overview/page distinction: %d / %d", len(got.Events), got.HistoryTotal)
	}
	page, err := s.History(25, 125, "sent", "BUILDER")
	if err != nil || len(page.Events) != 5 || page.Total != 130 {
		t.Fatalf("history pagination: %+v %v", page, err)
	}
	page, err = s.History(25, 0, "failed", "")
	if err != nil || page.Total != 0 {
		t.Fatalf("status filter: %+v %v", page, err)
	}
	if err := s.SetRetention(Retention{Mode: "days", Count: 130, Days: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE events SET created_at = ?", timestamp(time.Now().Add(-48*time.Hour))); err != nil {
		t.Fatal(err)
	}
	if err := s.PruneHistory(); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, s); got.HistoryTotal != 0 || got.Stats.Sent != 135 {
		t.Fatal("scheduled age pruning changed counters or left expired events")
	}
}

func TestSchemaOneUpgradePreservesCredentialsAndKeys(t *testing.T) {
	dir := t.TempDir()
	s := openTestStore(t, dir)
	_, secret, err := s.CreateKey("Existing app")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Setup("existing password", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("DROP TABLE preferences; PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s = openTestStore(t, dir)
	if _, valid, err := s.Authenticate(secret); err != nil || !valid {
		t.Fatal("schema upgrade discarded key")
	}
	if _, valid, err := s.VerifyPassword("existing password"); err != nil || !valid {
		t.Fatal("schema upgrade discarded password")
	}
	settings, err := s.Settings()
	if err != nil || settings.Retention.Mode != "count" || settings.Retention.Count != 100 {
		t.Fatalf("default settings not migrated: %+v %v", settings, err)
	}
}

func TestPublisherFeedNeverOverridesUsersChoice(t *testing.T) {
	s := openTestStore(t, t.TempDir())
	if err := s.InitializeUpdateFeed("https://publisher.example/feed.json"); err != nil {
		t.Fatal(err)
	}
	settings, _ := s.Settings()
	if settings.UpdateFeed != "https://publisher.example/feed.json" {
		t.Fatal("publisher default was not applied")
	}
	if err := s.SetUpdateFeed(""); err != nil {
		t.Fatal(err)
	}
	if err := s.InitializeUpdateFeed("https://another.example/feed.json"); err != nil {
		t.Fatal(err)
	}
	settings, _ = s.Settings()
	if settings.UpdateFeed != "" {
		t.Fatal("publisher default overrode disabled updates")
	}
}
