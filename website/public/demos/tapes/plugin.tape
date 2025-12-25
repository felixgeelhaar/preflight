# VHS tape for preflight plugin demo
# Run with: cd website/public/demos/tapes && vhs plugin.tape

Output ../gif/plugin.gif

Set Shell "zsh"
Set FontSize 16
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

Type "# Plugin ecosystem - extend Preflight"
Enter
Sleep 800ms

Type "preflight plugin --help"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# List installed plugins"
Enter
Sleep 500ms

Type "preflight plugin list"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Search for plugins on GitHub"
Enter
Sleep 500ms

Type "preflight plugin search --help"
Enter
Sleep 2s

Sleep 1s
