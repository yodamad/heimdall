Output site/docs/assets/heimdall-features.gif
Output site/docs/assets/heimdall-features.mp4

# A tour of Heimdall, recorded with VHS (https://github.com/charmbracelet/vhs).
#
#   go build -o heimdall .
#   ./demo/mkdemo.sh            # throwaway repositories in /tmp/heimdall-demo
#   vhs demo/features.tape      # run from the root of the repository
#
# demo/bin/heimdall puts the binary on the demo workspace, so nothing here
# touches the repositories you actually work on.

Set Shell bash
Set FontSize 20
Set Width 1800
Set Height 760
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

Hide
Type "export PATH=$PWD/demo/bin:$PATH" Enter
Type "clear" Enter
Show

Sleep 1s

# ── The list ──────────────────────────────────────────────────────────────
# Every repository under the work dir, grouped by folder, the ones needing
# attention first. WT counts the worktrees a repository has besides its own.
Type "heimdall" Enter
Sleep 5s

# ── The details of one repository ─────────────────────────────────────────
# yggdrasil: its branches, its worktrees, its last commits.
Down
Sleep 1500ms
Tab
Sleep 3500ms
Tab
Sleep 1s

# ── Filter by path ────────────────────────────────────────────────────────
Type "/"
Sleep 500ms
Type "backend"
Sleep 2500ms
Escape
Sleep 1500ms

# ── Fetch every repository at once ────────────────────────────────────────
Type "a"
Sleep 1s
Type "f"
Sleep 4s

# ── Pull the ones that can be pulled ──────────────────────────────────────
Type "u"
Sleep 2s
Type "p"
Sleep 3500ms

# ── Remove the worktrees holding nothing ──────────────────────────────────
Type "a"
Sleep 800ms
Type "w"
Sleep 2500ms
Type "y"
Sleep 4s

# ── Run the morning routine, and watch it go ──────────────────────────────
Type "m"
Sleep 2500ms
Type "y"
Sleep 7s

# ── The keys, all of them ─────────────────────────────────────────────────
Type "?"
Sleep 4s
Escape
Sleep 1s

Type "q"
Sleep 2s
