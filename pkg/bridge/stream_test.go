package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamJSONL_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values":     []any{map[string]any{"id": 1}, map[string]any{"id": 2}},
			"isLastPage": true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	items, err := CollectPage(c, "/api/repos", "values", "isLastPage", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestStreamJSONL_MultiPage(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		isLast := page >= 2
		json.NewEncoder(w).Encode(map[string]any{
			"values":     []any{map[string]any{"page": page}},
			"isLastPage": isLast,
			"nextPageStart": page * 25,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	items, err := CollectPage(c, "/api/repos", "values", "isLastPage", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items (one per page), got %d", len(items))
	}
}
