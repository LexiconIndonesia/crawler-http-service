-- POSTGRESQL

-- Create data_sources table to categorize different crawlers
CREATE TABLE "data_sources" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "country" varchar(100) NOT NULL,
  "source_type" varchar(100) NOT NULL,
  "base_url" varchar(255) NULL,
  "description" text NULL,
  "config" jsonb NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_data_sources_country ON data_sources (country);
CREATE INDEX idx_data_sources_source_type ON data_sources (source_type);

-- Create UrlFrontier Table with improved structure
CREATE TABLE "url_frontiers" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "data_source_id" varchar(64) NOT NULL,
  "domain" varchar(255) NOT NULL,
  "url" varchar(2048) NOT NULL, -- Increased length for longer URLs
  "keyword" varchar(255) NULL,
  "priority" smallint NOT NULL DEFAULT 0, -- For prioritizing certain crawls
  "status" smallint NOT NULL DEFAULT 0,
  "attempts" smallint NOT NULL DEFAULT 0, -- Track retry attempts
  "last_crawled_at" timestamptz NULL,
  "next_crawl_at" timestamptz NULL, -- For scheduling re-crawls
  "error_message" text NULL, -- To store error messages
  "metadata" jsonb NULL, -- For storing search parameters and other context
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_url_frontiers_data_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);
ALTER TABLE url_frontiers ADD CONSTRAINT url_frontiers_unique UNIQUE (url, data_source_id);

-- Set Comment for Status Column with more detailed statuses
COMMENT ON COLUMN "url_frontiers"."status" IS '0: Pending, 1: Crawled, 2: Changed, 3: Failed, 4: Scheduled, 5: Processing';
COMMENT ON COLUMN "url_frontiers"."metadata" IS 'Stores search parameters and context information';

-- Create index for domain
CREATE INDEX idx_domain ON url_frontiers (domain);

-- Create index for url
CREATE INDEX idx_url ON url_frontiers (url);

-- Create index for data_source_id
CREATE INDEX idx_data_source ON url_frontiers (data_source_id);

-- Create index for status
CREATE INDEX idx_status ON url_frontiers (status);

-- Create index for next_crawl_at to efficiently find URLs due for crawling
CREATE INDEX idx_next_crawl ON url_frontiers (next_crawl_at) WHERE next_crawl_at IS NOT NULL;

-- Create extractions table with improved structure
CREATE TABLE "extractions" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "url_frontier_id" varchar(64) NOT NULL,
  "site_content" text NULL,
  "artifact_link" varchar(2048) NULL, -- Increased length
  "raw_page_link" varchar(2048) NULL, -- Increased length
  "extraction_date" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "content_type" varchar(100) NULL, -- HTML, PDF, JSON, etc.
  "metadata" jsonb NULL, -- For storing structured extracted data
  "language" varchar(10) NOT NULL DEFAULT 'en',
  "page_hash" varchar(255) NULL,
  "version" integer NOT NULL DEFAULT 1, -- For tracking changes in data
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_extractions_url_frontiers_id FOREIGN KEY (url_frontier_id) REFERENCES url_frontiers(id)
);
COMMENT ON COLUMN "extractions"."metadata" IS 'Stores structured data extracted from the source';

-- Create index for url_frontier_id
CREATE INDEX idx_extraction_url_frontier ON extractions (url_frontier_id);

-- Create index for extraction_date
CREATE INDEX idx_extraction_date ON extractions (extraction_date);

-- Create index for content_type
CREATE INDEX idx_content_type ON extractions (content_type);

-- Create table for historical versions of extractions
CREATE TABLE "extraction_versions" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "extraction_id" varchar(64) NOT NULL,
  "site_content" text NULL,
  "artifact_link" varchar(2048) NULL,
  "raw_page_link" varchar(2048) NULL,
  "metadata" jsonb NULL, -- For storing structured extracted data
  "page_hash" varchar(255) NULL,
  "version" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_extraction_versions_extraction FOREIGN KEY (extraction_id) REFERENCES extractions(id)
);
COMMENT ON COLUMN "extraction_versions"."metadata" IS 'Stores structured data extracted from the source at this version';

-- Create index for extraction_id
CREATE INDEX idx_extraction_version_extraction ON extraction_versions (extraction_id);

-- Create table for crawler logs
CREATE TABLE "crawler_logs" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "data_source_id" varchar(64) NOT NULL,
  "url_frontier_id" varchar(64) NULL,
  "jobs_id" varchar(64) NULL,
  "event_type" varchar(100) NOT NULL, -- start, end, error, warning, etc.
  "message" text NULL,
  "details" jsonb NULL, -- For storing structured log data and context
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_crawler_logs_data_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id),
  CONSTRAINT fk_crawler_logs_url_frontier FOREIGN KEY (url_frontier_id) REFERENCES url_frontiers(id)
);
COMMENT ON COLUMN "crawler_logs"."details" IS 'Stores structured log data including context, parameters, and results';

-- Create index for data_source_id
CREATE INDEX idx_logs_data_source ON crawler_logs (data_source_id);

-- Create index for url_frontier_id
CREATE INDEX idx_logs_url_frontier ON crawler_logs (url_frontier_id);

-- Create index for event_type
CREATE INDEX idx_logs_event_type ON crawler_logs (event_type);

-- Create index for jobs_id
CREATE INDEX idx_logs_jobs_id ON crawler_logs (jobs_id);


