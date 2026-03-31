package vibesort

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSortStrings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"order":[2,0,1]}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	c, err := NewClient("test-key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.SortStrings(context.Background(), []string{"compiler", "linter", "vibes"}, "sort by chaotic energy")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"vibes", "compiler", "linter"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected order: got %v want %v", got, want)
	}
}

func TestSortStringsRejectsBadPermutation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"order":[0,0,1]}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	c, err := NewClient("test-key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SortStrings(context.Background(), []string{"a", "b", "c"}, "anything")
	if err == nil {
		t.Fatal("expected error for duplicate indexes")
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for missing api key")
	}
}
