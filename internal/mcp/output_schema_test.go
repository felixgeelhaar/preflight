package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.klarlabs.de/mcp/schema"
)

// TestOutputSchemaGenerates guards the output schemas advertised by the
// data-returning tools. OutputSchema runs schema.Generate at registration
// time; if generation errors, the ToolBuilder captures the error and the
// tool is silently dropped from the server. This test fails loudly instead,
// so a future change to any advertised output type that breaks schema
// generation is caught immediately.
func TestOutputSchemaGenerates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		example any
	}{
		{"PlanOutput", PlanOutput{}},
		{"DoctorOutput", DoctorOutput{}},
		{"ValidateOutput", ValidateOutput{}},
		{"StatusOutput", StatusOutput{}},
		{"DiffOutput", DiffOutput{}},
		{"TourOutput", TourOutput{}},
		{"SecurityOutput", SecurityOutput{}},
		{"OutdatedOutput", OutdatedOutput{}},
		{"MarketplaceOutput", MarketplaceOutput{}},
		{"ToolAnalyzeOutput", ToolAnalyzeOutput{}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := schema.Generate(tt.example)
			require.NoError(t, err, "schema.Generate must not error for advertised output type %s", tt.name)
			require.NotNil(t, s, "schema.Generate must return a schema for %s", tt.name)
		})
	}
}
