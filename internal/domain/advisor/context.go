package advisor

import (
	"errors"
	"fmt"
	"strings"
)

// Context errors.
var (
	ErrEmptyCategory     = errors.New("category cannot be empty")
	ErrInvalidProfile    = errors.New("user profile is invalid")
	ErrInvalidExperience = errors.New("invalid experience level")
)

// ExperienceLevel indicates the user's experience level.
type ExperienceLevel string

// Experience level constants.
const (
	ExperienceBeginner     ExperienceLevel = "beginner"
	ExperienceIntermediate ExperienceLevel = "intermediate"
	ExperienceAdvanced     ExperienceLevel = "advanced"
)

// WorkContext indicates whether this is a work or personal setup.
type WorkContext string

// Work context constants.
const (
	WorkContextUnknown  WorkContext = "unknown"
	WorkContextWork     WorkContext = "work"
	WorkContextPersonal WorkContext = "personal"
	WorkContextMixed    WorkContext = "mixed" // Both work and personal use
)

// String returns the work context as a string.
func (w WorkContext) String() string {
	return string(w)
}

// IsValid returns true if this is a known work context.
func (w WorkContext) IsValid() bool {
	switch w {
	case WorkContextUnknown, WorkContextWork, WorkContextPersonal, WorkContextMixed:
		return true
	default:
		return false
	}
}

// DeviceType indicates the type of device.
type DeviceType string

// Device type constants.
const (
	DeviceTypeUnknown DeviceType = "unknown"
	DeviceTypeLaptop  DeviceType = "laptop"
	DeviceTypeDesktop DeviceType = "desktop"
)

// String returns the device type as a string.
func (d DeviceType) String() string {
	return string(d)
}

// IsValid returns true if this is a known device type.
func (d DeviceType) IsValid() bool {
	switch d {
	case DeviceTypeUnknown, DeviceTypeLaptop, DeviceTypeDesktop:
		return true
	default:
		return false
	}
}

// String returns the experience level as a string.
func (e ExperienceLevel) String() string {
	return string(e)
}

// IsValid returns true if this is a known experience level.
func (e ExperienceLevel) IsValid() bool {
	switch e {
	case ExperienceBeginner, ExperienceIntermediate, ExperienceAdvanced:
		return true
	default:
		return false
	}
}

// ParseExperienceLevel parses a string into an ExperienceLevel.
func ParseExperienceLevel(s string) (ExperienceLevel, error) {
	level := ExperienceLevel(strings.ToLower(s))
	if !level.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidExperience, s)
	}
	return level, nil
}

// UserProfile contains information about the user for personalized recommendations.
type UserProfile struct {
	experience      ExperienceLevel
	preferences     map[string]string
	existingTools   []string
	operatingSystem string
	workContext     WorkContext
	deviceType      DeviceType
	emailDomains    []string // Detected email domains (for work/personal inference)
	teamSize        int      // 0 = unknown, 1 = solo, 2+ = team
}

// NewUserProfile creates a new UserProfile.
func NewUserProfile(experience ExperienceLevel) (UserProfile, error) {
	return UserProfile{
		experience:    experience,
		preferences:   map[string]string{},
		existingTools: []string{},
	}, nil
}

// Experience returns the user's experience level.
func (p UserProfile) Experience() ExperienceLevel {
	return p.experience
}

// Preferences returns user preferences as key-value pairs.
func (p UserProfile) Preferences() map[string]string {
	result := make(map[string]string, len(p.preferences))
	for k, v := range p.preferences {
		result[k] = v
	}
	return result
}

// ExistingTools returns tools the user already has installed.
func (p UserProfile) ExistingTools() []string {
	result := make([]string, len(p.existingTools))
	copy(result, p.existingTools)
	return result
}

// OperatingSystem returns the user's operating system.
func (p UserProfile) OperatingSystem() string {
	return p.operatingSystem
}

// WorkContext returns the detected work context.
func (p UserProfile) WorkContext() WorkContext {
	return p.workContext
}

