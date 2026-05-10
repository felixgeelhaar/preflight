package config

import (
	"sort"
	"strings"
	"testing"
)

// Property tests for the layer-merge algebra. Spec (CLAUDE.md):
//
//   * scalars last-wins
//   * maps deep-merge
//   * lists set-union
//
// These tests exercise the algebraic invariants directly so a refactor cannot
// silently break the contract that hand-written examples might miss.

func sortedStrings(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

func equalStringSets(a, b []string) bool {
	a = sortedStrings(a)
	b = sortedStrings(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func packagesEqual(a, b PackageSet) bool {
	return equalStringSets(a.Brew.Taps, b.Brew.Taps) &&
		equalStringSets(a.Brew.Formulae, b.Brew.Formulae) &&
		equalStringSets(a.Brew.Casks, b.Brew.Casks) &&
		equalStringSets(a.Apt.PPAs, b.Apt.PPAs) &&
		equalStringSets(a.Apt.Packages, b.Apt.Packages)
}

// TestMergePackages_Identity: Merge(a, Empty) ≡ a
func TestMergePackages_Identity(t *testing.T) {
	t.Parallel()
	r := NewLayerResolver()
	a := PackageSet{
		Brew: BrewPackages{Taps: []string{"x/y"}, Formulae: []string{"ripgrep", "fzf"}},
		Apt:  AptPackages{Packages: []string{"curl"}},
	}
	got := r.mergePackages(a, PackageSet{})
	if !packagesEqual(got, a) {
		t.Errorf("Merge(a, Empty) ≠ a:\n got = %#v\n want= %#v", got, a)
	}
	got = r.mergePackages(PackageSet{}, a)
	if !packagesEqual(got, a) {
		t.Errorf("Merge(Empty, a) ≠ a:\n got = %#v\n want= %#v", got, a)
	}
}

// TestMergePackages_Idempotent: Merge(a, a) ≡ a (set-union with itself).
func TestMergePackages_Idempotent(t *testing.T) {
	t.Parallel()
	r := NewLayerResolver()
	a := PackageSet{
		Brew: BrewPackages{
			Taps:     []string{"a/b", "c/d"},
			Formulae: []string{"ripgrep", "fzf", "bat"},
			Casks:    []string{"alacritty"},
		},
		Apt: AptPackages{Packages: []string{"curl", "git"}, PPAs: []string{"ppa:x"}},
	}
	got := r.mergePackages(a, a)
	if !packagesEqual(got, a) {
		t.Errorf("Merge(a, a) ≠ a (set-union not idempotent):\n got = %#v\n want= %#v", got, a)
	}
}

// TestMergePackages_Commutative_AsSets: As sets, Merge(a, b) ≡ Merge(b, a).
// Order in the result slice may differ, but the multiset of values must match.
func TestMergePackages_Commutative_AsSets(t *testing.T) {
	t.Parallel()
	r := NewLayerResolver()
	a := PackageSet{Brew: BrewPackages{Formulae: []string{"ripgrep", "fzf"}}}
	b := PackageSet{Brew: BrewPackages{Formulae: []string{"fzf", "bat"}}}

	ab := r.mergePackages(a, b)
	ba := r.mergePackages(b, a)
	if !packagesEqual(ab, ba) {
		t.Errorf("Merge not commutative as sets:\n a∪b = %v\n b∪a = %v", ab.Brew.Formulae, ba.Brew.Formulae)
	}
}

// TestMergePackages_Associative: Merge(Merge(a,b),c) ≡ Merge(a,Merge(b,c)) as sets.
func TestMergePackages_Associative(t *testing.T) {
	t.Parallel()
	r := NewLayerResolver()
	a := PackageSet{Brew: BrewPackages{Formulae: []string{"a", "b"}}}
	b := PackageSet{Brew: BrewPackages{Formulae: []string{"b", "c"}}}
	c := PackageSet{Brew: BrewPackages{Formulae: []string{"c", "d"}}}

	left := r.mergePackages(r.mergePackages(a, b), c)
	right := r.mergePackages(a, r.mergePackages(b, c))
	if !packagesEqual(left, right) {
		t.Errorf("Merge not associative:\n left  = %v\n right = %v", left.Brew.Formulae, right.Brew.Formulae)
	}
}

// FuzzMergePackages_SetUnion explores arbitrary input pairs and asserts the
// invariants: identity, idempotency, commutativity-as-sets.
func FuzzMergePackages_SetUnion(f *testing.F) {
	f.Add("ripgrep,fzf", "fzf,bat")
	f.Add("", "ripgrep")
	f.Add("a,b,c", "")
	f.Add(",,a,", "b,,c,")

	f.Fuzz(func(t *testing.T, aCSV, bCSV string) {
		r := NewLayerResolver()
		a := PackageSet{Brew: BrewPackages{Formulae: splitCSV(aCSV)}}
		b := PackageSet{Brew: BrewPackages{Formulae: splitCSV(bCSV)}}

		ab := r.mergePackages(a, b)
		ba := r.mergePackages(b, a)
		if !equalStringSets(ab.Brew.Formulae, ba.Brew.Formulae) {
			t.Fatalf("commutativity as sets violated:\n a∪b=%v\n b∪a=%v", ab.Brew.Formulae, ba.Brew.Formulae)
		}

		// Idempotency: a∪a == a (deduplicated)
		aa := r.mergePackages(a, a)
		if !equalStringSets(aa.Brew.Formulae, sortedStrings(uniqueStringsCopy(a.Brew.Formulae))) {
			t.Fatalf("idempotency violated: a∪a=%v want=%v", aa.Brew.Formulae, a.Brew.Formulae)
		}

		// Identity: a∪{} == a (deduplicated)
		aE := r.mergePackages(a, PackageSet{})
		if !equalStringSets(aE.Brew.Formulae, sortedStrings(uniqueStringsCopy(a.Brew.Formulae))) {
			t.Fatalf("identity violated: a∪{}=%v want=%v", aE.Brew.Formulae, a.Brew.Formulae)
		}
	})
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func uniqueStringsCopy(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	out := make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
