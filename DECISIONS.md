# Decision Log

Every significant decision and the reasoning behind it, ordered by decision
number. This is a learning project (TV show advisor over IMDb data using a vector
database), so the *why* matters as much as the *what*.

---

### D1 — Project: a TV show advisor using natural-language semantic search

**Decision:** Build a service where a user types a natural-language query
(e.g. "dark sci-fi with time travel") and gets back the most semantically
similar TV shows, ranked.
**Why:** Best balance for *learning vector databases* — exercises the full
text → embedding → nearest-neighbor-search loop without the extra complexity of
an LLM chatbot (RAG). Pure vector search makes the core concept visible.

### D2 — Language: Go

**Decision:** Implement in Go, not Python.
**Why:** User preference / the language they want to practice. Note this rules
out the usual Python embedding stack (sentence-transformers), which pushed us
toward Ollama for embeddings (see D5).

### D3 — Vector database: Qdrant

**Decision:** Use Qdrant as the vector store.
**Why:** It was the running example in the originating ChatGPT thread; mature,
simple REST API, easy to run in Docker, good fit for learning the
collection/point/payload model.

### D4 — Qdrant access from Go: REST via `net/http` (not the gRPC SDK)

**Decision:** Call Qdrant's REST API directly with the standard library, instead
of the official `qdrant/go-client` (gRPC).
**Why:** Keeps dependencies minimal and — more importantly for learning — keeps
the actual vector-DB requests (create collection, upsert, search) *visible* in
our own code rather than hidden behind an SDK. The official client remains a
later optional upgrade.

### D5 — Embeddings: local model served by Ollama

**Decision:** Generate embeddings with a local model run by Ollama, exposed over
its HTTP API.
**Why:** User wanted embeddings to run **locally** (free, no API key, no rate
limits). Since sentence-transformers is Python-only and we're in Go, Ollama is
the cleanest local option — Go just makes HTTP calls to it.

### D6 — Run Ollama in Docker (not a direct install)

**Decision:** Run Ollama as a Docker container alongside Qdrant, via
`docker compose`.
**Why:** User prefers to containerize "exotic" services and isn't familiar with
Ollama — a container means zero local install and an easy teardown. Both backing
services then come up with a single `docker compose up`.

### D7 — Embedding model: `nomic-embed-text` (768-dim, Cosine distance)

**Decision:** Use `nomic-embed-text` as the embedding model; create the Qdrant
collection with vector size **768** and **Cosine** distance.
**Why:** Best *default for our constraints* (local, free, HTTP-from-Go, semantic
search over short texts) — not necessarily the single best model overall:

- Purpose-built for **retrieval** (trained on query↔document pairs) — matches our
  task exactly.
- One-command local install via Ollama; runs fine CPU-only in Docker (D6).
- Small/fast (137M params), 768-dim sweet spot, 2048-token context (ample for
  title+genres+plot), strong MTEB scores for its size.
- **Swappable**: model is behind an HTTP call + config, so changing to
  `all-MiniLM` (384, faster) or `mxbai-embed-large` (1024, higher quality) is just
  a model name + collection-size change and a re-ingest.

