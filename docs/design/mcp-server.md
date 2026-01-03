# MCP Server Design: Agent Management for Preflight

## Overview

This document outlines the integration of `felixgeelhaar/mcp-go` to expose Preflight functionality to AI agents via the Model Context Protocol (MCP).

## Goals

1. **Parity with CLI** - All MCP tools must provide identical functionality to CLI commands
2. **Type Safety** - Use strongly-typed Go structs for inputs and outputs
3. **Reuse App Layer** - No code duplication; MCP handlers call existing app services
4. **Dual Transport** - Support both stdio (Claude Code) and HTTP (remote agents)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         MCP Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Tool: plan  │  │ Tool: apply │  │ Tool: doctor        │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼─────────────────────┼────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                      App Layer                               │
│   (internal/app/plan.go, apply.go, doctor.go, etc.)         │
└─────────────────────────────────────────────────────────────┘
```

## Proposed MCP Tools

### Core Operations

| Tool | Description | CLI Equivalent |
|------|-------------|----------------|
| `preflight_plan` | Show planned changes | `preflight plan` |
| `preflight_apply` | Apply configuration | `preflight apply` |
| `preflight_doctor` | Verify system state | `preflight doctor` |
| `preflight_diff` | Show config vs system diff | `preflight diff` |
| `preflight_validate` | Validate configuration | `preflight validate` |

### Configuration Management

| Tool | Description | CLI Equivalent |
|------|-------------|----------------|
| `preflight_capture` | Capture machine config | `preflight capture` |
| `preflight_init` | Initialize new config | `preflight init` |
| `preflight_profile` | Switch targets | `preflight profile` |

### Information & Status

| Tool | Description | CLI Equivalent |
|------|-------------|----------------|
| `preflight_status` | Get current state | `preflight status` |
| `preflight_history` | Show applied changes | `preflight history` |
| `preflight_outdated` | Check outdated packages | `preflight outdated` |

### Security & Compliance

| Tool | Description | CLI Equivalent |
|------|-------------|----------------|
| `preflight_security` | Run security scan | `preflight security` |
| `preflight_audit` | View audit logs | `preflight audit` |
| `preflight_compliance` | Generate compliance reports | `preflight compliance` |

## Implementation Structure

```
cmd/preflight-mcp/
├── main.go              # MCP server entrypoint
└── tools/
    ├── plan.go          # preflight_plan tool
    ├── apply.go         # preflight_apply tool
    ├── doctor.go        # preflight_doctor tool
    └── ...

internal/mcp/
├── server.go            # Server configuration
├── middleware.go        # Auth, logging, timeouts
└── types.go             # Shared input/output types
```

## Example Tool Implementation

```go
package tools

import (
    "context"

    "github.com/felixgeelhaar/mcp-go"
    "github.com/felixgeelhaar/preflight/internal/app"
)

type PlanInput struct {
    Target  string `json:"target,omitempty" jsonschema:"description=Target to plan for (e.g. work, personal)"`
    DryRun  bool   `json:"dry_run,omitempty" jsonschema:"description=Show plan without applying"`
    Verbose bool   `json:"verbose,omitempty" jsonschema:"description=Show detailed output"`
}

type PlanOutput struct {
    Steps     []StepInfo `json:"steps"`
    Summary   string     `json:"summary"`
    HasDrift  bool       `json:"has_drift"`
}

func RegisterPlanTool(srv *mcp.Server) {
    srv.Tool("preflight_plan").
        Description("Show what changes preflight would make to your system").
        Handler(func(ctx context.Context, in PlanInput) (*PlanOutput, error) {
            // Use existing app layer
            planner := app.NewPlanner(app.PlannerConfig{
                Target:  in.Target,
                DryRun:  in.DryRun,
                Verbose: in.Verbose,
            })

            plan, err := planner.Plan(ctx)
            if err != nil {
                return nil, err
            }

            return &PlanOutput{
                Steps:    convertSteps(plan.Steps),
                Summary:  plan.Summary(),
                HasDrift: plan.HasDrift(),
            }, nil
        })
}
```

## CLI Integration

Add MCP serve command to existing CLI:

```go
// cmd/preflight/mcp.go
func newMCPCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "mcp",
        Short: "Start MCP server for AI agent integration",
        Long:  "Exposes preflight functionality via Model Context Protocol",
        RunE: func(cmd *cobra.Command, args []string) error {
            srv := mcp.NewServer(mcp.ServerInfo{
                Name:    "preflight",
                Version: version.Version,
            })

            // Register all tools
            tools.RegisterAll(srv)

            return mcp.ServeStdio(cmd.Context(), srv)
        },
    }
}
```

## Configuration

MCP server configuration in `~/.config/claude-code/mcp.json`:

```json
{
  "mcpServers": {
    "preflight": {
      "command": "preflight",
      "args": ["mcp", "serve"],
      "env": {
        "PREFLIGHT_CONFIG": "~/.config/preflight"
      }
    }
  }
}
```

## Safety Considerations

1. **Confirmation for destructive operations** - Apply/clean/rollback require explicit confirmation
2. **Audit logging** - All MCP operations logged to audit trail
3. **Rate limiting** - Prevent rapid-fire operations that could destabilize system
4. **Sandbox mode** - Optional dry-run by default for agent operations

## Implementation Phases

### Phase 1: Core Tools
- `preflight_plan`
- `preflight_apply` (with confirmation)
- `preflight_doctor`
- `preflight_validate`
- `preflight_status`

### Phase 2: Configuration Management
- `preflight_capture`
- `preflight_init`
- `preflight_profile`
- `preflight_diff`

### Phase 3: Advanced Features
- `preflight_security`
- `preflight_audit`
- `preflight_compliance`
- `preflight_fleet` (multi-host)

## Dependencies

```go
require (
    github.com/felixgeelhaar/mcp-go v0.x.x
)
```

## Testing Strategy

1. **Unit tests** - Test each tool handler with mocked app layer
2. **Integration tests** - Full MCP request/response cycles
3. **CLI parity tests** - Verify MCP output matches CLI output
