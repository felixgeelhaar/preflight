# VHS tape for preflight sync demo
# Run with: cd website/public/demos/tapes && vhs sync.tape

Output ../gif/sync.gif

Set Shell "zsh"
Set FontSize 16
Set Width 900
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

Hide
Type "export PATH=/Users/felixgeelhaar/Developer/projects/preflight/bin:$PATH"
Enter
Type "cd /Users/felixgeelhaar/Developer/projects/preflight/website/public/demos/test-config"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 100ms
Show

Type "# Multi-machine sync: Keep configs in sync across devices"
Enter
Sleep 1s

Type "preflight sync --help | head -25"
Enter
Sleep 3s

Type ""
Enter
Sleep 500ms

Type "# Sync with dry-run to preview changes"
Enter
Sleep 500ms

Type "preflight sync --dry-run"
Enter
Sleep 2500ms

Type ""
Enter
Sleep 500ms

Type "# Sync and push local changes to remote"
Enter
Sleep 500ms

Type "preflight sync --push --yes"
Enter
Sleep 2500ms

Sleep 1s
