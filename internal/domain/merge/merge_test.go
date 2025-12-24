package merge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreeWayMerge_FastPaths(t *testing.T) {
	t.Parallel()

	t.Run("base equals ours - use theirs", func(t *testing.T) {
		t.Parallel()
		base := "line1\nline2\n"
		ours := "line1\nline2\n"
		theirs := "line1\nmodified\nline2\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		assert.True(t, result.CleanMerge)
		assert.False(t, result.HasConflicts)
		assert.Equal(t, theirs, result.Content)
	})

	t.Run("base equals theirs - use ours", func(t *testing.T) {
		t.Parallel()
		base := "line1\nline2\n"
		ours := "line1\nmodified\nline2\n"
		theirs := "line1\nline2\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		assert.True(t, result.CleanMerge)
		assert.False(t, result.HasConflicts)
		assert.Equal(t, ours, result.Content)
	})

	t.Run("ours equals theirs - use either", func(t *testing.T) {
		t.Parallel()
		base := "line1\nline2\n"
		ours := "line1\nmodified\nline2\n"
		theirs := "line1\nmodified\nline2\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		assert.True(t, result.CleanMerge)
		assert.False(t, result.HasConflicts)
		assert.Equal(t, ours, result.Content)
	})
}

func TestThreeWayMerge_Conflicts(t *testing.T) {
	t.Parallel()

	t.Run("both sides modified - generates conflict", func(t *testing.T) {
		t.Parallel()
		base := "line1\noriginal\nline3\n"
		ours := "line1\nconfig change\nline3\n"
		theirs := "line1\nuser change\nline3\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		assert.False(t, result.CleanMerge)
		assert.True(t, result.HasConflicts)
		assert.Len(t, result.Conflicts, 1)
		assert.Contains(t, result.Content, ConflictMarkers.Start)
		assert.Contains(t, result.Content, "config change")
		assert.Contains(t, result.Content, "user change")
		assert.Contains(t, result.Content, ConflictMarkers.End)
	})

	t.Run("diff3 style includes base", func(t *testing.T) {
		t.Parallel()
		base := "original\n"
		ours := "config change\n"
		theirs := "user change\n"

		result := ThreeWayMerge(base, ours, theirs, StyleDiff3)

		assert.True(t, result.HasConflicts)
		assert.Contains(t, result.Content, ConflictMarkers.Base)
		assert.Contains(t, result.Content, "original")
	})
}

func TestThreeWayMerge_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("all empty", func(t *testing.T) {
		t.Parallel()
		result := ThreeWayMerge("", "", "", StyleGit)

		assert.True(t, result.CleanMerge)
		assert.Equal(t, "", result.Content)
	})

	t.Run("empty base with different content", func(t *testing.T) {
		t.Parallel()
		base := ""
		ours := "line1\n"
		theirs := "line2\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		// Both added content to empty file - conflict
		assert.True(t, result.HasConflicts)
	})
}

func TestHasConflictMarkers(t *testing.T) {
	t.Parallel()

	t.Run("has markers", func(t *testing.T) {
		t.Parallel()
		content := "line1\n<<<<<<< ours\nconfig\n=======\nfile\n>>>>>>> theirs\nline2\n"
		assert.True(t, HasConflictMarkers(content))
	})

	t.Run("no markers", func(t *testing.T) {
		t.Parallel()
		content := "line1\nline2\nline3\n"
		assert.False(t, HasConflictMarkers(content))
	})

	t.Run("partial markers", func(t *testing.T) {
		t.Parallel()
		content := "line1\n<<<<<<< start only\nline2\n"
		assert.False(t, HasConflictMarkers(content))
	})
}

func TestParseConflictRegions(t *testing.T) {
	t.Parallel()

	t.Run("single conflict", func(t *testing.T) {
		t.Parallel()
		content := `line1
<<<<<<< ours
config line
=======
file line
>>>>>>> theirs
line2
`
		conflicts := ParseConflictRegions(content)

		require.Len(t, conflicts, 1)
		assert.Equal(t, []string{"config line"}, conflicts[0].Ours)
		assert.Equal(t, []string{"file line"}, conflicts[0].Theirs)
	})

	t.Run("diff3 style with base", func(t *testing.T) {
		t.Parallel()
		content := `line1
<<<<<<< ours
config line
||||||| base
original line
=======
file line
>>>>>>> theirs
line2
`
		conflicts := ParseConflictRegions(content)

		require.Len(t, conflicts, 1)
		assert.Equal(t, []string{"config line"}, conflicts[0].Ours)
		assert.Equal(t, []string{"original line"}, conflicts[0].Base)
		assert.Equal(t, []string{"file line"}, conflicts[0].Theirs)
	})

	t.Run("multiple conflicts", func(t *testing.T) {
		t.Parallel()
		content := `line1
<<<<<<< ours
a
=======
b
>>>>>>> theirs
middle
<<<<<<< ours
c
=======
d
>>>>>>> theirs
end
`
		conflicts := ParseConflictRegions(content)

		assert.Len(t, conflicts, 2)
	})

	t.Run("no conflicts", func(t *testing.T) {
		t.Parallel()
		content := "clean content\nno markers\n"
		conflicts := ParseConflictRegions(content)

		assert.Len(t, conflicts, 0)
	})
}

