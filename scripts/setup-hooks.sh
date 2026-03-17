#!/usr/bin/env bash
# Install git hooks for preflight development.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"

install_hook() {
  local name="$1"
  local src="$REPO_ROOT/scripts/$name"
  local dst="$HOOKS_DIR/$name"

  if [ ! -f "$src" ]; then
    echo "Warning: $src not found — skipping"
    return
  fi

  ln -sf "$src" "$dst"
  chmod +x "$dst"
  echo "Installed $name hook"
}

install_hook pre-commit

echo "Done. Hooks installed to $HOOKS_DIR"
