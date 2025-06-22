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
SELECT *
FROM data_sources
WHERE id = $1
LIMIT 1;

-- Get all active data sources
-- name: GetActiveDataSources :many
SELECT *
FROM data_sources
WHERE is_active = true AND deleted_at IS NULL;

-- Create a new data source
-- name: CreateDataSource :one
INSERT INTO data_sources (id, name, country, source_type, base_url, description, config, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- Update an existing data source
-- name: UpdateDataSource :one
UPDATE data_sources
SET
  name = $2,
  country = $3,
  source_type = $4,
  base_url = $5,
  description = $6,
  config = $7,
  is_active = $8,
  updated_at = $9
WHERE id = $1
RETURNING *;

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
SELECT *
FROM data_sources
WHERE name = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListDataSources :many
SELECT *
FROM data_sources
WHERE
    (sqlc.narg('data_source')::TEXT IS NULL OR name = sqlc.narg('data_source'))
AND
    (sqlc.narg('search')::TEXT IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
AND deleted_at IS NULL
ORDER BY
    name ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountDataSources :one
SELECT count(*)
FROM data_sources
WHERE
    (sqlc.narg('data_source')::TEXT IS NULL OR name = sqlc.narg('data_source'))
AND
    (sqlc.narg('search')::TEXT IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
AND deleted_at IS NULL;

-- Soft delete a data source by ID
-- name: DeleteDataSource :exec
UPDATE data_sources
SET deleted_at = NOW()
WHERE id = $1;

-- Get all data sources, including deleted ones
-- name: GetAllDataSources :many
SELECT * FROM data_sources;

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
-- name: CreateCrawlerLog :one
INSERT INTO crawler_logs (id, data_source_id, job_id, event_type, message, details, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- Get crawler logs by job ID
-- name: GetCrawlerLogsByJobId :many
SELECT * FROM crawler_logs WHERE job_id = $1 ORDER BY created_at DESC;

-- =============================================
-- Jobs Table Operations
-- =============================================

-- Create a new job (queued)
-- name: CreateJob :one
INSERT INTO jobs (
    id,
    status
) VALUES (
    $1, $2
) RETURNING id, status, created_at, updated_at, started_at, finished_at;

-- Update job status and updated_at timestamp
-- name: UpdateJobStatus :one
UPDATE jobs
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, status, created_at, updated_at, started_at, finished_at;

-- Get job by ID
-- name: GetJobByID :one
SELECT id, status, created_at, updated_at, started_at, finished_at
FROM jobs
WHERE id = $1
LIMIT 1;

-- List jobs with pagination and filtering
-- name: ListJobs :many
SELECT id, status, created_at, updated_at, started_at, finished_at
FROM jobs
WHERE
    (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
AND
    (sqlc.narg('search')::TEXT IS NULL OR id ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- Count jobs with filtering
-- name: CountJobs :one
SELECT count(*)
FROM jobs
WHERE
    (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
AND
    (sqlc.narg('search')::TEXT IS NULL OR id ILIKE '%' || sqlc.narg('search') || '%');
