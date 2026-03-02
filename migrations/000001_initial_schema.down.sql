-- Drop in reverse dependency order

DROP TABLE IF EXISTS audit_logs_2027;
DROP TABLE IF EXISTS audit_logs_2026;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS transform_cache;
DROP TABLE IF EXISTS image_styles;
DROP TABLE IF EXISTS collection_assets;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS asset_tags;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS domains;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

DROP FUNCTION IF EXISTS assets_search_update();

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "uuid-ossp";
