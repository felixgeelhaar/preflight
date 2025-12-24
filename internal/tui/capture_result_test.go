package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptureItemResult_Fields(t *testing.T) {
	t.Parallel()

	t.Run("contains all required fields", func(t *testing.T) {
		t.Parallel()
		result := CaptureItemResult{
			Name:     "git",
			Category: "brew",
			Type:     CaptureTypeFormula,
			Layer:    "base",
			Value:    "git",
		}

		assert.Equal(t, "git", result.Name)
		assert.Equal(t, "brew", result.Category)
		assert.Equal(t, CaptureTypeFormula, result.Type)
		assert.Equal(t, "base", result.Layer)
		assert.Equal(t, "git", result.Value)
	})
}

func TestCaptureReviewResult_RichTypes(t *testing.T) {
	t.Parallel()

	t.Run("accepted items preserve full information", func(t *testing.T) {
		t.Parallel()
		result := CaptureReviewResult{
			AcceptedItems: []CaptureItemResult{
				{Name: "git", Category: "brew", Type: CaptureTypeFormula, Layer: "base", Value: "git"},
				{Name: "nvim", Category: "brew", Type: CaptureTypeFormula, Layer: "base", Value: "neovim"},
			},
			RejectedItems: []CaptureItemResult{},
			Cancelled:     false,
		}

		assert.Len(t, result.AcceptedItems, 2)
		assert.Equal(t, "git", result.AcceptedItems[0].Name)
		assert.Equal(t, "brew", result.AcceptedItems[0].Category)
		assert.Equal(t, CaptureTypeFormula, result.AcceptedItems[0].Type)
		assert.Equal(t, "base", result.AcceptedItems[0].Layer)
	})

	t.Run("rejected items preserve full information", func(t *testing.T) {
		t.Parallel()
		result := CaptureReviewResult{
			AcceptedItems: []CaptureItemResult{},
			RejectedItems: []CaptureItemResult{
				{Name: "wget", Category: "brew", Type: CaptureTypeFormula, Layer: "captured", Value: "wget"},
			},
			Cancelled: false,
		}

		assert.Len(t, result.RejectedItems, 1)
		assert.Equal(t, "wget", result.RejectedItems[0].Name)
		assert.Equal(t, "captured", result.RejectedItems[0].Layer)
	})
}

func TestToCaptureItemResult(t *testing.T) {
	t.Parallel()

	t.Run("converts CaptureItem to CaptureItemResult", func(t *testing.T) {
		t.Parallel()
		item := CaptureItem{
			Name:     "git",
			Category: "brew",
			Type:     CaptureTypeFormula,
			Details:  "Distributed version control",
			Value:    "git",
			Layer:    "base",
		}

		result := ToCaptureItemResult(item)

		assert.Equal(t, item.Name, result.Name)
		assert.Equal(t, item.Category, result.Category)
		assert.Equal(t, item.Type, result.Type)
		assert.Equal(t, item.Layer, result.Layer)
		assert.Equal(t, item.Value, result.Value)
	})

	t.Run("uses default layer when empty", func(t *testing.T) {
		t.Parallel()
		item := CaptureItem{
			Name:     "git",
			Category: "brew",
			Type:     CaptureTypeFormula,
			// Layer is empty
		}

		result := ToCaptureItemResult(item)

		assert.Equal(t, "captured", result.Layer)
	})
}

func TestToCaptureItemResults(t *testing.T) {
	t.Parallel()

	t.Run("converts slice of CaptureItems", func(t *testing.T) {
		t.Parallel()
		items := []CaptureItem{
			{Name: "git", Category: "brew", Type: CaptureTypeFormula, Layer: "base"},
			{Name: "nvim", Category: "brew", Type: CaptureTypeFormula, Layer: "base"},
			{Name: "starship", Category: "brew", Type: CaptureTypeFormula}, // No layer
		}

		results := ToCaptureItemResults(items)

		assert.Len(t, results, 3)
		assert.Equal(t, "git", results[0].Name)
		assert.Equal(t, "base", results[0].Layer)
		assert.Equal(t, "nvim", results[1].Name)
		assert.Equal(t, "starship", results[2].Name)
		assert.Equal(t, "captured", results[2].Layer) // Default layer
	})

	t.Run("returns empty slice for nil input", func(t *testing.T) {
		t.Parallel()
		results := ToCaptureItemResults(nil)
		assert.NotNil(t, results)
		assert.Len(t, results, 0)
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()
		results := ToCaptureItemResults([]CaptureItem{})
		assert.NotNil(t, results)
		assert.Len(t, results, 0)
	})
}
