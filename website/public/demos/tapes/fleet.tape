# VHS tape for preflight fleet demo
# Run with: cd website/public/demos/tapes && vhs fleet.tape

Output ../gif/fleet.gif

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

Type "# Fleet management: Configure multiple machines over SSH"
Enter
Sleep 1s

Type "preflight fleet --help"
Enter
Sleep 2500ms

Type ""
Enter
Sleep 500ms

Type "# List all hosts in inventory"
Enter
Sleep 500ms

Type "preflight fleet list"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Target hosts by group"
Enter
Sleep 500ms

Type "preflight fleet list --target @production"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Target by tag"
Enter
Sleep 500ms

Type "preflight fleet list --target tag:darwin"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Rolling deployment strategy"
Enter
Sleep 500ms

Type "preflight fleet apply --target @all --strategy rolling --help"
Enter
Sleep 2s

Sleep 1s
