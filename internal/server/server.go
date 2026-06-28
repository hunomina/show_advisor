package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hunomina/show_advisor/internal/embed"
	"github.com/hunomina/show_advisor/internal/store"
)

type Server struct {
	embed      *embed.Client
	store      *store.Qdrant
	collection string
}

func New(e *embed.Client, s *store.Qdrant, collection string) *Server {
	return &Server{
		embed:      e,
		store:      s,
		collection: collection,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /shows", s.handleShows)
	mux.HandleFunc("GET /shows/{id}/similar", s.handleSimilar)

	return logging(mux)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("q")

	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "10"
	}

	castLimit, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Embed the query
	vec, err := s.embed.EmbedQuery(ctx, query)
	if err != nil {
		http.Error(w, "Failed to embed query: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Search in Qdrant
	similar, err := s.store.Search(ctx, s.collection, vec, castLimit)
	if err != nil {
		http.Error(w, "Failed to search: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(similar)
}

func (s *Server) handleShows(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "10"
	}

	castLimit, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	title_hits, err := s.store.LookupByTitle(r.Context(), s.collection, query, castLimit)

	if err != nil {
		http.Error(w, "Failed to search: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(title_hits)
}

func (s *Server) handleSimilar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)

	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "10"
	}

	castLimit, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	similar, err := s.store.RecommendSimilar(r.Context(), s.collection, id, castLimit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(similar)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK} // default 200
		next.ServeHTTP(rec, r)
		fmt.Printf("%s %s %d %s\n", r.Method, r.URL.RequestURI(), rec.status, time.Since(start))
	})
}
