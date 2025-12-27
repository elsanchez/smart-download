-- Add cookie validation fields to accounts table

ALTER TABLE accounts ADD COLUMN validated_at INTEGER;
ALTER TABLE accounts ADD COLUMN validation_status TEXT DEFAULT 'unknown';
ALTER TABLE accounts ADD COLUMN validation_error TEXT;

-- Create index for faster validation status queries
CREATE INDEX idx_accounts_validation_status ON accounts(validation_status);
