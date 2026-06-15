// Package urbandictionary is the library behind the urbandictionary command line:
// the HTTP client, request shaping, and the typed data models for the
// Urban Dictionary public API (api.urbandictionary.com).
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public API throws under load.
package urbandictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Host is the API host for Urban Dictionary.
const Host = "api.urbandictionary.com"

// BaseURL is the root every request is built from.
const BaseURL = "https://api.urbandictionary.com/v0"

// DefaultUserAgent identifies the client to Urban Dictionary.
const DefaultUserAgent = "urbandictionary-cli/0.1 (tamnd87@gmail.com)"

// Client talks to the Urban Dictionary API over HTTP.
type Client struct {
	HTTP      *http.Client
	BaseURL   string
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		BaseURL:   BaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
	}
}

// Definition is a single Urban Dictionary entry.
type Definition struct {
	DefID      int    `json:"def_id"`
	Word       string `json:"word"`
	Definition string `json:"definition"`
	Example    string `json:"example"`
	Author     string `json:"author"`
	ThumbsUp   int    `json:"thumbs_up"`
	ThumbsDown int    `json:"thumbs_down"`
	WrittenOn  string `json:"written_on"`
	Permalink  string `json:"permalink"`
}

// Suggestion is a single autocomplete result.
type Suggestion struct {
	Term string `json:"term"`
}

// wire types -----------------------------------------------------------------

type wireDefineResp struct {
	List []wireDefinition `json:"list"`
	Tags []string         `json:"tags"`
}

type wireDefinition struct {
	DefID      int    `json:"defid"`
	Word       string `json:"word"`
	Definition string `json:"definition"`
	Example    string `json:"example"`
	Author     string `json:"author"`
	ThumbsUp   int    `json:"thumbs_up"`
	ThumbsDown int    `json:"thumbs_down"`
	WrittenOn  string `json:"written_on"`
	Permalink  string `json:"permalink"`
}

func wireToDefinition(w wireDefinition) *Definition {
	return &Definition{
		DefID:      w.DefID,
		Word:       w.Word,
		Definition: w.Definition,
		Example:    w.Example,
		Author:     w.Author,
		ThumbsUp:   w.ThumbsUp,
		ThumbsDown: w.ThumbsDown,
		WrittenOn:  w.WrittenOn,
		Permalink:  w.Permalink,
	}
}

// DefineWord looks up a slang term and returns its definitions.
func (c *Client) DefineWord(ctx context.Context, term string) ([]*Definition, error) {
	u := c.BaseURL + "/define?term=" + url.QueryEscape(term)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp wireDefineResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse define response: %w", err)
	}
	out := make([]*Definition, 0, len(resp.List))
	for _, w := range resp.List {
		out = append(out, wireToDefinition(w))
	}
	return out, nil
}

// RandomDefinitions returns random Urban Dictionary definitions.
func (c *Client) RandomDefinitions(ctx context.Context) ([]*Definition, error) {
	body, err := c.Get(ctx, c.BaseURL+"/random")
	if err != nil {
		return nil, err
	}
	var resp wireDefineResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse random response: %w", err)
	}
	out := make([]*Definition, 0, len(resp.List))
	for _, w := range resp.List {
		out = append(out, wireToDefinition(w))
	}
	return out, nil
}

// Autocomplete returns term suggestions for the given prefix.
func (c *Client) Autocomplete(ctx context.Context, prefix string) ([]*Suggestion, error) {
	u := c.BaseURL + "/autocomplete?term=" + url.QueryEscape(prefix)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var terms []string
	if err := json.Unmarshal(body, &terms); err != nil {
		return nil, fmt.Errorf("parse autocomplete response: %w", err)
	}
	out := make([]*Suggestion, 0, len(terms))
	for _, t := range terms {
		out = append(out, &Suggestion{Term: t})
	}
	return out, nil
}

// Get fetches url and returns the response body. It paces and retries according
// to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