func TestResolveAllConflicts(t *testing.T) {
	t.Parallel()

	content := `line1
<<<<<<< ours
config
=======
file
>>>>>>> theirs
line2
`

	t.Run("resolve ours", func(t *testing.T) {
		t.Parallel()
		result := ResolveAllConflicts(content, ResolveOurs)

		assert.Contains(t, result, "config")
		assert.NotContains(t, result, "file")
		assert.NotContains(t, result, ConflictMarkers.Start)
	})

	t.Run("resolve theirs", func(t *testing.T) {
		t.Parallel()
		result := ResolveAllConflicts(content, ResolveTheirs)

		assert.NotContains(t, result, "config")
		assert.Contains(t, result, "file")
		assert.NotContains(t, result, ConflictMarkers.Start)
	})

	t.Run("resolve base", func(t *testing.T) {
		t.Parallel()
		contentWithBase := `line1
<<<<<<< ours
config
||||||| base
original
=======
file
>>>>>>> theirs
line2
`
		result := ResolveAllConflicts(contentWithBase, ResolveBase)

		assert.Contains(t, result, "original")
		assert.NotContains(t, result, "config")
		assert.NotContains(t, result, "file")
	})

	t.Run("multiple conflicts", func(t *testing.T) {
		t.Parallel()
		multiConflict := `a
<<<<<<< ours
x
=======
y
>>>>>>> theirs
b
<<<<<<< ours
m
=======
n
>>>>>>> theirs
c
`
		result := ResolveAllConflicts(multiConflict, ResolveOurs)

		assert.Contains(t, result, "x")
		assert.Contains(t, result, "m")
		assert.NotContains(t, result, "y")
		assert.NotContains(t, result, "n")
	})
}

func TestNewCleanResult(t *testing.T) {
	t.Parallel()

	result := NewCleanResult("content")

	assert.Equal(t, "content", result.Content)
	assert.True(t, result.CleanMerge)
	assert.False(t, result.HasConflicts)
	assert.Nil(t, result.Conflicts)
}

func TestNewConflictResult(t *testing.T) {
	t.Parallel()

	conflicts := []Conflict{{Start: 1, End: 5}}
	result := NewConflictResult("content", conflicts)

	assert.Equal(t, "content", result.Content)
	assert.False(t, result.CleanMerge)
	assert.True(t, result.HasConflicts)
	assert.Len(t, result.Conflicts, 1)
}

func TestSummarize(t *testing.T) {
	t.Parallel()

	t.Run("with conflicts", func(t *testing.T) {
		t.Parallel()
		conflicts := []Conflict{
			{Ours: []string{"a", "b"}, Theirs: []string{"c"}, Base: []string{"d"}},
			{Ours: []string{"x"}, Theirs: []string{"y", "z"}, Base: nil},
		}

		summary := Summarize("/path/to/file", conflicts)

		assert.Equal(t, "/path/to/file", summary.Path)
		assert.Equal(t, 2, summary.ConflictCount)
		assert.Equal(t, 3, summary.OursLineCount)
		assert.Equal(t, 3, summary.TheirsLineCount)
		assert.Equal(t, 1, summary.BaseLineCount)
	})

	t.Run("no conflicts", func(t *testing.T) {
		t.Parallel()
		summary := Summarize("/path", nil)

		assert.Equal(t, 0, summary.ConflictCount)
	})
}

func TestConflictSummary_Description(t *testing.T) {
	t.Parallel()

	t.Run("with conflicts", func(t *testing.T) {
		t.Parallel()
		summary := ConflictSummary{
			Path:            "/file",
			ConflictCount:   2,
			OursLineCount:   5,
			TheirsLineCount: 3,
		}

		desc := summary.Description()

		assert.Contains(t, desc, "/file")
		assert.Contains(t, desc, "2 conflict")
		assert.Contains(t, desc, "5 lines from config")
		assert.Contains(t, desc, "3 lines from file")
	})

	t.Run("no conflicts", func(t *testing.T) {
		t.Parallel()
		summary := ConflictSummary{Path: "/file", ConflictCount: 0}

		desc := summary.Description()

		assert.Contains(t, desc, "no conflicts")
	})
}

