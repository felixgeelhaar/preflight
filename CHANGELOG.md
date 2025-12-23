# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure with DDD architecture
- Configuration domain with layer merging and provenance tracking
- Compiler domain with step graph compilation
- Execution domain with DAG scheduler
- CLI commands: `plan`, `apply`, `version`

### Providers
- **brew**: Homebrew package management (taps, formulae, casks)
- **apt**: Debian/Ubuntu package management with PPAs
- **files**: Dotfile management (links, templates, copies)
- **git**: .gitconfig generation with aliases and includes
- **ssh**: ~/.ssh/config generation with host and match blocks
- **runtime**: Tool version management (rtx/asdf compatible)
- **shell**: Shell configuration (zsh/bash/fish, oh-my-zsh, starship)
- **nvim**: Neovim configuration (LazyVim, NvChad, AstroNvim, Kickstart)
- **vscode**: VSCode configuration (extensions, settings, keybindings)

### Infrastructure
- GitHub Actions CI pipeline
- golangci-lint configuration with 23+ linters
- coverctl integration for domain-aware coverage enforcement
- TDD workflow with 80%+ coverage requirement

[Unreleased]: https://github.com/felixgeelhaar/preflight/commits/main
