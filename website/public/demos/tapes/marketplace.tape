# VHS tape for preflight marketplace demo
# Run with: cd website/public/demos/tapes && vhs marketplace.tape

Output ../gif/marketplace.gif

Set Shell "zsh"
Set FontSize 16
Set Width 900
Set Height 550
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

Type "# Community marketplace for presets"
Enter
Sleep 800ms

Type "preflight marketplace --help"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# List installed packages"
Enter
Sleep 500ms

Type "preflight marketplace list"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Alias: mp for marketplace"
Enter
Sleep 500ms

Type "preflight mp search --help"
Enter
Sleep 2s

Sleep 1s
