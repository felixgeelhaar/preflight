# VHS tape for preflight tour demo
# Run with: cd website/public/demos/tapes && vhs tour.tape

Output ../gif/tour.gif

Set Shell "zsh"
Set FontSize 16
Set Width 900
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

Type "# Interactive guided walkthroughs"
Enter
Sleep 800ms

Type "preflight tour --list"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Start the basics tour"
Enter
Sleep 500ms

Type "preflight tour basics"
Enter
Sleep 2s

# Scroll through content
Sleep 500ms
Down
Sleep 300ms
Down
Sleep 300ms

# Navigate to next section
Type "n"
Sleep 1s

# Navigate to next section again
Type "n"
Sleep 1s

# Show section navigation
Type "2"
Sleep 1s

# Quit
Type "q"
Sleep 500ms