func TestSplitLines(t *testing.T) {
	t.Parallel()

	t.Run("normal content", func(t *testing.T) {
		t.Parallel()
		lines := splitLines("a\nb\nc\n")
		assert.Equal(t, []string{"a", "b", "c"}, lines)
	})

	t.Run("no trailing newline", func(t *testing.T) {
		t.Parallel()
		lines := splitLines("a\nb\nc")
		assert.Equal(t, []string{"a", "b", "c"}, lines)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		lines := splitLines("")
		assert.Nil(t, lines)
	})

	t.Run("single line", func(t *testing.T) {
		t.Parallel()
		lines := splitLines("single\n")
		assert.Equal(t, []string{"single"}, lines)
	})
}

func TestSliceEqual(t *testing.T) {
	t.Parallel()

	t.Run("equal slices", func(t *testing.T) {
		t.Parallel()
		assert.True(t, sliceEqual([]string{"a", "b"}, []string{"a", "b"}))
	})

	t.Run("different length", func(t *testing.T) {
		t.Parallel()
		assert.False(t, sliceEqual([]string{"a"}, []string{"a", "b"}))
	})

	t.Run("different content", func(t *testing.T) {
		t.Parallel()
		assert.False(t, sliceEqual([]string{"a", "b"}, []string{"a", "c"}))
	})

	t.Run("empty slices", func(t *testing.T) {
		t.Parallel()
		assert.True(t, sliceEqual([]string{}, []string{}))
	})

	t.Run("nil slices", func(t *testing.T) {
		t.Parallel()
		assert.True(t, sliceEqual(nil, nil))
	})
}

func TestConflictStyle(t *testing.T) {
	t.Parallel()

	t.Run("git style", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, StyleGit, ConflictStyle("git"))
	})

	t.Run("diff3 style", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, StyleDiff3, ConflictStyle("diff3"))
	})
}

func TestDetectChangeType(t *testing.T) {
	t.Parallel()

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		ct := DetectChangeType("same", "same", "same")
		assert.Equal(t, ChangeNone, ct)
	})

	t.Run("ours changed", func(t *testing.T) {
		t.Parallel()
		ct := DetectChangeType("base", "ours", "base")
		assert.Equal(t, ChangeOurs, ct)
	})

	t.Run("theirs changed", func(t *testing.T) {
		t.Parallel()
		ct := DetectChangeType("base", "base", "theirs")
		assert.Equal(t, ChangeTheirs, ct)
	})

	t.Run("both changed same way", func(t *testing.T) {
		t.Parallel()
		ct := DetectChangeType("base", "same", "same")
		assert.Equal(t, ChangeSame, ct)
	})

	t.Run("both changed differently", func(t *testing.T) {
		t.Parallel()
		ct := DetectChangeType("base", "ours", "theirs")
		assert.Equal(t, ChangeBoth, ct)
	})
}

func TestNeedsManualResolution(t *testing.T) {
	t.Parallel()

	t.Run("ChangeBoth needs resolution", func(t *testing.T) {
		t.Parallel()
		assert.True(t, NeedsManualResolution(ChangeBoth))
	})

	t.Run("other types don't need resolution", func(t *testing.T) {
		t.Parallel()
		assert.False(t, NeedsManualResolution(ChangeNone))
		assert.False(t, NeedsManualResolution(ChangeOurs))
		assert.False(t, NeedsManualResolution(ChangeTheirs))
		assert.False(t, NeedsManualResolution(ChangeSame))
	})
}

func TestChangeType_Description(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ct       ChangeType
		contains string
	}{
		{ChangeNone, "no changes"},
		{ChangeOurs, "config changed"},
		{ChangeTheirs, "file changed externally"},
		{ChangeBoth, "conflict"},
		{ChangeSame, "identical changes"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tt.ct.Description(), tt.contains)
		})
	}
}

