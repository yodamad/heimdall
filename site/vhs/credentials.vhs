Output ../docs/assets/heimdall-credentials-demo.gif

Set Shell "zsh"
Set FontSize 22
Set Width 2000
Set Height 1100
Set PlaybackSpeed 0.5
Set WindowBar Colorful

Require heimdall

# A configuration of its own, copied from credentials-demo.yml, so that the demo
# never reads nor writes the real one. Every credential in it is made up.
Hide
Type "export DEMO=$(mktemp -d)"
Enter
Type "cp credentials-demo.yml $DEMO/heimdall.yml"
Enter
Type "alias heimdall='heimdall -c $DEMO/heimdall.yml'"
Enter
Type "unset CORP_GITLAB_TOKEN; clear"
Enter
Show

# Which credential is broken, and why
Type "heimdall credentials check"
Sleep 500ms
Enter
Sleep 6s

Hide
Type "clear"
Enter
Show

# One secret for the two self hosted platforms sharing it
Type "heimdall credentials update"
Sleep 500ms
Enter
Sleep 4s

# github.com is not one of them
Down
Sleep 1s
Space
Sleep 1s
Enter
Sleep 2s

# Name of the keyring entry the two platforms will point at
Type "corp-demo"
Sleep 500ms
Enter
Sleep 1s

# The secret itself is never displayed
Type "glpat-aBrandNewTokenNobodySees"
Sleep 500ms
Enter
Sleep 1s
Type "glpat-aBrandNewTokenNobodySees"
Sleep 500ms
Enter
Sleep 6s

Hide
Type "clear"
Enter
Show

# Both platforms now read their token from the one keyring entry
Type "heimdall credentials list"
Sleep 500ms
Enter
Sleep 5s

# And the configuration file kept its comments
Type "cat $DEMO/heimdall.yml"
Sleep 500ms
Enter
Sleep 5s

Hide
Type "security delete-generic-password -s heimdall -a corp-demo > /dev/null 2>&1; rm -rf $DEMO"
Enter
Show
