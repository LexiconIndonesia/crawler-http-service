ALTER TABLE "data_sources" ADD COLUMN "deleted_at" timestamptz NULL;
CREATE INDEX idx_data_sources_deleted_at ON data_sources (deleted_at);
