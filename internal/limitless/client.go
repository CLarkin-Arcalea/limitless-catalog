package limitless

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is the production Limitless API host.
const DefaultBaseURL = "https://api.limitless.ai"

const maxAttempts = 5

// Client calls the Limitless Developer API.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
	// Sleep is time.Sleep, injectable so retry tests run instantly.
	Sleep func(time.Duration)
}

// NewClient returns a Client. Empty baseURL means production.
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		Sleep:   time.Sleep,
	}
}

// ListParams narrow a lifelogs listing. Date XOR Start/End.
type ListParams struct {
	Date      string // YYYY-MM-DD
	Start     string // YYYY-MM-DD or "YYYY-MM-DD HH:MM:SS"
	End       string
	Timezone  string // IANA name; API defaults to UTC
	Direction string // asc|desc; API defaults to desc
}

// FetchAll pages through every lifelog matching p and returns them all.
// It always requests limit=10 (the API max; default is only 3) and
// includeContents=true (default is false, and we need speaker blocks).
func (c *Client) FetchAll(ctx context.Context, p ListParams) ([]Lifelog, error) {
	var out []Lifelog
	cursor := ""
	for {
		logs, next, err := c.fetchPage(ctx, p, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, logs...)
		if next == "" {
			return out, nil
		}
		cursor = next
	}
}

func (c *Client) fetchPage(ctx context.Context, p ListParams, cursor string) ([]Lifelog, string, error) {
	q := url.Values{}
	q.Set("limit", "10")
	q.Set("includeMarkdown", "true")
	q.Set("includeHeadings", "true")
	q.Set("includeContents", "true")
	if p.Date != "" {
		q.Set("date", p.Date)
	}
	if p.Start != "" {
		q.Set("start", p.Start)
	}
	if p.End != "" {
		q.Set("end", p.End)
	}
	if p.Timezone != "" {
		q.Set("timezone", p.Timezone)
	}
	if p.Direction != "" {
		q.Set("direction", p.Direction)
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}

	body, err := c.getWithRetry(ctx, c.BaseURL+"/v1/lifelogs?"+q.Encode())
	if err != nil {
		return nil, "", err
	}
	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("decode response: %w", err)
	}
	return resp.Data.Lifelogs, resp.Meta.Lifelogs.NextCursor, nil
}

// getWithRetry GETs u, retrying 429 (honoring Retry-After) and 5xx/network
// errors with capped exponential backoff. 4xx other than 429 is fatal.
func (c *Client) getWithRetry(ctx context.Context, u string) ([]byte, error) {
	backoff := time.Second
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-API-Key", c.APIKey)

		res, err := c.HTTP.Do(req)
		if err != nil {
			if attempt >= maxAttempts {
				return nil, fmt.Errorf("request failed after %d attempts: %w", attempt, err)
			}
			c.Sleep(backoff)
			backoff *= 2
			continue
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()

		switch {
		case res.StatusCode == http.StatusOK:
			if readErr != nil {
				return nil, fmt.Errorf("read body: %w", readErr)
			}
			return body, nil
		case res.StatusCode == http.StatusTooManyRequests:
			if attempt >= maxAttempts {
				return nil, fmt.Errorf("rate limited after %d attempts", attempt)
			}
			c.Sleep(retryAfter(res, backoff))
			backoff *= 2
		case res.StatusCode >= 500:
			if attempt >= maxAttempts {
				return nil, fmt.Errorf("server error %d after %d attempts", res.StatusCode, attempt)
			}
			c.Sleep(backoff)
			backoff *= 2
		default:
			return nil, fmt.Errorf("api error %d: %s", res.StatusCode, string(body))
		}
	}
}

func retryAfter(res *http.Response, fallback time.Duration) time.Duration {
	if v := res.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return fallback
}