// DeviceType returns the detected device type.
func (p UserProfile) DeviceType() DeviceType {
	return p.deviceType
}

// EmailDomains returns detected email domains.
func (p UserProfile) EmailDomains() []string {
	result := make([]string, len(p.emailDomains))
	copy(result, p.emailDomains)
	return result
}

// TeamSize returns the detected team size (0 = unknown, 1 = solo, 2+ = team).
func (p UserProfile) TeamSize() int {
	return p.teamSize
}

// WithPreferences returns a new UserProfile with preferences set.
func (p UserProfile) WithPreferences(prefs map[string]string) UserProfile {
	newPrefs := make(map[string]string, len(prefs))
	for k, v := range prefs {
		newPrefs[k] = v
	}

	return UserProfile{
		experience:      p.experience,
		preferences:     newPrefs,
		existingTools:   p.existingTools,
		operatingSystem: p.operatingSystem,
		workContext:     p.workContext,
		deviceType:      p.deviceType,
		emailDomains:    p.emailDomains,
		teamSize:        p.teamSize,
	}
}

// WithExistingTools returns a new UserProfile with existing tools set.
func (p UserProfile) WithExistingTools(tools []string) UserProfile {
	newTools := make([]string, len(tools))
	copy(newTools, tools)

	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   newTools,
		operatingSystem: p.operatingSystem,
		workContext:     p.workContext,
		deviceType:      p.deviceType,
		emailDomains:    p.emailDomains,
		teamSize:        p.teamSize,
	}
}

// WithOperatingSystem returns a new UserProfile with OS set.
func (p UserProfile) WithOperatingSystem(os string) UserProfile {
	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   p.existingTools,
		operatingSystem: os,
		workContext:     p.workContext,
		deviceType:      p.deviceType,
		emailDomains:    p.emailDomains,
		teamSize:        p.teamSize,
	}
}

// WithWorkContext returns a new UserProfile with work context set.
func (p UserProfile) WithWorkContext(ctx WorkContext) UserProfile {
	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   p.existingTools,
		operatingSystem: p.operatingSystem,
		workContext:     ctx,
		deviceType:      p.deviceType,
		emailDomains:    p.emailDomains,
		teamSize:        p.teamSize,
	}
}

// WithDeviceType returns a new UserProfile with device type set.
func (p UserProfile) WithDeviceType(dt DeviceType) UserProfile {
	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   p.existingTools,
		operatingSystem: p.operatingSystem,
		workContext:     p.workContext,
		deviceType:      dt,
		emailDomains:    p.emailDomains,
		teamSize:        p.teamSize,
	}
}

// WithEmailDomains returns a new UserProfile with email domains set.
func (p UserProfile) WithEmailDomains(domains []string) UserProfile {
	newDomains := make([]string, len(domains))
	copy(newDomains, domains)

	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   p.existingTools,
		operatingSystem: p.operatingSystem,
		workContext:     p.workContext,
		deviceType:      p.deviceType,
		emailDomains:    newDomains,
		teamSize:        p.teamSize,
	}
}

// WithTeamSize returns a new UserProfile with team size set.
func (p UserProfile) WithTeamSize(size int) UserProfile {
	return UserProfile{
		experience:      p.experience,
		preferences:     p.preferences,
		existingTools:   p.existingTools,
		operatingSystem: p.operatingSystem,
		workContext:     p.workContext,
		deviceType:      p.deviceType,
		emailDomains:    p.emailDomains,
		teamSize:        size,
	}
}

// IsZero returns true if this is a zero-value UserProfile.
func (p UserProfile) IsZero() bool {
	return p.experience == ""
}

// SuggestContext contains all context needed for AI to generate suggestions.
type SuggestContext struct {
	category           string
	userProfile        UserProfile
	constraints        []string
	additionalContext  string
	maxRecommendations int
}

