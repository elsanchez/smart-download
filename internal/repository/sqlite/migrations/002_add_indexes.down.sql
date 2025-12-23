-- Rollback de índices
DROP INDEX IF EXISTS idx_downloads_status;
DROP INDEX IF EXISTS idx_downloads_created_at;
DROP INDEX IF EXISTS idx_downloads_platform;
DROP INDEX IF EXISTS idx_downloads_active;
DROP INDEX IF EXISTS idx_accounts_platform;
DROP INDEX IF EXISTS idx_accounts_active;
