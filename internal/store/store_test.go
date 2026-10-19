package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func openTestStore(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func snapshot(t *testing.T, s *Store) Snapshot {
	t.Helper()
	result, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSQLitePasswordSetupPersistsAndCannotBeRepeated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "user's data & # π")
	s := openTestStore(t, dir)
	status, err := s.SetupStatus()
	if err != nil || !status.SetupRequired || status.LegacyTokenRequired {
		t.Fatalf("fresh setup: %+v %v", status, err)
	}
	if err := s.Setup("short", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("weak password accepted: %v", err)
	}
	password := "correct horse battery staple"
	if err := s.Setup(password, ""); err != nil {
		t.Fatal(err)
	}
	version, valid, err := s.VerifyPassword(password)
	if err != nil || !valid || version == "" {
		t.Fatalf("password rejected: %v", err)
	}
	if _, valid, err := s.VerifyPassword("wrong"); err != nil || valid {
		t.Fatal("wrong password accepted")
	}
	if err := s.Setup("replacement password", ""); !errors.Is(err, ErrAlreadyConfigured) {
		t.Fatalf("setup replaced existing password: %v", err)
	}
	var encoded string
	if err := s.db.QueryRow("SELECT password_hash FROM admin").Scan(&encoded); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "pbkdf2-sha256$600000$") || strings.Contains(encoded, password) {
		t.Fatal("password was not securely hashed")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "localert.db"))
	if err != nil || !bytes.HasPrefix(data, []byte("SQLite format 3\x00")) {
		t.Fatal("runtime state is not a SQLite database")
	}
	if bytes.Contains(data, []byte(password)) {
		t.Fatal("database contains plaintext password")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(s.Path())
		if info.Mode().Perm() != 0600 {
			t.Fatalf("private database permissions: %v", info.Mode())
		}
	}
	s = openTestStore(t, dir)
	if _, valid, err := s.VerifyPassword(password); err != nil || !valid {
		t.Fatal("password did not survive restart")
	}
}

func TestKeysPersistAsHashesAndRevokeIndividually(t *testing.T) {
	dir := t.TempDir()
	s := openTestStore(t, dir)
	first, secret, err := s.CreateKey("Build pipeline")
	if err != nil {
		t.Fatal(err)
	}
	_, secondSecret, err := s.CreateKey("Backups")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateKey(" build PIPELINE "); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected duplicate name, got %v", err)
	}
	var stored string
	if err := s.db.QueryRow("SELECT secret_hash FROM app_keys WHERE id = ?", first.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != hash(secret) {
		t.Fatal("app key was not stored as a hash")
	}
	public, _ := json.Marshal(snapshot(t, s))
	if strings.Contains(string(public), "hash") || strings.Contains(string(public), secret) {
		t.Fatal("public snapshot exposes credentials")
	}
	s.Close()
	s = openTestStore(t, dir)
	if key, ok, err := s.Authenticate(secret); err != nil || !ok || key.ID != first.ID {
		t.Fatal("app key did not persist")
	}
	if err := s.RevokeKey(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(secret); ok {
		t.Fatal("revoked key still works")
	}
	if _, ok, err := s.Authenticate(secondSecret); err != nil || !ok {
		t.Fatal("revoking a key affected another app")
	}
	s.Close()
	s = openTestStore(t, dir)
	if _, ok, _ := s.Authenticate(secret); ok {
		t.Fatal("revocation did not persist")
	}
}

func TestPasswordChangeAndRecoveryKeepAppKeys(t *testing.T) {
	s := openTestStore(t, t.TempDir())
	if err := s.Setup("original password", ""); err != nil {
		t.Fatal(err)
	}
	before, _ := s.CredentialVersion()
	_, secret, err := s.CreateKey("Always running")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword("incorrect", "replacement password"); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatal("incorrect password changed credentials")
	}
	if err := s.ChangePassword("original password", "replacement password"); err != nil {
		t.Fatal(err)
	}
	after, _ := s.CredentialVersion()
	if before == after {
		t.Fatal("password change did not rotate credential version")
	}
	if _, valid, _ := s.VerifyPassword("original password"); valid {
		t.Fatal("old password still works")
	}
	if _, valid, _ := s.VerifyPassword("replacement password"); !valid {
		t.Fatal("new password rejected")
	}
	if err := s.ResetPassword(); err != nil {
		t.Fatal(err)
	}
	if version, _ := s.CredentialVersion(); version != "" {
		t.Fatal("recovery did not clear dashboard credentials")
	}
	if _, valid, _ := s.Authenticate(secret); !valid {
		t.Fatal("password recovery discarded app key")
	}
	status, err := s.SetupStatus()
	if err != nil || !status.SetupRequired {
		t.Fatal("recovery did not re-enable setup")
	}
	if err := s.Setup("recovered password", ""); err != nil {
		t.Fatal(err)
	}
	recovered, _ := s.CredentialVersion()
	if recovered == before || recovered == after {
		t.Fatal("recovery reused credential version")
	}
}

