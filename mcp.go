package main

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// mcpHandlers binds the read-only store to tool implementations so each
// handler is a plain method that tests can call directly.
type mcpHandlers struct {
	s      *store.Store
	loc    *time.Location
	dbPath string
}

func (h mcpHandlers) stats() (store.Stats, error) {
	return h.s.Stats(h.dbPath)
}

type emptyArgs struct{}

// newMCPServer registers every read-only tool on a fresh MCP server.
func newMCPServer(s *store.Store, loc *time.Location, dbPath string) *mcp.Server {
	h := mcpHandlers{s: s, loc: loc, dbPath: dbPath}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "limitless-catalog",
		Version: "1.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "catalog_stats",
		Description: "Catalog coverage: totals, date range, hours, per-month counts, gap days, categories.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, store.Stats, error) {
		st, err := h.stats()
		if err != nil {
			return nil, store.Stats{}, err
		}
		return nil, st, nil
	})

	return server
}

// cmdMCP serves the catalog to MCP clients over stdio, read-only.
func cmdMCP(cfg config, args []string) error {
	s, err := store.OpenReadOnly(cfg.dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	server := newMCPServer(s, cfg.loc, cfg.dbPath)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
