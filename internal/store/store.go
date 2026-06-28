package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Qdrant struct {
	baseURL string
	http    *http.Client
}

func NewQdrant(baseURL string) *Qdrant {
	return &Qdrant{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

type Collection struct {
	Name     string
	Size     int
	Distance string
	Indexes  []FieldIndex
}

type FieldIndex struct {
	Name      string
	Tokenizer string
}

func (q *Qdrant) EnsureCollection(ctx context.Context, collection Collection) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q.baseURL+"/collections/"+collection.Name, nil)
	if err != nil {
		return fmt.Errorf("new get qdrant request: %w", err)
	}

	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("call get qdrant collections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		err = q.createCollection(ctx, collection)
		if err != nil {
			return fmt.Errorf("create collection => %w", err)
		}
	} else if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("get qdrant collections: status %d: %s", resp.StatusCode, b)
	}

	for _, index := range collection.Indexes {
		if err := q.EnsureCollectionLowercaseTextIndex(ctx, collection.Name, index); err != nil {
			return fmt.Errorf("ensure collection index => %w", err)
		}
	}

	return nil
}

type vectorParams struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
}

type createCollectionRequest struct {
	Vectors vectorParams `json:"vectors"`
}

func (q *Qdrant) createCollection(ctx context.Context, collection Collection) error {
	body, err := json.Marshal(createCollectionRequest{
		Vectors: vectorParams{Size: collection.Size, Distance: collection.Distance},
	})

	if err != nil {
		return fmt.Errorf("marshal create qdrant collection request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, q.baseURL+"/collections/"+collection.Name, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new create qdrant collection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("call create qdrant collections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create qdrant collections: status %d: %s", resp.StatusCode, b)
	}

	return nil
}

type createCollectionLowerCaseTextIndexRequest struct {
	FieldName   string `json:"field_name"`
	FieldSchema struct {
		Type      string `json:"type"`
		Tokenizer string `json:"tokenizer"`
		Lowercase bool   `json:"lowercase"`
	} `json:"field_schema"`
}

func newCreateCollectionLowerCaseTextIndexRequest(fieldName string, tokenizer string) createCollectionLowerCaseTextIndexRequest {
	return createCollectionLowerCaseTextIndexRequest{
		FieldName: fieldName,
		FieldSchema: struct {
			Type      string `json:"type"`
			Tokenizer string `json:"tokenizer"`
			Lowercase bool   `json:"lowercase"`
		}{
			Type:      "text",
			Tokenizer: tokenizer,
			Lowercase: true,
		},
	}
}

func (q *Qdrant) EnsureCollectionLowercaseTextIndex(ctx context.Context, collectionName string, fieldIndex FieldIndex) error {

	body, err := json.Marshal(newCreateCollectionLowerCaseTextIndexRequest(fieldIndex.Name, fieldIndex.Tokenizer))
	if err != nil {
		return fmt.Errorf("marshal create qdrant collection index request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, q.baseURL+"/collections/"+collectionName+"/index", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("get qdrant collection index request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("call qdrant collection index request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("get qdrant collection index: status %d: %s", resp.StatusCode, b)
	}

	fmt.Printf("collection index ready %s\n", resp.Body)

	return nil
}

type upsertRequest struct {
	Points []Point `json:"points"`
}

type Point struct {
	ID      uint64         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

func (q *Qdrant) Upsert(ctx context.Context, collectionName string, points []Point) error {
	body, err := json.Marshal(upsertRequest{
		Points: points,
	})

	if err != nil {
		return fmt.Errorf("marshal upsert qdrant points request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, q.baseURL+"/collections/"+collectionName+"/points?wait=true", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new upsert qdrant points request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("call upsert qdrant points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("call upsert qdrant points: status %d: %s", resp.StatusCode, b)
	}

	return nil
}

type searchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload,omitempty"`
}

type searchResponse struct {
	Result []SearchResult `json:"result"`
}

type SearchResult struct {
	ID      uint64         `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func (q *Qdrant) Search(ctx context.Context, collectionName string, vector []float32, limit int) ([]SearchResult, error) {

	body, err := json.Marshal(searchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	})

	if err != nil {
		return nil, fmt.Errorf("marshal search qdrant request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.baseURL+"/collections/"+collectionName+"/points/search", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("new search qdrant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call search qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search qdrant: status %d: %s", resp.StatusCode, b)
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode search qdrant response: %w", err)
	}

	return searchResp.Result, nil
}

type TitleHit struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
}

type scrollResponse struct {
	Result struct {
		Points []struct {
			ID      uint64         `json:"id"`
			Payload map[string]any `json:"payload"`
		} `json:"points"`
	} `json:"result"`
}

func (q *Qdrant) LookupByTitle(ctx context.Context, collectionName string, title string, limit int) ([]TitleHit, error) {
	body, _ := json.Marshal(map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "title", "match": map[string]any{"text": title}},
			},
		},
		"with_payload": []string{"title"},
		"with_vector":  false,
		"limit":        limit,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.baseURL+"/collections/"+collectionName+"/points/scroll", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("new scroll qdrant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call scroll qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scroll qdrant: status %d: %s", resp.StatusCode, b)
	}

	var scrollResp scrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&scrollResp); err != nil {
		return nil, fmt.Errorf("decode scroll qdrant response: %w", err)
	}

	results := make([]TitleHit, 0, len(scrollResp.Result.Points))
	for _, point := range scrollResp.Result.Points {
		title, ok := point.Payload["title"].(string)
		if !ok {
			continue
		}
		results = append(results, TitleHit{
			ID:    point.ID,
			Title: title,
		})
	}

	return results, nil
}

func (q *Qdrant) RecommendSimilar(ctx context.Context, collectionName string, itemId uint64, limit int) ([]SearchResult, error) {

	body, _ := json.Marshal(map[string]any{"positive": []uint64{itemId}, "limit": limit, "with_payload": true})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.baseURL+"/collections/"+collectionName+"/points/recommend", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("new recommend qdrant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call recommend qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("recommend qdrant: status %d: %s", resp.StatusCode, b)
	}

	var searchResponse searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("decode recommend qdrant response: %w", err)
	}

	return searchResponse.Result, nil
}
