# VHS demo for updo, see https://github.com/charmbracelet/vhs
# Record from the repo root after `go build -o updo .`:
#   vhs images/demo.tape
# Multi-region targets need `updo aws deploy` and a valid AWS profile.

Output images/demo.gif

Set FontSize 18
Set Width 1280
Set Height 720
Set Theme {"name":"Updo","black":"#111111","red":"#FF6B5B","green":"#5BFF6B","yellow":"#F59E0B","blue":"#8A4A22","magenta":"#C94F0E","cyan":"#FF6C19","white":"#E5E5E5","brightBlack":"#888888","brightRed":"#FF6B5B","brightGreen":"#5BFF6B","brightYellow":"#F59E0B","brightBlue":"#8A4A22","brightMagenta":"#C94F0E","brightCyan":"#FF6C19","brightWhite":"#FFFFFF","background":"#0A0A0A","foreground":"#E5E5E5","selection":"#2A2A2A","cursor":"#FF6C19"}

Type "# updo demo"
Sleep 500ms
Enter

Type "./updo monitor --profile Owloops-Development -C images/demo-config.toml"
Sleep 500ms
Enter

# Checks run in three regions, give the Lambdas a moment
Sleep 3s

# First row is a target group header, step onto its region rows to see data
Down
Sleep 2500ms
Down
Sleep 1200ms
Down
Sleep 1200ms
Up
Sleep 900ms
Up
Sleep 900ms

# Filter to a single target
Type "/"
Sleep 500ms
Type "Cloudflare"
Sleep 1200ms
Enter
Sleep 2500ms
Escape
Sleep 1500ms

# Quit
Ctrl+C
Sleep 1s