func TestFormatWholeFileConflict(t *testing.T) {
	t.Parallel()

	conflict := Conflict{
		Base:   []string{"original"},
		Ours:   []string{"config"},
		Theirs: []string{"file"},
	}

	t.Run("git style", func(t *testing.T) {
		t.Parallel()
		result := formatWholeFileConflict(conflict, StyleGit)

		assert.Contains(t, result, "<<<<<<< ours")
		assert.Contains(t, result, "config")
		assert.Contains(t, result, "=======")
		assert.Contains(t, result, "file")
		assert.Contains(t, result, ">>>>>>> theirs")
		assert.NotContains(t, result, "|||||||")
	})

	t.Run("diff3 style", func(t *testing.T) {
		t.Parallel()
		result := formatWholeFileConflict(conflict, StyleDiff3)

		assert.Contains(t, result, "<<<<<<< ours")
		assert.Contains(t, result, "||||||| base")
		assert.Contains(t, result, "original")
		assert.Contains(t, result, "=======")
		assert.Contains(t, result, ">>>>>>> theirs")
	})
}

func TestRealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("gitconfig - only config changes email", func(t *testing.T) {
		t.Parallel()
		base := `[user]
	name = John Doe
	email = john@example.com
`
		ours := `[user]
	name = John Doe
	email = john@work.com
`
		theirs := base // User didn't change

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		// Only config changed - use ours
		assert.True(t, result.CleanMerge)
		assert.Contains(t, result.Content, "john@work.com")
	})

	t.Run("gitconfig - only user adds alias", func(t *testing.T) {
		t.Parallel()
		base := `[user]
	name = John Doe
`
		ours := base // Config didn't change
		theirs := `[user]
	name = John Doe
[alias]
	st = status
`

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		// Only user changed - use theirs
		assert.True(t, result.CleanMerge)
		assert.Contains(t, result.Content, "st = status")
	})

	t.Run("shell config - both change EDITOR", func(t *testing.T) {
		t.Parallel()
		base := `export PATH="/usr/bin:$PATH"
export EDITOR="vim"
`
		ours := `export PATH="/usr/bin:$PATH"
export EDITOR="nvim"
`
		theirs := `export PATH="/usr/bin:$PATH"
export EDITOR="code"
`
		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		// Both changed - conflict
		assert.True(t, result.HasConflicts)
		assert.Contains(t, result.Content, "nvim")
		assert.Contains(t, result.Content, "code")
		assert.Contains(t, result.Content, ConflictMarkers.Start)
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("create conflict then resolve ours", func(t *testing.T) {
		t.Parallel()
		base := "original\n"
		ours := "config\n"
		theirs := "user\n"

		// Create conflict
		result := ThreeWayMerge(base, ours, theirs, StyleGit)
		require.True(t, result.HasConflicts)

		// Resolve using ours
		resolved := ResolveAllConflicts(result.Content, ResolveOurs)

		assert.Equal(t, ours, resolved)
	})

	t.Run("create conflict then resolve theirs", func(t *testing.T) {
		t.Parallel()
		base := "original\n"
		ours := "config\n"
		theirs := "user\n"

		// Create conflict
		result := ThreeWayMerge(base, ours, theirs, StyleGit)
		require.True(t, result.HasConflicts)

		// Resolve using theirs
		resolved := ResolveAllConflicts(result.Content, ResolveTheirs)

		assert.Equal(t, theirs, resolved)
	})

	t.Run("create diff3 conflict then resolve base", func(t *testing.T) {
		t.Parallel()
		base := "original\n"
		ours := "config\n"
		theirs := "user\n"

		// Create conflict with diff3 style
		result := ThreeWayMerge(base, ours, theirs, StyleDiff3)
		require.True(t, result.HasConflicts)

		// Resolve using base
		resolved := ResolveAllConflicts(result.Content, ResolveBase)

		assert.Equal(t, base, resolved)
	})
}

func TestConflictMarkerLabels(t *testing.T) {
	t.Parallel()

	t.Run("markers have descriptive labels", func(t *testing.T) {
		t.Parallel()
		base := "original\n"
		ours := "config\n"
		theirs := "user\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		// Check markers have helpful labels
		assert.Contains(t, result.Content, "ours (config)")
		assert.Contains(t, result.Content, "theirs (file)")
	})
}

func TestMultilineContent(t *testing.T) {
	t.Parallel()

	t.Run("multiline conflict", func(t *testing.T) {
		t.Parallel()
		base := "line1\nline2\nline3\n"
		ours := "line1\nconfig1\nconfig2\nline3\n"
		theirs := "line1\nuser1\nuser2\nuser3\nline3\n"

		result := ThreeWayMerge(base, ours, theirs, StyleGit)

		assert.True(t, result.HasConflicts)
		conflicts := ParseConflictRegions(result.Content)
		require.Len(t, conflicts, 1)

		// Both ours and theirs should have multiline content
		assert.Contains(t, strings.Join(conflicts[0].Ours, "\n"), "config1")
		assert.Contains(t, strings.Join(conflicts[0].Theirs, "\n"), "user1")
	})
}
