-- Remove active column from organizations table
DROP INDEX IF EXISTS idx_organizations_active;
ALTER TABLE organizations
DROP COLUMN active;

