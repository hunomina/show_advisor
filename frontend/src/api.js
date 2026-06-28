// API client for the ShowAdvisor backend.
//
// Requests go to /api/* which nginx (prod) or the Vite dev proxy forwards to the
// Go backend — so the browser stays same-origin and no CORS is needed.
//
// IMPORTANT: point IDs are uint64 (e.g. 723797795888263189), larger than
// Number.MAX_SAFE_INTEGER. Parsing them as JS numbers loses precision and breaks
// the /shows/{id}/similar round-trip. We quote them into strings *before*
// JSON.parse so they survive intact (and we only ever pass them back verbatim).
const BASE = '/api'

async function getJSON(path) {
  const res = await fetch(BASE + path)
  const text = await res.text()
  if (!res.ok) {
    throw new Error(text || res.statusText)
  }
  const safe = text.replace(/"id":\s*(\d+)/g, '"id":"$1"')
  return JSON.parse(safe)
}

export const api = {
  // Semantic search: free-text query -> ranked shows (full payload + score).
  search: (q, limit = 12) =>
    getJSON(`/search?q=${encodeURIComponent(q)}&limit=${limit}`),

  // Title lookup: prefix match on title -> [{ id, title }].
  shows: (q) => getJSON(`/shows?q=${encodeURIComponent(q)}`),

  // "More like this": neighbours of a chosen show id (full payload + score).
  similar: (id, limit = 12) =>
    getJSON(`/shows/${id}/similar?limit=${limit}`),

  // Liveness check.
  health: async () => {
    try {
      const res = await fetch(`${BASE}/healthz`)
      return res.ok
    } catch {
      return false
    }
  },
}
