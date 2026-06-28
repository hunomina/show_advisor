# TODO — TV Show Advisor

Build plan as a checklist. Kept current as we go. See `DECISIONS.md` for the
*why* behind each choice.

Legend: `[ ]` todo · `[~]` in progress · `[x]` done

## Setup
- [x] **Step 0** — Upgrade Go to native arm64 1.26.x (`brew install go`), verify
  `go version` → `darwin/arm64` (done: go1.26.4 darwin/arm64)
- [x] **Task 1** — Go project skeleton: `go mod init`, CLI via **`spf13/cobra`**
  — `root` + `ingest`/`serve` subcommands compile & dispatch (`--help` works)

## Infrastructure
- [x] `docker-compose.yml` — Qdrant (`:6333` REST, `:6334` gRPC) + Ollama
  (`:11434`), with volumes for persistence
- [x] Bump Qdrant image v1.2.0 → pinned recent for built-in dashboard at
  `/dashboard` (working); wiped volume + re-ingest
- [x] Pull embedding model: `nomic-embed-text` into Ollama (768-dim, confirmed)
- [x] Smoke-test both services by hand with `curl` (collections, /api/tags,
  /api/embed all OK)

## Go application
- [x] `internal/config` — env config (QdrantURL, OllamaURL, HttpApiPort, Model,
  Collection) with host defaults; `ingest`/`serve` now config-driven
- [x] `internal/embed` — Ollama client: `Embed(ctx, []string) → [][]float32` via
  `POST /api/embed` (verified: 1 input → 1×768 vector)
  - [x] nomic task prefixes via `EmbedDocuments`/`EmbedQuery`
    (`search_document:` / `search_query:`); re-ingested
- [x] `internal/store` — Qdrant REST client (`package store`, `NewQdrant()`)
  - [x] `EnsureCollection` — idempotent GET→404→PUT; verified size:768/Cosine
  - [x] `Upsert` — write points (id, vector, payload); verified points_count:1
  - [x] `Search` — query vector → ranked hits; verified 0.7759 on test point
  - [x] `EnsureCollection`: full-text `title` index (prefix, lowercase); verified
    in payload_schema over 2985 points
  - [x] `LookupByTitle` — scroll + prefix filter on `title` → `[]TitleHit`;
    verified ("marvel" → Marvelous titles)
  - [x] `RecommendSimilar` — `/points/recommend` by id (stored vector, excl. self);
    verified (Mrs. Maisel → comedies about women/family)
- [x] `internal/dataset` — CSV loader → `[]Show`; parses all 2986
  - [x] `Show{Title, About; Genres, Actors []string; Rating float64; StartYear, EndYear int}`
  - [x] parse Genres+Actors (split `,`, trim), Years range (en-dash `–`, empty end ⇒ 0)
  - [x] `EmbedText()` (title+genres+actors+about) and `Snippet()` (~200 runes)
  - [x] skip rows missing Title/About (or unparseable Rating); consistent counters
- [x] `ingest` command — CSV → batch-embed → upsert (id = `HashID(Title+Year)`);
  2985 points (1 true dup collapsed). Actors added to payload.
- [x] `internal/server` + `serve` command:
  - [x] `GET /healthz`
  - [x] `GET /search?q=...` — semantic search; verified real results over HTTP
  - [x] `GET /shows?q=...` — title lookup via `LookupByTitle` → `[{id, title}]`
  - [x] `GET /shows/{id}/similar` — `RecommendSimilar` by picked id; verified

## Containerize the Go app (run without local Go)
- [x] `internal/config` env vars (prerequisite for service-name URLs in Docker)
- [x] `Dockerfile` — multi-stage (`golang:1.26` → distroless static),
  `CGO_ENABLED=0`, `ENTRYPOINT ["/showadvisor"]`, `CMD ["serve"]`
- [x] `.dockerignore` — exclude `data/`, `.git`, `*.md` from build context
- [x] compose **`backend`** service — `build: .`, `8080:8080`, env service-name
  URLs, `./data:/data:ro`, `depends_on: qdrant, ollama`
- [x] `ingest --csv` flag added (default `data/imdb_tvshows.csv`); verified via
  `docker compose run --rm backend ingest --csv /data/imdb_tvshows.csv`
- [x] `Makefile` — convenience targets (docker: up/down/clean/logs/pull-model/
  ingest/dashboard; dev: dev-serve/dev-ingest/fmt/tidy)
- [x] `healthcheck` subcommand + compose healthcheck on `backend` (binary self-
  pings `/healthz` — distroless-friendly, no curl/shell needed)

## Frontend (Vue web UI)
- [x] `frontend/` — Vue 3 + Vite + Tailwind v4 SPA; `npm run build` verified
- [x] `api.js` — calls `/api/*`; quotes uint64 ids to strings (JS precision)
- [x] features: semantic search, find-by-title, "more like this" (`ShowCard`,
  mode toggle, similar section, health indicator)
- [x] nginx serves SPA + reverse-proxies `/api/*` → backend (no CORS, no backend
  change); Vite dev proxy mirrors it
- [x] `frontend/Dockerfile` (node build → nginx) + compose `frontend` service `:3000`
- [x] gitignore `frontend/node_modules`, `frontend/dist`
