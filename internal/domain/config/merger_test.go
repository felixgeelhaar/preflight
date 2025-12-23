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

func TestMerger_Merge_SSH_Hosts(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
ssh:
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
ssh:
  hosts:
    - host: work
      hostname: git.company.com
      user: developer
      identityfile: ~/.ssh/id_work
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.Len(t, merged.SSH.Hosts, 2)
}

func TestMerger_Merge_SSH_Defaults(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
    serveraliveinterval: 60
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.True(t, merged.SSH.Defaults.AddKeysToAgent)
	assert.True(t, merged.SSH.Defaults.IdentitiesOnly)
	assert.Equal(t, 60, merged.SSH.Defaults.ServerAliveInterval)
}

func TestMerger_Merge_SSH_Include(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
ssh:
  include: ~/.ssh/config.d/*
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Equal(t, "~/.ssh/config.d/*", merged.SSH.Include)
}

func TestMerger_Merge_Runtime_Tools(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
runtime:
  tools:
    - name: node
      version: "20.10.0"
    - name: golang
      version: "1.21.5"
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Len(t, merged.Runtime.Tools, 2)
}

func TestMerger_Merge_Runtime_Tools_LastWins(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
runtime:
  tools:
    - name: node
      version: "18.0.0"
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
runtime:
  tools:
    - name: node
      version: "20.10.0"
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.Len(t, merged.Runtime.Tools, 1)
	assert.Equal(t, "20.10.0", merged.Runtime.Tools[0].Version)
}

func TestMerger_Merge_Runtime_Plugins(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
runtime:
  plugins:
    - name: golang
      url: https://github.com/asdf-community/asdf-golang.git
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Len(t, merged.Runtime.Plugins, 1)
	assert.Equal(t, "golang", merged.Runtime.Plugins[0].Name)
}

func TestMerger_Merge_Runtime_Backend(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
runtime:
  backend: rtx
  scope: global
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.Equal(t, "rtx", merged.Runtime.Backend)
	assert.Equal(t, "global", merged.Runtime.Scope)
}

func TestMerger_Merge_Shell_SingleShell(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
shell:
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: robbyrussell
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	require.Len(t, merged.Shell.Shells, 1)
	assert.Equal(t, "zsh", merged.Shell.Shells[0].Name)
	assert.Equal(t, "oh-my-zsh", merged.Shell.Shells[0].Framework)
	assert.Equal(t, "robbyrussell", merged.Shell.Shells[0].Theme)
}

func TestMerger_Merge_Shell_LastWinsPerName(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
shell:
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: agnoster
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
shell:
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: powerlevel10k
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	require.Len(t, merged.Shell.Shells, 1)
	assert.Equal(t, "powerlevel10k", merged.Shell.Shells[0].Theme)
}

func TestMerger_Merge_Shell_Starship(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
shell:
  starship:
    enabled: true
    preset: nerd-font-symbols
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer})

	require.NoError(t, err)
	assert.True(t, merged.Shell.Starship.Enabled)
	assert.Equal(t, "nerd-font-symbols", merged.Shell.Starship.Preset)
}

func TestMerger_Merge_Shell_Env_DeepMerge(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
shell:
  env:
    EDITOR: vim
    PAGER: less
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
shell:
  env:
    EDITOR: nvim
    PATH_EXTRA: /usr/local/bin
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.Equal(t, "nvim", merged.Shell.Env["EDITOR"])
	assert.Equal(t, "less", merged.Shell.Env["PAGER"])
	assert.Equal(t, "/usr/local/bin", merged.Shell.Env["PATH_EXTRA"])
}

func TestMerger_Merge_Shell_Aliases_DeepMerge(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
shell:
  aliases:
    ll: ls -la
    vim: nvim
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
shell:
  aliases:
    ll: ls -lah
    k: kubectl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.Equal(t, "ls -lah", merged.Shell.Aliases["ll"])
	assert.Equal(t, "nvim", merged.Shell.Aliases["vim"])
	assert.Equal(t, "kubectl", merged.Shell.Aliases["k"])
}
