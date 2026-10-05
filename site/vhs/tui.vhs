Output ../docs/assets/heimdall-tui-demo.gif

Set Shell "zsh"
Set FontSize 24
Set Width 2000
Set Height 1200
Set PlaybackSpeed 0.5
Set WindowBar Colorful

# Setup: the TUI is not in the released version yet, use the local build
Hide
Type "alias heimdall=/Users/yodamad/work/github/heimdall/heimdall"
Enter
# Put a repository behind origin to have something to pull, unless it holds local work
Type "git -C ~/work/demo/github/yodamad/heimdall reset -q --keep origin/main~5"
Enter
Type 'clear' Enter
Sleep 100ms
Show

Type "heimdall -w ~/work/demo/github"
Sleep 500ms
Enter
Sleep 2s

# Move in the tree: the details are the ones of the folder, then of the repository
Down
Sleep 1500ms
Down
Sleep 3s

# Back to the folder to collapse it, then all of them, and expand them back
Left
Sleep 1s
Enter
Sleep 1s
Type "z"
Sleep 1s
Type "z"
Sleep 1s

# Fetch all the repositories
Type "a"
Sleep 500ms
Type "f"
Sleep 4s

# Pull the ones which can be
Type "u"
Sleep 1500ms
Type "p"
Sleep 1s
Type "y"
Sleep 4s
Escape
Sleep 500ms

# Filter, and run a command on the repository left
Type "/"
Sleep 500ms
Type "heim"
Sleep 500ms
Enter
Sleep 500ms
Down
Sleep 1s
Type "!"
Sleep 500ms
Type "git status -sb"
Sleep 500ms
Enter
Sleep 3s

# Details of the repository
Tab
Sleep 1500ms
Tab
Sleep 500ms
Escape
Sleep 1s

# Help
Type "?"
Sleep 3s
Type "q"
Sleep 500ms
Type "q"
Sleep 500ms
