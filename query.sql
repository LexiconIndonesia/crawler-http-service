-- =============================================
-- URL Frontiers Table Operations
-- =============================================

-- Upsert a single URL frontier record
-- name: UpsertUrlFrontier :exec
INSERT INTO url_frontiers (id, data_source_id, domain, url, keyword, priority, status, attempts, last_crawled_at, next_crawl_at, error_message, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (url, data_source_id) DO UPDATE
SET
  domain = $3,
  keyword = $5,
  priority = $6,
  status = $7,
  attempts = $8,
  last_crawled_at = $9,
  next_crawl_at = $10,
  error_message = $11,
  metadata = $12,
  updated_at = $14;

-- Upsert multiple URL frontier records in batch
-- name: UpsertUrlFrontiers :batchexec
INSERT INTO url_frontiers (id, data_source_id, domain, url, keyword, priority, status, attempts, last_crawled_at, next_crawl_at, error_message, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (url, data_source_id) DO UPDATE
SET
  domain = $3,
  keyword = $5,
  priority = $6,
  status = $7,
  attempts = $8,
  last_crawled_at = $9,
  next_crawl_at = $10,
  error_message = $11,
  metadata = $12,
  updated_at = $14;

-- Update status and increment attempts for URL frontiers
-- name: UpdateUrlFrontierStatus :exec
UPDATE url_frontiers
SET
  status = $2,
  attempts = COALESCE(attempts, 0) + 1,
  last_crawled_at = CURRENT_TIMESTAMP,
  error_message = $3,
  updated_at = $4
WHERE id = $1;

-- Update Status and increment by batch ids
-- name: UpdateUrlFrontierStatusBatch :exec
UPDATE url_frontiers
SET
  status = $2,
  attempts = COALESCE(attempts, 0) + 1,
  last_crawled_at = CURRENT_TIMESTAMP,
  error_message = $3,
  updated_at = $4
WHERE id = ANY($1);


-- Get unscrapped URL frontiers for a data source
-- name: GetUnscrappedUrlFrontiers :many
SELECT id, data_source_id, domain, url, keyword, priority, status, attempts, last_crawled_at, next_crawl_at, error_message, metadata, created_at, updated_at
FROM url_frontiers
WHERE
  data_source_id = $1
  AND status = $2
ORDER BY priority DESC, created_at ASC LIMIT $3;

-- Get URL frontier by URL
-- name: GetUrlFrontierByUrl :one
SELECT id, data_source_id, domain, url, keyword, priority, status, attempts, last_crawled_at, next_crawl_at, error_message, metadata, created_at, updated_at
FROM url_frontiers
WHERE url = $1
LIMIT 1;

-- Get URL frontier by ID
-- name: GetUrlFrontierById :one
SELECT id, data_source_id, domain, url, keyword, priority, status, attempts, last_crawled_at, next_crawl_at, error_message, metadata, created_at, updated_at
FROM url_frontiers
WHERE id = $1
LIMIT 1;

-- =============================================
-- Data Sources Table Operations
-- =============================================

-- Get data source by ID
-- name: GetDataSourceById :one
SELECT id, name, country, source_type, base_url, description, config, is_active, created_at, updated_at
FROM data_sources
WHERE id = $1
LIMIT 1;

-- Get all active data sources
-- name: GetActiveDataSources :many
SELECT id, name, country, source_type, base_url, description, config, is_active, created_at, updated_at
FROM data_sources
WHERE is_active = true;

-- Upsert data source
-- name: UpsertDataSource :exec
INSERT INTO data_sources (id, name, country, source_type, base_url, description, config, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE
SET
  name = $2,
  country = $3,
  source_type = $4,
  base_url = $5,
  description = $6,
  config = $7,
  is_active = $8,
  updated_at = $10;

-- Get data source by name
-- name: GetDataSourceByName :one
SELECT id, name, country, source_type, base_url, description, config, is_active, created_at, updated_at
FROM data_sources
WHERE name = $1
LIMIT 1;

-- =============================================
-- Extractions Table Operations
-- =============================================

-- Upsert extraction records in batch
-- name: UpsertExtraction :batchexec
INSERT INTO extractions (id, url_frontier_id, site_content, artifact_link, raw_page_link, extraction_date, content_type, metadata, language, page_hash, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE
SET
  url_frontier_id = $2,
  site_content = $3,
  artifact_link = $4,
  raw_page_link = $5,
  extraction_date = $6,
  content_type = $7,
  metadata = $8,
  language = $9,
  page_hash = $10,
  version = COALESCE(extractions.version, 0) + 1,
  updated_at = $13;

-- Get extraction by ID
-- name: GetExtractionById :one
SELECT id, url_frontier_id, site_content, artifact_link, raw_page_link,
       extraction_date, content_type, metadata, language, page_hash,
       version, created_at, updated_at
FROM extractions
WHERE id = $1
LIMIT 1;

-- Get extractions by URL frontier ID
-- name: GetExtractionsByUrlFrontierID :many
SELECT id, url_frontier_id, site_content, artifact_link, raw_page_link,
       extraction_date, content_type, metadata, language, page_hash,
       version, created_at, updated_at
FROM extractions
WHERE url_frontier_id = $1
ORDER BY version DESC, created_at DESC;

-- =============================================
-- Extraction Versions Table Operations
-- =============================================

-- Create a new extraction version
-- name: CreateExtractionVersion :exec
INSERT INTO extraction_versions (
    id, extraction_id, site_content, artifact_link, raw_page_link,
    metadata, page_hash, version, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- =============================================
-- Crawler Logs Table Operations
-- =============================================

-- Create a new crawler log entry
-- name: CreateCrawlerLog :exec
INSERT INTO crawler_logs (id, data_source_id, url_frontier_id, event_type, message, details, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);