func TestConcurrentHistoryIsBoundedAndCountersSurviveClearing(t *testing.T) {
	s := openTestStore(t, t.TempDir())
	key, _, err := s.CreateKey("Worker")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 104; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status := "sent"
			if i%2 == 0 {
				status = "failed"
			}
			if err := s.Record(Event{AppID: key.ID, Status: status, CreatedAt: time.Now().UTC()}); err != nil {
				t.Errorf("record: %v", err)
			}
		}(i)
	}
	wg.Wait()
	result := snapshot(t, s)
	if len(result.Events) != 100 || result.Stats.Sent != 52 || result.Stats.Failed != 52 || result.Keys[0].Sent != 52 || result.Keys[0].LastUsedAt == nil {
		t.Fatalf("incorrect concurrent counters: %+v", result.Stats)
	}
	if err := s.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	result = snapshot(t, s)
	if len(result.Events) != 0 || result.Stats.Sent != 52 || len(result.Keys) != 1 {
		t.Fatal("clearing history modified counters or keys")
	}
}

func TestFailedTransactionRollsBackHistoryAndCounters(t *testing.T) {
	s := openTestStore(t, t.TempDir())
	key, _, err := s.CreateKey("Worker")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("CREATE TRIGGER reject_usage BEFORE UPDATE ON app_keys BEGIN SELECT RAISE(ABORT, 'test write failure'); END"); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(Event{AppID: key.ID, Status: "sent", CreatedAt: time.Now()}); err == nil {
		t.Fatal("expected persistence failure")
	}
	result := snapshot(t, s)
	if len(result.Events) != 0 || result.Stats.Sent != 0 || result.Keys[0].Sent != 0 {
		t.Fatal("partial notification transaction was committed")
	}
}

func TestLegacyMigrationPreservesKeysAndDoesNotReplay(t *testing.T) {
	dir := t.TempDir()
	secret, token := "la_existing_app_secret", "la_admin_old_token"
	key := Key{ID: "app_existing", Name: "Existing app", Prefix: "la_existing…cret", CreatedAt: time.Now().UTC(), Sent: 7}
	old := map[string]any{
		"version": 1, "admin_token": token,
		"keys": []any{struct {
			Key
			Hash string `json:"hash"`
		}{key, hash(secret)}},
		"events": []Event{{ID: "evt_existing", AppID: key.ID, AppName: key.Name, Title: "Existing event", Status: "sent", CreatedAt: time.Now().UTC()}},
		"stats":  Stats{Sent: 7, Failed: 1},
	}
	data, _ := json.Marshal(old)
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s := openTestStore(t, dir)
	result := snapshot(t, s)
	if len(result.Keys) != 1 || len(result.Events) != 1 || result.Stats.Sent != 7 || result.Keys[0].Sent != 7 {
		t.Fatal("legacy data was not preserved")
	}
	if _, valid, _ := s.Authenticate(secret); !valid {
		t.Fatal("legacy app key stopped working")
	}
	if err := s.Setup("new dashboard password", "wrong"); !errors.Is(err, ErrLegacyToken) {
		t.Fatal("migration setup did not check the previous admin token")
	}
	if err := s.Setup("new dashboard password", token); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeKey(key.ID); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s = openTestStore(t, dir)
	if len(snapshot(t, s).Keys) != 0 {
		t.Fatal("migration replay resurrected a revoked key")
	}
	if _, valid, _ := s.VerifyPassword("new dashboard password"); !valid {
		t.Fatal("new dashboard password did not persist")
	}
	backup, _ := os.ReadFile(path)
	if !bytes.Equal(backup, data) {
		t.Fatal("migration modified the original JSON backup")
	}
}

func TestCorruptLegacyFileIsLeftUntouched(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("invalid JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(dir); err == nil {
		s.Close()
		t.Fatal("corrupt legacy data was silently discarded")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "invalid JSON" {
		t.Fatal("corrupt legacy data was overwritten")
	}
}
