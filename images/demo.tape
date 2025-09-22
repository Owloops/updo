# VHS Demo for updo

# https://github.com/charmbracelet/vhs

Output demo.gif
Set FontSize 18
Set Width 1280
Set Height 720
Set Theme "OneDark"

# Start with a clean terminal
Type "# updo Demo"
Sleep 1s
Enter

# Start updo monitoring
Type "./updo monitor --profile Owloops-Development --config example-config.toml"
Sleep 1s
Enter
Sleep 5s

# Press arrow down several times to show navigation
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms

# Now demonstrate search functionality
Type "/"
Sleep 500ms
Type "GitHub"
Sleep 1s
Enter
Sleep 3s

# Show the filtered results for a moment
Sleep 4s

# Exit the search/filter
Escape
Sleep 2s

# Exit the application
Ctrl+C
Sleep 2s
