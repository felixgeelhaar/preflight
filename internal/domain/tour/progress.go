// Package tour provides tour progress tracking and persistence.
package tour

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Progress tracks user's tour completion state.
type Progress struct {
	Topics    map[string]*TopicProgress `json:"topics"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

// TopicProgress tracks completion for a single topic.
type TopicProgress struct {
	ID                string    `json:"id"`
	CompletedSections []int     `json:"completed_sections"`
	TotalSections     int       `json:"total_sections"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	CompletedAt       time.Time `json:"completed_at,omitempty"`
}

// NewProgress creates a new empty progress tracker.
func NewProgress() *Progress {
	return &Progress{
		Topics:    make(map[string]*TopicProgress),
		UpdatedAt: time.Now(),
	}
}

// GetTopicProgress returns progress for a specific topic.
func (p *Progress) GetTopicProgress(topicID string) *TopicProgress {
	if tp, ok := p.Topics[topicID]; ok {
		return tp
	}
	return nil
}

// StartTopic marks a topic as started.
func (p *Progress) StartTopic(topicID string, totalSections int) {
	if _, ok := p.Topics[topicID]; !ok {
		p.Topics[topicID] = &TopicProgress{
			ID:                topicID,
			CompletedSections: []int{},
			TotalSections:     totalSections,
			StartedAt:         time.Now(),
		}
	} else if p.Topics[topicID].TotalSections == 0 {
		p.Topics[topicID].TotalSections = totalSections
	}
	p.UpdatedAt = time.Now()
}

// CompleteSection marks a section as completed.
func (p *Progress) CompleteSection(topicID string, sectionIndex int) {
	tp := p.Topics[topicID]
	if tp == nil {
		return
	}

	// Check if already completed
	for _, idx := range tp.CompletedSections {
		if idx == sectionIndex {
			return
		}
	}

	tp.CompletedSections = append(tp.CompletedSections, sectionIndex)
	p.UpdatedAt = time.Now()

	// Check if topic is now complete
	if len(tp.CompletedSections) >= tp.TotalSections {
		tp.CompletedAt = time.Now()
	}
}

// IsSectionCompleted checks if a specific section is completed.
func (p *Progress) IsSectionCompleted(topicID string, sectionIndex int) bool {
	tp := p.Topics[topicID]
	if tp == nil {
		return false
	}
	for _, idx := range tp.CompletedSections {
		if idx == sectionIndex {
			return true
		}
	}
	return false
}

// IsTopicCompleted checks if a topic is fully completed.
func (p *Progress) IsTopicCompleted(topicID string) bool {
	tp := p.Topics[topicID]
	if tp == nil {
		return false
	}
	return len(tp.CompletedSections) >= tp.TotalSections && tp.TotalSections > 0
}

// IsTopicStarted checks if a topic has been started.
func (p *Progress) IsTopicStarted(topicID string) bool {
	tp := p.Topics[topicID]
	return tp != nil && !tp.StartedAt.IsZero()
}

// TopicCompletionPercent returns the completion percentage for a topic.
func (p *Progress) TopicCompletionPercent(topicID string) int {
	tp := p.Topics[topicID]
	if tp == nil || tp.TotalSections == 0 {
		return 0
	}
	return (len(tp.CompletedSections) * 100) / tp.TotalSections
}

// OverallCompletionPercent returns overall completion across all known topics.
func (p *Progress) OverallCompletionPercent(totalTopics int) int {
	if totalTopics == 0 {
		return 0
	}
	completed := 0
	for _, tp := range p.Topics {
		if tp.TotalSections > 0 && len(tp.CompletedSections) >= tp.TotalSections {
			completed++
		}
	}
	return (completed * 100) / totalTopics
}

// CompletedTopicsCount returns the number of fully completed topics.
func (p *Progress) CompletedTopicsCount() int {
	count := 0
	for _, tp := range p.Topics {
		if tp.TotalSections > 0 && len(tp.CompletedSections) >= tp.TotalSections {
			count++
		}
	}
	return count
}

// Reset clears all progress.
func (p *Progress) Reset() {
	p.Topics = make(map[string]*TopicProgress)
	p.UpdatedAt = time.Now()
}

// ProgressStore handles persistence of tour progress.
type ProgressStore struct {
	path string
	mu   sync.RWMutex
}

// NewProgressStore creates a store with the default path.
func NewProgressStore() (*ProgressStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(homeDir, ".preflight", "tour-progress.json")
	return &ProgressStore{path: path}, nil
}

// NewProgressStoreWithPath creates a store with a custom path.
func NewProgressStoreWithPath(path string) *ProgressStore {
	return &ProgressStore{path: path}
}

// Load reads progress from disk.
func (s *ProgressStore) Load() (*Progress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewProgress(), nil
		}
		return nil, err
	}

	var progress Progress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}

	// Ensure map is initialized
	if progress.Topics == nil {
		progress.Topics = make(map[string]*TopicProgress)
	}

	return &progress, nil
}

// Save writes progress to disk.
func (s *ProgressStore) Save(progress *Progress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Path returns the storage path.
func (s *ProgressStore) Path() string {
	return s.path
}
