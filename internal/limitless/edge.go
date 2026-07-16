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

// FetchEdge returns the single oldest (direction "asc") or newest
// (direction "desc") lifelog the account has, or nil when there are none.
func (c *Client) FetchEdge(ctx context.Context, direction string) (*Lifelog, error) {
	return c.FetchFirst(ctx, ListParams{Direction: direction})
}
