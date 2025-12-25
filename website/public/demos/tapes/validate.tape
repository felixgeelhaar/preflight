# VHS tape for preflight validate demo
# Run with: cd website/public/demos/tapes && vhs validate.tape

Output ../gif/validate.gif

Set Shell "zsh"
Set FontSize 16
Set Width 900
Set Height 500
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

Type "# Validate configuration for CI/CD"
Enter
Sleep 800ms

Type "preflight validate"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# With JSON output for automation"
Enter
Sleep 500ms

Type "preflight validate --json | jq ."
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Validate a different target"
Enter
Sleep 500ms

Type "preflight validate --target work"
Enter
Sleep 2s

Sleep 1s
