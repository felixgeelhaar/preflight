package tour

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgress(t *testing.T) {
	p := NewProgress()

	assert.NotNil(t, p.Topics)
	assert.Empty(t, p.Topics)
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestProgress_StartTopic(t *testing.T) {
	p := NewProgress()

	p.StartTopic("basics", 5)

	tp := p.GetTopicProgress("basics")
	require.NotNil(t, tp)
	assert.Equal(t, "basics", tp.ID)
	assert.Equal(t, 5, tp.TotalSections)
	assert.Empty(t, tp.CompletedSections)
	assert.False(t, tp.StartedAt.IsZero())
	assert.True(t, tp.CompletedAt.IsZero())
}

func TestProgress_StartTopic_AlreadyStarted(t *testing.T) {
	p := NewProgress()

	p.StartTopic("basics", 5)
	originalStartTime := p.Topics["basics"].StartedAt

	// Starting again should not reset
	p.StartTopic("basics", 10)

	tp := p.GetTopicProgress("basics")
	assert.Equal(t, originalStartTime, tp.StartedAt)
	// But should update total sections if it was 0
	assert.Equal(t, 5, tp.TotalSections)
}

func TestProgress_CompleteSection(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 3)

	p.CompleteSection("basics", 0)

	tp := p.GetTopicProgress("basics")
	assert.Contains(t, tp.CompletedSections, 0)
	assert.Len(t, tp.CompletedSections, 1)
}

func TestProgress_CompleteSection_Duplicate(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 3)

	p.CompleteSection("basics", 0)
	p.CompleteSection("basics", 0) // Duplicate

	tp := p.GetTopicProgress("basics")
	assert.Len(t, tp.CompletedSections, 1)
}

func TestProgress_CompleteSection_CompletesTotalTopic(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 2)

	p.CompleteSection("basics", 0)
	assert.True(t, p.Topics["basics"].CompletedAt.IsZero())

	p.CompleteSection("basics", 1)
	assert.False(t, p.Topics["basics"].CompletedAt.IsZero())
}

func TestProgress_CompleteSection_UnknownTopic(t *testing.T) {
	p := NewProgress()

	// Should not panic
	p.CompleteSection("unknown", 0)

	assert.Nil(t, p.GetTopicProgress("unknown"))
}

func TestProgress_IsSectionCompleted(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 3)

	assert.False(t, p.IsSectionCompleted("basics", 0))

	p.CompleteSection("basics", 0)
	assert.True(t, p.IsSectionCompleted("basics", 0))
	assert.False(t, p.IsSectionCompleted("basics", 1))
}

func TestProgress_IsSectionCompleted_UnknownTopic(t *testing.T) {
	p := NewProgress()

	assert.False(t, p.IsSectionCompleted("unknown", 0))
}

func TestProgress_IsTopicCompleted(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 2)

	assert.False(t, p.IsTopicCompleted("basics"))

	p.CompleteSection("basics", 0)
	assert.False(t, p.IsTopicCompleted("basics"))

	p.CompleteSection("basics", 1)
	assert.True(t, p.IsTopicCompleted("basics"))
}

func TestProgress_IsTopicCompleted_UnknownTopic(t *testing.T) {
	p := NewProgress()

	assert.False(t, p.IsTopicCompleted("unknown"))
}

func TestProgress_IsTopicStarted(t *testing.T) {
	p := NewProgress()

	assert.False(t, p.IsTopicStarted("basics"))

	p.StartTopic("basics", 3)
	assert.True(t, p.IsTopicStarted("basics"))
}

func TestProgress_TopicCompletionPercent(t *testing.T) {
	tests := []struct {
		name             string
		totalSections    int
		completeSections []int
		expected         int
	}{
		{"no progress", 4, nil, 0},
		{"25%", 4, []int{0}, 25},
		{"50%", 4, []int{0, 1}, 50},
		{"75%", 4, []int{0, 1, 2}, 75},
		{"100%", 4, []int{0, 1, 2, 3}, 100},
		{"33%", 3, []int{0}, 33},
		{"66%", 3, []int{0, 1}, 66},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress()
			p.StartTopic("test", tt.totalSections)
			for _, idx := range tt.completeSections {
				p.CompleteSection("test", idx)
			}

			assert.Equal(t, tt.expected, p.TopicCompletionPercent("test"))
		})
	}
}

