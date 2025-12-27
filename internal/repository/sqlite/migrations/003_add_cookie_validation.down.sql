-- Rollback cookie validation fields

DROP INDEX IF EXISTS idx_accounts_validation_status;

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE accounts_backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform TEXT NOT NULL,
    name TEXT NOT NULL,
    cookie_path TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 0,
    last_used INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(platform, name)
);

INSERT INTO accounts_backup (id, platform, name, cookie_path, is_active, last_used, created_at)
SELECT id, platform, name, cookie_path, is_active, last_used, created_at
FROM accounts;

DROP TABLE accounts;
ALTER TABLE accounts_backup RENAME TO accounts;
