# VHS tape for preflight agent demo
# Run with: cd website/public/demos/tapes && vhs agent.tape

Output ../gif/agent.gif

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

Type "# Background agent: Automatic drift detection and remediation"
Enter
Sleep 1s

Type "preflight agent --help"
Enter
Sleep 2500ms

Type ""
Enter
Sleep 500ms

Type "# Check agent status"
Enter
Sleep 500ms

Type "preflight agent status"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Start agent with 30-minute schedule"
Enter
Sleep 500ms

Type "preflight agent start --schedule 30m --remediation notify"
Enter
Sleep 2s

Type ""
Enter
Sleep 500ms

Type "# Install as system service (launchd/systemd)"
Enter
Sleep 500ms

Type "preflight agent install --help"
Enter
Sleep 2s

Sleep 1s