func TestProgress_TopicCompletionPercent_UnknownTopic(t *testing.T) {
	p := NewProgress()

	assert.Equal(t, 0, p.TopicCompletionPercent("unknown"))
}

func TestProgress_OverallCompletionPercent(t *testing.T) {
	p := NewProgress()

	// No topics
	assert.Equal(t, 0, p.OverallCompletionPercent(0))
	assert.Equal(t, 0, p.OverallCompletionPercent(6))

	// Start and complete one topic
	p.StartTopic("basics", 2)
	p.CompleteSection("basics", 0)
	p.CompleteSection("basics", 1)

	assert.Equal(t, 16, p.OverallCompletionPercent(6)) // 1/6 = 16%

	// Complete another
	p.StartTopic("config", 3)
	p.CompleteSection("config", 0)
	p.CompleteSection("config", 1)
	p.CompleteSection("config", 2)

	assert.Equal(t, 33, p.OverallCompletionPercent(6)) // 2/6 = 33%
}

func TestProgress_CompletedTopicsCount(t *testing.T) {
	p := NewProgress()

	assert.Equal(t, 0, p.CompletedTopicsCount())

	p.StartTopic("basics", 2)
	p.CompleteSection("basics", 0)
	p.CompleteSection("basics", 1)

	assert.Equal(t, 1, p.CompletedTopicsCount())

	p.StartTopic("config", 2)
	p.CompleteSection("config", 0)

	assert.Equal(t, 1, p.CompletedTopicsCount()) // config not complete

	p.CompleteSection("config", 1)
	assert.Equal(t, 2, p.CompletedTopicsCount())
}

func TestProgress_Reset(t *testing.T) {
	p := NewProgress()
	p.StartTopic("basics", 3)
	p.CompleteSection("basics", 0)

	p.Reset()

	assert.Empty(t, p.Topics)
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestProgressStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "progress.json")
	store := NewProgressStoreWithPath(path)

	// Create progress
	p := NewProgress()
	p.StartTopic("basics", 3)
	p.CompleteSection("basics", 0)
	p.CompleteSection("basics", 1)

	// Save
	err := store.Save(p)
	require.NoError(t, err)

	// Load
	loaded, err := store.Load()
	require.NoError(t, err)

	assert.Len(t, loaded.Topics, 1)
	tp := loaded.GetTopicProgress("basics")
	require.NotNil(t, tp)
	assert.Equal(t, 3, tp.TotalSections)
	assert.Len(t, tp.CompletedSections, 2)
	assert.Contains(t, tp.CompletedSections, 0)
	assert.Contains(t, tp.CompletedSections, 1)
}

func TestProgressStore_Load_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")
	store := NewProgressStoreWithPath(path)

	p, err := store.Load()
	require.NoError(t, err)

	assert.NotNil(t, p)
	assert.Empty(t, p.Topics)
}

func TestProgressStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(path, []byte("not json"), 0644)
	require.NoError(t, err)

	store := NewProgressStoreWithPath(path)
	_, err = store.Load()

	assert.Error(t, err)
}

func TestProgressStore_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "progress.json")
	store := NewProgressStoreWithPath(path)

	p := NewProgress()
	err := store.Save(p)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestProgressStore_Path(t *testing.T) {
	store := NewProgressStoreWithPath("/custom/path.json")
	assert.Equal(t, "/custom/path.json", store.Path())
}

func TestNewProgressStore(t *testing.T) {
	store, err := NewProgressStore()
	require.NoError(t, err)

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".preflight", "tour-progress.json")
	assert.Equal(t, expected, store.Path())
}

func TestGetTopicProgress_NotFound(t *testing.T) {
	p := NewProgress()

	tp := p.GetTopicProgress("nonexistent")
	assert.Nil(t, tp)
}