**Vector size 768** must match the model's output exactly — it's the Qdrant
collection contract. **Cosine** distance because embeddings encode meaning in
*direction* not magnitude, and the model is trained with a cosine objective;
matching the metric to the model is the point. (For normalized vectors Cosine/Dot/
Euclid rank similarly; Cosine normalizes internally, so it's the safe default.)

**Known refinement (TODO):** nomic recommends **task prefixes** —
`search_document:` for stored items, `search_query:` for the user query — which
measurably improves retrieval and fits our query/document asymmetry. To be added
in the `embed` layer when wiring ingest/search.

### D8 — Data: a Kaggle IMDb dataset that includes plot descriptions

**Decision:** Source data from a Kaggle "IMDb" dataset that bundles plot
summaries, downloaded manually into `data/`.
**Why:** The *official* IMDb non-commercial datasets (`title.basics.tsv`, etc.)
have **no plot text** — and the plot is the most valuable thing to embed for
semantic search. A Kaggle dataset with descriptions avoids needing a separate
enrichment step (e.g. TMDB API). Manual download because Kaggle requires auth.

### D9 — Interface: HTTP API (ingestion as a CLI subcommand)

**Decision:** Expose search via an HTTP API in Go; data ingestion is a CLI
subcommand of the same binary.
**Why:** User chose HTTP API. Leaves the door open for a web UI later. Ingestion
is a one-time batch job, so a CLI command fits it better than an endpoint.

### D10 — Upgrade Go to native arm64 1.26.x

**Decision:** Upgrade Go from 1.19.5 to the latest (1.26.4) via Homebrew.
**Why:** The installed Go was an **amd64 build running under Rosetta** on an
arm64 Mac (slower, non-native). Upgrading also gets the native arm64 build and a
current toolchain for the `go` directive in `go.mod`.

### D11 — Description handling: embed the full text, store only a short snippet

**Decision:** Feed the **full** description into the embedding input
(`title + genres + full description`), but store only a **~200-char snippet** in
the Qdrant payload. (Payload field set later refined by D16.)
**Why:** The description's *meaning* is captured in the vector, so storing the
full raw text in the payload would just be bloat (the "don't dump large raw text
you already embedded" anti-pattern). But a short snippet is genuinely useful for
*displaying* results so the user sees why a show matched. Snippet truncation
happens in `dataset.go` at parse time.

### D12 — Working mode: user codes, assistant supervises (goal + hints)

**Decision:** The user writes all application code. The assistant explains
concepts, hands over one task at a time with the goal + key APIs/gotchas, and
reviews the result. Build order: Go skeleton first. The assistant does NOT write
application code (docs like this file are fine).
**Why:** This is a learning exercise — the user wants to understand Ollama and
Qdrant by building, not receive a finished implementation.

### D13 — CLI: use the `spf13/cobra` library (not hand-rolled arg parsing)

**Decision:** Build the CLI (`ingest`/`serve` subcommands) with `spf13/cobra`.
**Why:** Implementing argument parsing is not the point of this project, and
Cobra is the de-facto Go CLI standard (kubectl, Hugo, gh, Docker) — transferable
knowledge, free subcommands/flags/help. No code-generator needed; commands are
written by hand. (`urfave/cli/v2` considered as a lighter alternative.)

### D13b — Store package: `package store` with a `NewQdrant()` constructor

**Decision:** Use `package store` (matching the `internal/store` directory) and
name the constructor `NewQdrant()`, called as `store.NewQdrant(...)`.
**Why:** Keeps the Go convention (package name = dir name) while still surfacing
the implementation at the call site via the constructor name. Scales cleanly: a
second backend would just be another `store.NewX()` in the same package, no
package-name juggling. (Superseded an earlier idea of `package qdrant` in a
`store` dir.)

### D14 — Two user features: semantic search + "pick a show → similar"

**Decision:** Ship two distinct features over the same Qdrant collection:

1. **Semantic search** — free-text query → embed → vector `Search`
   (`GET /search?q=...`).
2. **Pick-a-show → more like this** — `LookupByTitle` (full-text on `title`,
   returns `[]{id, title}`) lets the user pick a specific show, then
   `RecommendSimilar` returns neighbours (`GET /shows?q=...` then
   `GET /shows/{id}/similar`).
**Why:** A text query like "Marvel" matches many shows; the user wants to choose
a *specific* one and get things like it. That's a different operation from
semantic search — it starts from a known item, not a description.
**Key choices:**

- **Title lookup = full-text** payload index on `title` (tokenizer **`prefix`**,
  lowercase), created in `EnsureCollection`; returns **all** matching shows
  (e.g. "Marvel" → "Marvel Agents of Shield" + "Marvel Inhumans" + "Marvels").
  Prefix chosen over `word` (misses plural/variant forms) and over app-side
  `strings.Contains` (true "contains" but scans every title per request — doesn't
  scale). Prefix is index-backed and scalable; trade-off is no mid-word substring.
- Lookup returns **id + title only** (id is needed to drive the recommend call,
  even though only the title is displayed).
- "Similar" uses Qdrant's **`/points/recommend`** with the chosen id — it reuses
  the **stored** vector (no re-embedding) and excludes the source show
  automatically. Cheap given we already store the vectors.

### D15 — Dataset schema mapping (Kaggle IMDb TV CSV)

**Decision:** Map the CSV
(`Title, About, EpisodeDuration(in Minutes), Genres, Actors, Rating, Votes, Years`)
to `Show{ Title, About string; Genres, Actors []string; Rating float64; StartYear, EndYear int }`:

- `Title`→title, `About`→description (embed input + ~200-char snippet),
  `Genres`/`Actors`→split on comma → `[]string`, `Rating`→float.
- **`Years` is a range** like `"2019–"`, `"2016–2018"`, `"2011–2019"` using an
  **en-dash `–` (U+2013, not hyphen)**, sometimes with a trailing space. Parse:
  `TrimSpace`, split on `–` → `StartYear` (before) and `EndYear` (after);
  **empty end ⇒ `EndYear = 0` (ongoing)**.
- `EpisodeDuration`/`Votes` ignored for now.
**Why:** These are the columns the dataset provides; the range gives us both
"started after X" and "still running" later.
**Consequences:**
- **No id column** → Qdrant point id = **hash of `Title` + `StartYear`**
  (FNV→uint64, null-separated). Title-only collided ~44 same-named shows
  (reboots/remakes) into one point; adding the year disambiguates distinct shows
  while still collapsing true duplicate rows. Keeps re-ingest idempotent.
- **No type column** → dataset is TV-only, no movie filtering needed.

### D16 — What goes in the vector vs. the payload

**Decision:** Embed only meaningful **text** fields; keep numeric/structured
fields in the payload.

- **Embedded (`EmbedText`)**: `Title`, `Genres`, `Actors`, full `About`.
- **Payload**: `Title`, `Genres`, `Actors`, `Rating`, `StartYear`, `EndYear`
  (+ Duration/Votes if ever added) for filtering/sorting/display; About kept as a
  ~200-char `Snippet`.
- `Genres`/`Actors` are in **both** — embedded for semantic similarity, *and* in
  payload as `[]string` so they can be filtered (`match`) and displayed.
**Why:** Embedding numbers like a year adds noise, not meaning — two shows aren't
"similar" because both say 2011. Years/ratings are *filter* fields (e.g. "after
2015", "still running"), which is a payload job. Genres and especially `Actors`
are real semantic signal (overlapping cast/genre → genuinely similar shows), so
they belong in the vector.

### D17 — Pin a recent Qdrant image for the built-in dashboard

**Decision:** Bump the Qdrant image from `v1.2.0` to a pinned recent version
(e.g. `qdrant/qdrant:v1.12.4`) to get the **built-in web dashboard** at
`/dashboard`. No separate UI container — the dashboard ships inside Qdrant.
**Why:** v1.2.0 predates the bundled dashboard (hence the 404). A pinned recent
version gives the UI and reproducible builds (chose pinning over `:latest`).
**Consequence:** the 1.2→1.x storage format jump isn't guaranteed compatible, so
the volume is wiped and data re-ingested (only synthetic test data existed).

### D18 — Web UI: Vue 3 + Vite + Tailwind v4, served by nginx with API proxy

**Decision:** Add a `frontend/` SPA (Vue 3 `<script setup>`, Vite, Tailwind CSS
v4) covering all three features (semantic search, find-by-title, "more like
this"). In Docker it's served by **nginx**, which also **reverse-proxies
`/api/*` → `http://backend:8080`**. New compose service `frontend` on `:3000`.
**Why (and why the backend needs NO changes):**
- **Reverse proxy instead of CORS** — the browser only ever calls the frontend
  origin (`/api/...`); nginx forwards to the backend. No cross-origin request, so
  no CORS headers needed on the Go API. (Vite dev server mirrors this with a
  `/api` proxy.)
- **uint64 IDs vs JS numbers** — point IDs (FNV hash) exceed
  `Number.MAX_SAFE_INTEGER`; parsing as JS numbers corrupts them and breaks the
  `/shows/{id}/similar` round-trip. Handled **client-side**: `api.js` quotes
  `"id":<digits>` → `"id":"<digits>"` before `JSON.parse`, keeping IDs as exact
  strings. No backend change.
- Tailwind v4 chosen for a recent, clean styling setup (single `@import
  "tailwindcss"` + `@tailwindcss/vite` plugin, no separate config file).
**Result:** backend untouched; `frontend` is a separate multi-stage image
(node build → nginx). Verified `npm run build` succeeds.
