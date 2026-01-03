package main

import (
	"context"
	"os"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/preflight/internal/app"
	mcptools "github.com/felixgeelhaar/preflight/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI agent integration",
	Long: `Start a Model Context Protocol (MCP) server for AI agent integration.

The MCP server exposes preflight functionality to AI agents (like Claude Code)
via the Model Context Protocol, enabling intelligent configuration management.

Available tools:
  - preflight_plan      Show what changes would be made
  - preflight_apply     Apply configuration changes
  - preflight_doctor    Verify system state
  - preflight_validate  Validate configuration
  - preflight_status    Get current status

Examples:
  preflight mcp                     # Start stdio MCP server
  preflight mcp --http :8080        # Start HTTP MCP server
  preflight mcp --config path.yaml  # Use specific config file`,
	RunE: runMCP,
}

var (
	mcpHTTP       string
	mcpConfigPath string
	mcpTarget     string
)

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().StringVar(&mcpHTTP, "http", "", "Start HTTP server on address (e.g., :8080)")
	mcpCmd.Flags().StringVarP(&mcpConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	mcpCmd.Flags().StringVarP(&mcpTarget, "target", "t", "default", "Default target")
}

func runMCP(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the preflight application
	preflight := app.New(os.Stdout)

	// Create MCP server
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight",
		Version: version,
	})

	// Register all tools
	mcptools.RegisterAll(srv, preflight, mcpConfigPath, mcpTarget)

	// Serve based on transport
	if mcpHTTP != "" {
		return mcp.ServeHTTP(ctx, srv, mcpHTTP)
	}

	// Default to stdio
	return mcp.ServeStdio(ctx, srv)
}
