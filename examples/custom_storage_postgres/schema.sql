-- PostgreSQL schema for go-prompty template storage
-- Run this to create the required tables:
--   psql your_database < schema.sql

-- Templates table stores all template versions
CREATE TABLE IF NOT EXISTS templates (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    source      TEXT NOT NULL,
    version     INTEGER NOT NULL,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by  VARCHAR(255),
    tenant_id   VARCHAR(255),
    tags        JSONB DEFAULT '[]',

    -- Ensure name+version uniqueness
    CONSTRAINT templates_name_version_unique UNIQUE (name, version)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);
CREATE INDEX IF NOT EXISTS idx_templates_tenant_id ON templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_templates_name_version ON templates(name, version DESC);
CREATE INDEX IF NOT EXISTS idx_templates_tags ON templates USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_templates_metadata ON templates USING GIN(metadata);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at on row changes
DROP TRIGGER IF EXISTS update_templates_updated_at ON templates;
CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Example queries for common operations:

-- Get latest version of a template
-- SELECT * FROM templates WHERE name = 'greeting' ORDER BY version DESC LIMIT 1;

-- Get specific version
-- SELECT * FROM templates WHERE name = 'greeting' AND version = 1;

-- List all templates (latest versions only)
-- SELECT DISTINCT ON (name) * FROM templates ORDER BY name, version DESC;

-- Find templates by tag
-- SELECT * FROM templates WHERE tags @> '["production"]'::jsonb;

-- Find templates by tenant
-- SELECT * FROM templates WHERE tenant_id = 'tenant_abc';

-- Find templates with metadata key
-- SELECT * FROM templates WHERE metadata ? 'author';

-- Find templates where metadata matches
-- SELECT * FROM templates WHERE metadata @> '{"author": "team-a"}'::jsonb;
