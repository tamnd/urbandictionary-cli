package urbandictionary

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0 // no pacing in the test

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestDefineWord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/define" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"list": []map[string]interface{}{
					{
						"defid":       12345,
						"word":        "test",
						"definition":  "Something you take in school",
						"example":     "I [failed] the [test]",
						"author":      "user123",
						"thumbs_up":   100,
						"thumbs_down": 5,
						"written_on":  "2020-01-01T00:00:00.000Z",
						"permalink":   "https://test.urbanup.com/12345",
					},
				},
			})
		}
	}))
	defer ts.Close()

	c := NewClient()
	c.BaseURL = ts.URL
	c.Rate = 0

	defs, err := c.DefineWord(context.Background(), "test")
	if err != nil {
		t.Fatalf("DefineWord error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
	d := defs[0]
	if d.DefID != 12345 {
		t.Errorf("DefID = %d, want 12345", d.DefID)
	}
	if d.Word != "test" {
		t.Errorf("Word = %q, want test", d.Word)
	}
	if d.ThumbsUp != 100 {
		t.Errorf("ThumbsUp = %d, want 100", d.ThumbsUp)
	}
	if d.Author != "user123" {
		t.Errorf("Author = %q, want user123", d.Author)
	}
}

func TestRandomDefinitions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/random" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"list": []map[string]interface{}{
					{
						"defid":       99,
						"word":        "random",
						"definition":  "Something unpredictable",
						"example":     "That was [random]",
						"author":      "someone",
						"thumbs_up":   50,
						"thumbs_down": 2,
						"written_on":  "2021-06-01T00:00:00.000Z",
						"permalink":   "https://random.urbanup.com/99",
					},
				},
			})
		}
	}))
	defer ts.Close()

	c := NewClient()
	c.BaseURL = ts.URL
	c.Rate = 0

	defs, err := c.RandomDefinitions(context.Background())
	if err != nil {
		t.Fatalf("RandomDefinitions error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
	if defs[0].Word != "random" {
		t.Errorf("Word = %q, want random", defs[0].Word)
	}
}

func TestAutocomplete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/autocomplete" {
			_ = json.NewEncoder(w).Encode([]string{"Code Red", "code", "Code Blue"})
		}
	}))
	defer ts.Close()

	c := NewClient()
	c.BaseURL = ts.URL
	c.Rate = 0

	suggestions, err := c.Autocomplete(context.Background(), "code")
	if err != nil {
		t.Fatalf("Autocomplete error: %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("got %d suggestions, want 3", len(suggestions))
	}
	if suggestions[0].Term != "Code Red" {
		t.Errorf("Term = %q, want Code Red", suggestions[0].Term)
	}
}
