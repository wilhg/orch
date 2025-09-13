package chromadb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/wilhg/orch/pkg/adapters/vectorstore"
)

// Config controls the ChromaDB adapter behavior.
//
// BaseURL: Chroma server base URL (e.g., http://localhost:8000). Defaults to ORCH_CHROMADB_URL or http://localhost:8000.
// Collection: Single collection name to use for all items. If empty, per-namespace collections are used.
// CreateIfMissing: Whether to create collections automatically when missing. Default true.
type Config struct {
	BaseURL         string
	Collection      string
	CreateIfMissing bool
}

type Store struct {
	baseURL    *url.URL
	useSingle  string
	autoCreate bool
	http       *http.Client

	mu       sync.RWMutex
	nameToID map[string]string // cache: collection name -> id
}

// Register this provider under name "chromadb".
func init() { _ = vectorstore.Register("chromadb", Factory) }

// Factory constructs a ChromaDB-backed VectorStore. Config keys:
// - base_url (string)
// - collection (string)
// - create_if_missing (bool)
func Factory(ctx context.Context, cfg map[string]any) (vectorstore.VectorStore, error) {
	base := os.Getenv("ORCH_CHROMADB_URL")
	if v, ok := cfg["base_url"].(string); ok && v != "" {
		base = v
	}
	if base == "" {
		base = "http://localhost:8000"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("chromadb: invalid base_url: %w", err)
	}

	single := ""
	if v, ok := cfg["collection"].(string); ok {
		single = v
	}
	create := true
	if v, ok := cfg["create_if_missing"].(bool); ok {
		create = v
	}

	return &Store{
		baseURL:    u,
		useSingle:  single,
		autoCreate: create,
		http:       http.DefaultClient,
		nameToID:   make(map[string]string),
	}, nil
}

func (s *Store) Upsert(ctx context.Context, items []vectorstore.Item) error {
	if len(items) == 0 {
		return nil
	}
	// Group by collection name
	groups := map[string][]vectorstore.Item{}
	for _, it := range items {
		coll := s.resolveCollectionName(it.Namespace)
		groups[coll] = append(groups[coll], it)
	}
	for coll, batch := range groups {
		id, err := s.ensureCollection(ctx, coll)
		if err != nil {
			return err
		}
		payload := addRequest{
			IDs:        make([]string, 0, len(batch)),
			Embeddings: make([][]float32, 0, len(batch)),
			Metadatas:  make([]map[string]any, 0, len(batch)),
		}
		for _, it := range batch {
			payload.IDs = append(payload.IDs, it.ID)
			payload.Embeddings = append(payload.Embeddings, []float32(it.Vector))
			if it.Metadata == nil {
				payload.Metadatas = append(payload.Metadatas, map[string]any{})
			} else {
				payload.Metadatas = append(payload.Metadatas, it.Metadata)
			}
		}
		if err := s.postJSON(ctx, path.Join("/api/v1/collections", id, "add"), payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Query(ctx context.Context, query vectorstore.Vector, k int, filter vectorstore.Filter) ([]vectorstore.Match, error) {
	coll := s.resolveCollectionName(filter.Namespace)
	id, err := s.ensureCollection(ctx, coll)
	if err != nil {
		return nil, err
	}
	payload := queryRequest{
		QueryEmbeddings: [][]float32{[]float32(query)},
		NResults:        k,
		Where:           filter.Equals,
		Include:         []string{"distances", "metadatas", "ids"},
	}
	var resp queryResponse
	if err := s.postJSON(ctx, path.Join("/api/v1/collections", id, "query"), payload, &resp); err != nil {
		return nil, err
	}
	// Response fields are nested per query; we sent 1 query, so index 0.
	if len(resp.IDs) == 0 {
		return nil, nil
	}
	ids := resp.IDs[0]
	var dists []float32
	if len(resp.Distances) > 0 {
		dists = resp.Distances[0]
	}
	var metas []map[string]any
	if len(resp.Metadatas) > 0 {
		metas = resp.Metadatas[0]
	}
	out := make([]vectorstore.Match, 0, len(ids))
	for i := range ids {
		var md map[string]any
		if i < len(metas) {
			md = metas[i]
		}
		score := float32(0)
		if i < len(dists) {
			score = -dists[i]
		} // invert distance so higher is more similar
		out = append(out, vectorstore.Match{
			Item:  vectorstore.Item{ID: ids[i], Namespace: filter.Namespace, Metadata: md},
			Score: score,
		})
	}
	return out, nil
}

// Helpers

func (s *Store) resolveCollectionName(namespace string) string {
	if s.useSingle != "" {
		return s.useSingle
	}
	if namespace == "" {
		return "default"
	}
	return namespace
}

func (s *Store) ensureCollection(ctx context.Context, name string) (string, error) {
	s.mu.RLock()
	if id, ok := s.nameToID[name]; ok {
		s.mu.RUnlock()
		return id, nil
	}
	s.mu.RUnlock()
	// Lookup
	var list listCollectionsResponse
	if err := s.getJSON(ctx, "/api/v1/collections?name="+url.QueryEscape(name), &list); err != nil {
		return "", err
	}
	for _, c := range list.Collections {
		if c.Name == name {
			s.mu.Lock()
			s.nameToID[name] = c.ID
			s.mu.Unlock()
			return c.ID, nil
		}
	}
	if !s.autoCreate {
		return "", fmt.Errorf("chromadb: collection %q not found", name)
	}
	// Create (get_or_create semantics vary; we use simple create)
	var created collection
	if err := s.postJSON(ctx, "/api/v1/collections", createCollectionRequest{Name: name}, &created); err != nil {
		return "", err
	}
	s.mu.Lock()
	s.nameToID[name] = created.ID
	s.mu.Unlock()
	return created.ID, nil
}

func (s *Store) endpoint(p string) string {
	u := *s.baseURL
	u.Path = path.Join(u.Path, p)
	return u.String()
}

func (s *Store) getJSON(ctx context.Context, p string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint(p), nil)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("chromadb: GET %s => %s", p, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Store) postJSON(ctx context.Context, p string, body any, out any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint(p), &buf)
	req.Header.Set("content-type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("chromadb: POST %s => %s", p, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Wire types (best-effort minimal shapes)
type collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listCollectionsResponse struct {
	Collections []collection `json:"collections"`
}

type createCollectionRequest struct {
	Name string `json:"name"`
}

type addRequest struct {
	IDs        []string         `json:"ids"`
	Embeddings [][]float32      `json:"embeddings"`
	Metadatas  []map[string]any `json:"metadatas"`
}

type queryRequest struct {
	QueryEmbeddings [][]float32    `json:"query_embeddings"`
	NResults        int            `json:"n_results"`
	Where           map[string]any `json:"where,omitempty"`
	Include         []string       `json:"include,omitempty"`
}

type queryResponse struct {
	IDs       [][]string         `json:"ids"`
	Distances [][]float32        `json:"distances"`
	Metadatas [][]map[string]any `json:"metadatas"`
}
