package limitless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type singleResponse struct {
	Data struct {
		Lifelog Lifelog `json:"lifelog"`
	} `json:"data"`
}

// GetLifelog fetches one lifelog by id via GET /v1/lifelogs/{id}, with
// contents included. Useful for records unreachable by date queries.
func (c *Client) GetLifelog(ctx context.Context, id string) (*Lifelog, error) {
	if id == "" {
		return nil, fmt.Errorf("empty lifelog id")
	}
	q := url.Values{}
	q.Set("includeMarkdown", "true")
	q.Set("includeHeadings", "true")
	q.Set("includeContents", "true")

	body, err := c.getWithRetry(ctx, c.BaseURL+"/v1/lifelogs/"+url.PathEscape(id)+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	var resp singleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode lifelog %s: %w", id, err)
	}
	if resp.Data.Lifelog.ID == "" {
		return nil, fmt.Errorf("lifelog %s: empty response", id)
	}
	return &resp.Data.Lifelog, nil
}
