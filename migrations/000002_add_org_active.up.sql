-- Add active column to organizations table
ALTER TABLE organizations
ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;

-- Create an index on active for queries that filter by status
CREATE INDEX idx_organizations_active ON organizations(active);

