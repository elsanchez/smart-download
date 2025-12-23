-- Tabla de descargas
CREATE TABLE downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    platform TEXT,
    username TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    output_path TEXT,
    options TEXT NOT NULL DEFAULT '{}',
    account_id INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    completed_at INTEGER,
    error_message TEXT,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
);

-- Tabla de cuentas de plataforma
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform TEXT NOT NULL,
    name TEXT NOT NULL,
    cookie_path TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 0,
    last_used INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(platform, name)
);

-- Tabla de configuración
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- Versión de schema
INSERT INTO config (key, value) VALUES ('schema_version', '001');
