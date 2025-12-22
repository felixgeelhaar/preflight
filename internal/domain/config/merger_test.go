package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerger_Merge_SingleLayer_ReturnsUnchanged(t *testing.T) {
	t.Parallel()

	layer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
`))
	require.NoError(t, err)
	layer.SetProvenance("layers/base.yaml")

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*layer})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git"}, merged.Packages.Brew.Formulae)
}

func TestMerger_Merge_TwoLayers_ListUnion(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
      - ripgrep
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    formulae:
      - docker
      - kubectl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git", "ripgrep", "docker", "kubectl"}, merged.Packages.Brew.Formulae)
}

func TestMerger_Merge_Files_CombinesDeclarations(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
files:
  - path: ~/.gitconfig
    mode: generated
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
files:
  - path: ~/.ssh/config
    mode: template
    template: ssh/config.tmpl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	require.Len(t, merged.Files, 2)
}

func TestMerger_Merge_Files_LastWinsForSamePath(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
files:
  - path: ~/.gitconfig
    mode: generated
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
files:
  - path: ~/.gitconfig
    mode: template
    template: git/config.tmpl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	require.Len(t, merged.Files, 1)
	assert.Equal(t, config.FileModeTemplate, merged.Files[0].Mode)
	assert.Equal(t, "git/config.tmpl", merged.Files[0].Template)
}

func TestMerger_Merge_Provenance_TracksSourceLayer(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
`))
	require.NoError(t, err)
	baseLayer.SetProvenance("layers/base.yaml")

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    formulae:
      - docker
`))
	require.NoError(t, err)
	workLayer.SetProvenance("layers/identity.work.yaml")

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)

	// Check provenance is tracked
	gitProv := merged.GetProvenance("packages.brew.formulae", "git")
	assert.Equal(t, "layers/base.yaml", gitProv)

	dockerProv := merged.GetProvenance("packages.brew.formulae", "docker")
	assert.Equal(t, "layers/identity.work.yaml", dockerProv)
}

func TestMerger_Merge_EmptyLayers_ReturnsEmptyConfig(t *testing.T) {
	t.Parallel()

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{})

	require.NoError(t, err)
	assert.Empty(t, merged.Packages.Brew.Formulae)
	assert.Empty(t, merged.Files)
}

func TestMerger_Merge_Casks_Union(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    casks:
      - visual-studio-code
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    casks:
      - docker
      - visual-studio-code
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	// Should be union (deduplicated)
	assert.Len(t, merged.Packages.Brew.Casks, 2)
	assert.ElementsMatch(t, []string{"visual-studio-code", "docker"}, merged.Packages.Brew.Casks)
}

func TestMerger_Merge_Git_UserConfig_LastWins(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
git:
  user:
    name: Base User
    email: base@example.com
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
git:
  user:
    email: work@company.com
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	// Name from base, email overridden by work
	assert.Equal(t, "Base User", merged.Git.User.Name)
	assert.Equal(t, "work@company.com", merged.Git.User.Email)
}

func TestMerger_Merge_Git_Aliases_DeepMerge(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
git:
  alias:
    co: checkout
    st: status
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
git:
  alias:
    br: branch
    st: status -sb
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	// Deep merge: co from base, br from work, st overwritten by work
	assert.Equal(t, "checkout", merged.Git.Aliases["co"])
	assert.Equal(t, "branch", merged.Git.Aliases["br"])
	assert.Equal(t, "status -sb", merged.Git.Aliases["st"])
}

func TestMerger_Merge_Git_Includes_SetUnion(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
git:
  includes:
    - path: ~/.gitconfig.local
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
git:
  includes:
    - path: ~/.gitconfig.work
      ifconfig: "gitdir:~/work/"
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.Len(t, merged.Git.Includes, 2)
}

func TestMerger_Merge_Git_GPGSigning(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
git:
  user:
    name: John Doe
    signingkey: ABCD1234
  commit:
    gpgsign: true
  gpg:
    format: openpgp
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Equal(t, "ABCD1234", merged.Git.User.SigningKey)
	assert.True(t, merged.Git.Commit.GPGSign)
	assert.Equal(t, "openpgp", merged.Git.GPG.Format)
}

func TestMerger_Merge_Git_CoreConfig(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
git:
  core:
    editor: nvim
    autocrlf: input
    excludesfile: ~/.gitignore_global
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Equal(t, "nvim", merged.Git.Core.Editor)
	assert.Equal(t, "input", merged.Git.Core.AutoCRLF)
	assert.Equal(t, "~/.gitignore_global", merged.Git.Core.ExcludesFile)
}