// NewSuggestContext creates a new SuggestContext.
func NewSuggestContext(category string, profile UserProfile) (SuggestContext, error) {
	if category == "" {
		return SuggestContext{}, ErrEmptyCategory
	}

	if profile.IsZero() {
		return SuggestContext{}, ErrInvalidProfile
	}

	return SuggestContext{
		category:           category,
		userProfile:        profile,
		constraints:        []string{},
		maxRecommendations: 3, // default
	}, nil
}

// Category returns the category for recommendations (e.g., "nvim", "shell").
func (c SuggestContext) Category() string {
	return c.category
}

// UserProfile returns the user profile.
func (c SuggestContext) UserProfile() UserProfile {
	return c.userProfile
}

// Constraints returns any constraints on recommendations.
func (c SuggestContext) Constraints() []string {
	result := make([]string, len(c.constraints))
	copy(result, c.constraints)
	return result
}

// AdditionalContext returns any additional context provided by the user.
func (c SuggestContext) AdditionalContext() string {
	return c.additionalContext
}

// MaxRecommendations returns the maximum number of recommendations to return.
func (c SuggestContext) MaxRecommendations() int {
	return c.maxRecommendations
}

// WithConstraints returns a new SuggestContext with constraints set.
func (c SuggestContext) WithConstraints(constraints []string) SuggestContext {
	newConstraints := make([]string, len(constraints))
	copy(newConstraints, constraints)

	return SuggestContext{
		category:           c.category,
		userProfile:        c.userProfile,
		constraints:        newConstraints,
		additionalContext:  c.additionalContext,
		maxRecommendations: c.maxRecommendations,
	}
}

// WithAdditionalContext returns a new SuggestContext with additional context set.
func (c SuggestContext) WithAdditionalContext(ctx string) SuggestContext {
	return SuggestContext{
		category:           c.category,
		userProfile:        c.userProfile,
		constraints:        c.constraints,
		additionalContext:  ctx,
		maxRecommendations: c.maxRecommendations,
	}
}

// WithMaxRecommendations returns a new SuggestContext with max recommendations set.
func (c SuggestContext) WithMaxRecommendations(maxRec int) SuggestContext {
	return SuggestContext{
		category:           c.category,
		userProfile:        c.userProfile,
		constraints:        c.constraints,
		additionalContext:  c.additionalContext,
		maxRecommendations: maxRec,
	}
}

// IsZero returns true if this is a zero-value SuggestContext.
func (c SuggestContext) IsZero() bool {
	return c.category == ""
}

// ExplainRequest contains parameters for explaining a preset.
type ExplainRequest struct {
	presetID       string
	userExperience ExperienceLevel
	questions      []string
}

// NewExplainRequest creates a new ExplainRequest.
func NewExplainRequest(presetID string) (ExplainRequest, error) {
	if presetID == "" {
		return ExplainRequest{}, ErrEmptyPresetID
	}

	return ExplainRequest{
		presetID:  presetID,
		questions: []string{},
	}, nil
}

// PresetID returns the preset to explain.
func (r ExplainRequest) PresetID() string {
	return r.presetID
}

// UserExperience returns the user's experience level for tailored explanations.
func (r ExplainRequest) UserExperience() ExperienceLevel {
	return r.userExperience
}

// Questions returns specific questions to answer.
func (r ExplainRequest) Questions() []string {
	result := make([]string, len(r.questions))
	copy(result, r.questions)
	return result
}

// WithUserExperience returns a new ExplainRequest with experience set.
func (r ExplainRequest) WithUserExperience(exp ExperienceLevel) ExplainRequest {
	return ExplainRequest{
		presetID:       r.presetID,
		userExperience: exp,
		questions:      r.questions,
	}
}

// WithQuestions returns a new ExplainRequest with questions set.
func (r ExplainRequest) WithQuestions(questions []string) ExplainRequest {
	newQuestions := make([]string, len(questions))
	copy(newQuestions, questions)

	return ExplainRequest{
		presetID:       r.presetID,
		userExperience: r.userExperience,
		questions:      newQuestions,
	}
}
