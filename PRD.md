
# 🕸️ Web Crawler with Extractor - Product Requirement Document

## 📌 Overview

This system is designed to perform targeted keyword-based crawls on government or organizational websites, extract structured data, and store artifact files (PDF, DOCs, HTML) in Google Cloud Storage. All operations are initiated manually via a Web Dashboard, and each data source has custom extraction logic.

---

## 🎯 Goals

- Support custom, keyword-based search crawling across defined websites.
- Store structured extracted data in PostgreSQL.
- Handle artifact storage (PDFs, HTML, etc.) in Google Cloud Storage.
- Modular and easy to maintain by a small team.
- Scalable, observable, and Docker-deployable in a single VPS.
- Integrate with a Web Dashboard for complete control and transparency.

---

## 👥 Users

- Internal admin team or researchers
- Software engineers maintaining the crawler
- Analysts monitoring data quality and progress

---

## 🏗️ Architecture

```mermaid
graph TD
  A[Web Dashboard] --> B[Go API Service]
  B --> C1[PostgreSQL]
  B --> C2[NATS JetStream]
  
  C2 --> D1[Crawler Worker (Rod/Playwright)]
  D1 --> C1
  D1 --> E1[GCS Storage]
  D1 --> F[Emit extract.run]

  C2 --> D2[Extractor Worker (Goquery/Custom Rules)]
  D2 --> C1
  D2 --> G[Versioning Check]
  D2 --> E1

  B --> H[Logger Service]
  H --> C1

  subgraph Monitoring
    I[Prometheus] --> D1
    I --> D2
    I --> B
    J[Grafana] --> I
  end
```

---

## 🧱 System Components

### 1. **Web Dashboard (Frontend)**
- View data sources, URL frontiers, extraction results
- Trigger crawls by keyword
- Review logs and system metrics
- Tech: Next.js / React

### 2. **Go API Service**
- Manage `data_sources`, `url_frontiers`
- Routes:
  - `POST /crawl/:data_source_id`
  - `GET /logs`
  - `GET /status`
  - `POST /data_sources`
- Publishes `crawl.search` to NATS

### 3. **Crawler Worker**
- Listens to `crawl.search`
- Performs site search using **Rod/Playwright**
- Saves result URLs to `url_frontiers`
- Downloads PDFs, HTML to GCS
- Emits `extract.run` messages
- Respects `config` rules per `data_source`

### 4. **Extractor Worker**
- Listens to `extract.run`
- Extracts data using configured logic (e.g., XPath, CSS selectors)
- Stores result in `extractions`
- If `page_hash` differs, version is saved to `extraction_versions`

### 5. **Logger**
- Logs events with:
  - `event_type`: start, error, complete
  - `url_frontier_id`
  - `data_source_id`
  - Structured metadata
- Stored in `crawler_logs`

### 6. **Monitoring**
- Metrics exposed via Prometheus:
  - `crawl_duration_seconds`
  - `extractions_total`
  - `crawler_errors_total`
- Dashboards in Grafana

---

## 🗃️ Data Storage

### PostgreSQL Tables
- `data_sources`
- `url_frontiers`
- `extractions`
- `extraction_versions`
- `crawler_logs`

### Google Cloud Storage
- Artifact PDFs, HTML pages, DOCs
- Path convention: `gcs://bucket/{data_source_id}/{url_id}/{filename}`

---

## 📬 Messaging: NATS JetStream

### Subjects
- `crawl.search`
- `extract.run`

### Message Example: `crawl.search`
```json
{
  "data_source_id": "source_001",
  "keyword": "climate change report",
  "base_url": "https://gov-agency.org"
}
```

### Message Example: `extract.run`
```json
{
  "url_frontier_id": "url_123456",
  "data_source_id": "source_001"
}
```

---

## 🔁 Workflow

1. Admin triggers a crawl from Dashboard.
2. API publishes `crawl.search` to NATS.
3. Crawler Worker performs site search, downloads files to GCS, populates `url_frontiers`.
4. For each result, `extract.run` is published.
5. Extractor Worker parses content, saves data to `extractions`.
6. If data changes, versioning logic stores in `extraction_versions`.
7. Logs and metrics are continuously updated.

---

## 🔐 Security

- API authenticated via admin credentials or tokens.
- GCS uses service account with limited permissions.
- Sensitive configs (e.g., XPath) stored in `config` JSONB.

---

## 🔁 Retry Logic

- `url_frontiers.attempts` increments on failure
- If `attempts >= 3`, mark as failed, log reason
- Manual retry allowed via Dashboard

---

## ⚙️ Deployment

Docker Stack services:
- `postgres`
- `nats`
- `api` (Go)
- `crawler` (Go with Rod)
- `extractor` (Go)
- `dashboard` (Next.js)
- `prometheus`, `grafana`

All configured via `docker-compose.yml`

---

## 🧪 Observability

Expose metrics:
- HTTP: `/metrics`
- Logs: structured JSON
- Grafana dashboards for:
  - Crawl durations
  - Error rates
  - NATS queue sizes

---

## ✏️ Configuration (`data_sources.config`)

```json
{
  "search_url": "https://gov-agency.org/search?q={{KEYWORD}}",
  "result_selector": ".result-item a",
  "artifact_selector": "a[href$='.pdf']",
  "extraction_rules": {
    "title": "h1",
    "date": "span.date",
    "summary": "div.summary"
  }
}
```

---

## 📅 Milestones

| Phase | Task | Est. Duration |
|-------|------|---------------|
| 1 | Setup DB, schema, GCS | 2 days |
| 2 | Crawler worker + GCS upload | 4 days |
| 3 | Extractor worker + versioning | 3 days |
| 4 | API & NATS integration | 2 days |
| 5 | Dashboard UI + logs | 3 days |
| 6 | Prometheus + Grafana | 1 day |

---

## 📣 Future Enhancements

- Add cron-based re-crawling (`next_crawl_at`)
- Pluggable headless browser support
- Natural language rule builder in UI
- Elasticsearch for full-text search on extracted data

---
