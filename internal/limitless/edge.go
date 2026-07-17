package limitless

import "context"

// FetchFirst returns the first lifelog of a single page matching p, or
// nil when the page is empty. It never follows the cursor.
func (c *Client) FetchFirst(ctx context.Context, p ListParams) (*Lifelog, error) {
	logs, _, err := c.fetchPage(ctx, p, "")
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[0], nil
}

// FetchPage fetches one page matching p at the given cursor, returning
// the page's lifelogs and the next cursor (empty when exhausted).
func (c *Client) FetchPage(ctx context.Context, p ListParams, cursor string) ([]Lifelog, string, error) {
	return c.fetchPage(ctx, p, cursor)
}

// FetchEdge returns the single oldest (direction "asc") or newest
// (direction "desc") lifelog the account has, or nil when there are none.
func (c *Client) FetchEdge(ctx context.Context, direction string) (*Lifelog, error) {
	return c.FetchFirst(ctx, ListParams{Direction: direction})
}
