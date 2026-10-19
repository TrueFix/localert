package store

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const passwordIterations = 600_000

var (
	ErrAlreadyConfigured = errors.New("Localert is already configured; sign in with your password")
	ErrInvalidPassword   = errors.New("password must contain 8–128 characters")
	ErrIncorrectPassword = errors.New("incorrect current password")
	ErrLegacyToken       = errors.New("enter the previous admin token to finish upgrading this installation")
)

type SetupStatus struct {
	SetupRequired       bool `json:"setup_required"`
	LegacyTokenRequired bool `json:"legacy_token_required"`
	PasswordRecovery    bool `json:"password_recovery"`
}

func (s *Store) SetupStatus() (SetupStatus, error) {
	var configured, legacy, recovery int
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin WHERE id = 1),
		EXISTS(SELECT 1 FROM metadata WHERE key = 'legacy_admin_hash'),
		EXISTS(SELECT 1 FROM metadata WHERE key = 'password_recovery' AND value = '1')`).Scan(&configured, &legacy, &recovery)
	return SetupStatus{
		SetupRequired:       configured == 0,
		LegacyTokenRequired: legacy != 0 && configured == 0,
		PasswordRecovery:    configured == 0 && recovery != 0,
	}, err
}

func (s *Store) Setup(password, legacyToken string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	status, err := s.SetupStatus()
	if err != nil {
		return err
	}
	if !status.SetupRequired {
		return ErrAlreadyConfigured
	}
	if status.LegacyTokenRequired {
		var expected string
		if err := s.db.QueryRow("SELECT value FROM metadata WHERE key = 'legacy_admin_hash'").Scan(&expected); err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(hash(legacyToken))) != 1 {
			return ErrLegacyToken
		}
	}
	encoded, err := passwordHash(password)
	if err != nil {
		return err
	}
	version, err := randomToken("", 16)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec("INSERT INTO admin (id, password_hash, credential_version, updated_at) VALUES (1, ?, ?, ?) ON CONFLICT (id) DO NOTHING", encoded, version, timestamp(time.Now()))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrAlreadyConfigured
	}
	if _, err := tx.Exec("DELETE FROM metadata WHERE key IN ('legacy_admin_hash', 'password_recovery')"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) VerifyPassword(password string) (version string, valid bool, err error) {
	var encoded string
	err = s.db.QueryRow("SELECT password_hash, credential_version FROM admin WHERE id = 1").Scan(&encoded, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	valid, err = matchesPassword(password, encoded)
	return version, valid, err
}

func (s *Store) CredentialVersion() (string, error) {
	var version string
	err := s.db.QueryRow("SELECT credential_version FROM admin WHERE id = 1").Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return version, err
}

func (s *Store) ChangePassword(current, replacement string) error {
	if err := validatePassword(replacement); err != nil {
		return err
	}
	version, valid, err := s.VerifyPassword(current)
	if err != nil {
		return err
	}
	if !valid {
		return ErrIncorrectPassword
	}
	encoded, err := passwordHash(replacement)
	if err != nil {
		return err
	}
	nextVersion, err := randomToken("", 16)
	if err != nil {
		return err
	}
	result, err := s.db.Exec("UPDATE admin SET password_hash = ?, credential_version = ?, updated_at = ? WHERE id = 1 AND credential_version = ?", encoded, nextVersion, timestamp(time.Now()), version)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrIncorrectPassword
	}
	return err
}

// ResetPassword is a local CLI recovery operation. It retains all app keys and
// events; the next setup is restricted to a loopback connection by the server.
func (s *Store) ResetPassword() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM admin"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM metadata WHERE key = 'legacy_admin_hash'"); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO metadata (key, value) VALUES ('password_recovery', '1')
		ON CONFLICT(key) DO UPDATE SET value = '1'`); err != nil {
		return err
	}
	return tx.Commit()
}

func validatePassword(password string) error {
	if count := utf8.RuneCountInString(password); count < 8 || count > 128 || !utf8.ValidString(password) {
		return ErrInvalidPassword
	}
	return nil
}

func passwordHash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func matchesPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false, errors.New("unsupported stored password format")
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 || iterations > 1_000_000 {
		return false, errors.New("invalid stored password parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		return false, errors.New("invalid stored password salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(expected) != 32 {
		return false, errors.New("invalid stored password hash")
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(expected))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
