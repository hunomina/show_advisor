// Package turns text into vector embeddings by calling a local Ollama
// server's /api/embed endpoint.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

func New(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Returns one vector per input string, in the same order. Passing many
// inputs in a single call batches them into one HTTP request.
func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	body, err := json.Marshal(embedRequest{Model: c.model, Input: inputs})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed: status %d: %s", resp.StatusCode, b)
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	// Ollama should return exactly one vector per input; guard against a
	// surprise so callers can rely on index alignment.
	if len(out.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("ollama embed: got %d vectors for %d inputs", len(out.Embeddings), len(inputs))
	}

	return out.Embeddings, nil
}

// nomic-embed-text is an instructed model: it was trained on (query, document)
// pairs where each side carried a task prefix, so the same prefixes must be
// replayed at inference. Documents (stored items) use "search_document: ",
// queries use "search_query: " — this asymmetry is what aligns a short query
// with its longer matching document in vector space.

const (
	docPrefix   = "search_document: "
	queryPrefix = "search_query: "
)

func (c *Client) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	prefixed := make([]string, len(texts))
	for i, t := range texts {
		prefixed[i] = docPrefix + t
	}
	return c.Embed(ctx, prefixed)
}

func (c *Client) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{queryPrefix + text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}
