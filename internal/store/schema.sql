BEGIN;
CREATE TABLE IF NOT EXISTS metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS admin (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  password_hash TEXT NOT NULL,
  credential_version TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS app_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL,
  secret_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  last_used_at TEXT,
  sent INTEGER NOT NULL DEFAULT 0 CHECK (sent >= 0)
);
CREATE TABLE IF NOT EXISTS events (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  id TEXT NOT NULL UNIQUE,
  app_id TEXT NOT NULL,
  app_name TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT NOT NULL,
  icon TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
  error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS stats (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  sent INTEGER NOT NULL DEFAULT 0 CHECK (sent >= 0),
  failed INTEGER NOT NULL DEFAULT 0 CHECK (failed >= 0)
);
CREATE INDEX IF NOT EXISTS events_age ON events(julianday(created_at));
INSERT INTO stats (id, sent, failed) VALUES (1, 0, 0) ON CONFLICT (id) DO NOTHING;
CREATE TABLE IF NOT EXISTS preferences (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  retention_mode TEXT NOT NULL DEFAULT 'count',
  retention_count INTEGER NOT NULL DEFAULT 100,
  retention_days INTEGER NOT NULL DEFAULT 30,
  update_feed TEXT NOT NULL DEFAULT ''
);
INSERT INTO preferences (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
PRAGMA user_version = 2;
COMMIT;
