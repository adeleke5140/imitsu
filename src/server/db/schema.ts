import Database from "better-sqlite3";
import path from "path";

const DB_PATH = process.env.IMITSU_DB_PATH || path.join(process.cwd(), "imitsu.db");

let db: Database.Database;

export function getDb(): Database.Database {
  if (!db) {
    db = new Database(DB_PATH);
    db.pragma("journal_mode = WAL");
    db.pragma("foreign_keys = ON");
    initSchema(db);
  }
  return db;
}

function initSchema(db: Database.Database) {
  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      id TEXT PRIMARY KEY,
      email TEXT UNIQUE NOT NULL,
      name TEXT NOT NULL,
      password_hash TEXT NOT NULL,
      role TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('admin', 'member')),
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      updated_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS secrets (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      encrypted_value TEXT NOT NULL,
      iv TEXT NOT NULL,
      auth_tag TEXT NOT NULL,
      salt TEXT NOT NULL,
      category TEXT DEFAULT 'general',
      created_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      updated_at TEXT NOT NULL DEFAULT (datetime('now')),
      expires_at TEXT,
      version INTEGER NOT NULL DEFAULT 1
    );

    CREATE TABLE IF NOT EXISTS secret_versions (
      id TEXT PRIMARY KEY,
      secret_id TEXT NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
      encrypted_value TEXT NOT NULL,
      iv TEXT NOT NULL,
      auth_tag TEXT NOT NULL,
      salt TEXT NOT NULL,
      version INTEGER NOT NULL,
      created_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS secret_access (
      id TEXT PRIMARY KEY,
      secret_id TEXT NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
      user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      permission TEXT NOT NULL DEFAULT 'read' CHECK(permission IN ('read', 'write', 'admin')),
      granted_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      UNIQUE(secret_id, user_id)
    );

    CREATE TABLE IF NOT EXISTS audit_log (
      id TEXT PRIMARY KEY,
      user_id TEXT REFERENCES users(id),
      action TEXT NOT NULL,
      resource_type TEXT NOT NULL,
      resource_id TEXT,
      details TEXT,
      ip_address TEXT,
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS teams (
      id TEXT PRIMARY KEY,
      name TEXT UNIQUE NOT NULL,
      created_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS team_members (
      id TEXT PRIMARY KEY,
      team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
      user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      role TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('admin', 'member')),
      added_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      UNIQUE(team_id, user_id)
    );

    CREATE TABLE IF NOT EXISTS secret_team_access (
      id TEXT PRIMARY KEY,
      secret_id TEXT NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
      team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
      permission TEXT NOT NULL DEFAULT 'read' CHECK(permission IN ('read', 'write', 'admin')),
      granted_by TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      UNIQUE(secret_id, team_id)
    );

    CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);
    CREATE INDEX IF NOT EXISTS idx_team_members_team ON team_members(team_id);
    CREATE INDEX IF NOT EXISTS idx_secret_team_access_secret ON secret_team_access(secret_id);
    CREATE INDEX IF NOT EXISTS idx_secret_team_access_team ON secret_team_access(team_id);
    CREATE INDEX IF NOT EXISTS idx_secrets_created_by ON secrets(created_by);
    CREATE INDEX IF NOT EXISTS idx_secret_access_user ON secret_access(user_id);
    CREATE INDEX IF NOT EXISTS idx_secret_access_secret ON secret_access(secret_id);
    CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
    CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at);
  `);
}
