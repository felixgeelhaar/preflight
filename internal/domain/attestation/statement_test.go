package attestation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatement_Valid(t *testing.T) {
	t.Parallel()

	subjects := []Subject{
		{
			Name:   "pkg:brew/git@2.40.0",
			Digest: map[string]string{"sha256": "abc123def456"},
		},
	}

	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, []byte(`{"key":"value"}`))
	require.NoError(t, err)
	assert.Equal(t, "https://in-toto.io/Statement/v1", stmt.Type())
	assert.Equal(t, PredicateTypeSLSAProvenanceV1, stmt.PredicateType())
	assert.Equal(t, subjects, stmt.Subject())
	assert.JSONEq(t, `{"key":"value"}`, string(stmt.Predicate()))
	assert.False(t, stmt.IsZero())
}

func TestNewStatement_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		predicateType string
		subjects      []Subject
		predicate     []byte
		wantErr       string
	}{
		{
			name:          "empty predicate type",
			predicateType: "",
			subjects: []Subject{
				{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{"sha256": "abc123"}},
			},
			predicate: []byte(`{}`),
			wantErr:   "predicate type is required",
		},
		{
			name:          "no subjects",
			predicateType: PredicateTypeSLSAProvenanceV1,
			subjects:      nil,
			predicate:     []byte(`{}`),
			wantErr:       "at least one subject is required",
		},
		{
			name:          "empty subjects slice",
			predicateType: PredicateTypeSLSAProvenanceV1,
			subjects:      []Subject{},
			predicate:     []byte(`{}`),
			wantErr:       "at least one subject is required",
		},
		{
			name:          "subject without name",
			predicateType: PredicateTypeSLSAProvenanceV1,
			subjects: []Subject{
				{Name: "", Digest: map[string]string{"sha256": "abc123"}},
			},
			predicate: []byte(`{}`),
			wantErr:   "subject[0]: name is required",
		},
		{
			name:          "subject without digest",
			predicateType: PredicateTypeSLSAProvenanceV1,
			subjects: []Subject{
				{Name: "pkg:brew/git@2.40.0", Digest: nil},
			},
			predicate: []byte(`{}`),
			wantErr:   "subject[0]: at least one digest is required",
		},
		{
			name:          "subject with empty digest map",
			predicateType: PredicateTypeSLSAProvenanceV1,
			subjects: []Subject{
				{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{}},
			},
			predicate: []byte(`{}`),
			wantErr:   "subject[0]: at least one digest is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmt, err := NewStatement(tt.predicateType, tt.subjects, tt.predicate)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidStatement)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.True(t, stmt.IsZero())
		})
	}
}

func TestStatement_IsZero(t *testing.T) {
	t.Parallel()

	var zero Statement
	assert.True(t, zero.IsZero())
}

func TestStatement_Validate(t *testing.T) {
	t.Parallel()

	subjects := []Subject{
		{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{"sha256": "abc123"}},
	}

	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, []byte(`{}`))
	require.NoError(t, err)
	assert.NoError(t, stmt.Validate())

	var zero Statement
	assert.Error(t, zero.Validate())
}

func TestStatement_SubjectMatchesDigest(t *testing.T) {
	t.Parallel()

	subjects := []Subject{
		{
			Name:   "pkg:brew/git@2.40.0",
			Digest: map[string]string{"sha256": "abc123def456"},
		},
		{
			Name:   "pkg:brew/curl@8.0.0",
			Digest: map[string]string{"sha256": "xyz789", "sha512": "longdigest"},
		},
	}

	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, []byte(`{}`))
	require.NoError(t, err)

	tests := []struct {
		name      string
		subjName  string
		algorithm string
		digest    string
		want      bool
	}{
		{
			name:      "exact match",
			subjName:  "pkg:brew/git@2.40.0",
			algorithm: "sha256",
			digest:    "abc123def456",
			want:      true,
		},
		{
			name:      "second subject sha512 match",
			subjName:  "pkg:brew/curl@8.0.0",
			algorithm: "sha512",
			digest:    "longdigest",
			want:      true,
		},
		{
			name:      "wrong name",
			subjName:  "pkg:brew/wget@1.0.0",
			algorithm: "sha256",
			digest:    "abc123def456",
			want:      false,
		},
		{
			name:      "wrong algorithm",
			subjName:  "pkg:brew/git@2.40.0",
			algorithm: "sha512",
			digest:    "abc123def456",
			want:      false,
		},
		{
			name:      "wrong digest",
			subjName:  "pkg:brew/git@2.40.0",
			algorithm: "sha256",
			digest:    "wrong",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stmt.SubjectMatchesDigest(tt.subjName, tt.algorithm, tt.digest))
		})
	}
}

func TestStatement_Immutability(t *testing.T) {
	t.Parallel()

	subjects := []Subject{
		{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{"sha256": "abc123"}},
	}
	predicate := []byte(`{"key":"value"}`)

	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, predicate)
	require.NoError(t, err)

	// Mutating the original slices should not affect the statement.
	subjects[0].Name = "mutated"
	predicate[0] = 'X'

	assert.Equal(t, "pkg:brew/git@2.40.0", stmt.Subject()[0].Name)
	assert.Equal(t, byte('{'), stmt.Predicate()[0])
}

func TestNewStatement_MultipleSubjects(t *testing.T) {
	t.Parallel()

	subjects := []Subject{
		{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{"sha256": "aaa"}},
		{Name: "pkg:brew/curl@8.0.0", Digest: map[string]string{"sha256": "bbb"}},
	}

	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, []byte(`{}`))
	require.NoError(t, err)
	assert.Len(t, stmt.Subject(), 2)
}
