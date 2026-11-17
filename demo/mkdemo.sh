#!/usr/bin/env bash
#
# Build a throwaway workspace of git repositories for the VHS recording.
#
#   ./demo/mkdemo.sh [root]      (default root: /tmp/heimdall-demo)
#
# Layout under <root>:
#   repos/      the tree Heimdall scans
#   origins/    bare "remotes" the repositories push to
#   worktrees/  git worktrees, kept out of the scanned tree
#   heimdall.yml  config pointing at repos/
#
# The root is deleted and rebuilt on every run, so each recording starts from
# the same state.
set -euo pipefail

ROOT="${1:-/tmp/heimdall-demo}"
REPOS="$ROOT/repos"
ORIGINS="$ROOT/origins"
WT="$ROOT/worktrees"

rm -rf "$ROOT"
mkdir -p "$REPOS" "$ORIGINS" "$WT"

export GIT_AUTHOR_NAME="Demo"
export GIT_AUTHOR_EMAIL="demo@example.com"
export GIT_COMMITTER_NAME="Demo"
export GIT_COMMITTER_EMAIL="demo@example.com"

# repo <path> <name> — a repository with one commit, pushed to its own bare origin
repo() {
  local path="$1" name="$2"
  git init -q --bare -b main "$ORIGINS/$name.git"
  mkdir -p "$path"
  git -C "$path" init -q -b main
  printf '# %s\n' "$name" > "$path/README.md"
  git -C "$path" add -A
  git -C "$path" commit -qm "Initial commit"
  git -C "$path" remote add origin "$ORIGINS/$name.git"
  git -C "$path" push -q -u origin main
}

repo "$REPOS/work/backend/asgard"      asgard
repo "$REPOS/work/backend/bifrost"     bifrost
repo "$REPOS/work/backend/gjallarhorn" gjallarhorn
repo "$REPOS/work/frontend/valhalla"   valhalla
repo "$REPOS/work/frontend/ragnarok"   ragnarok
repo "$REPOS/oss/yggdrasil"            yggdrasil
repo "$REPOS/oss/mjolnir"              mjolnir

# --- a different state per repository, so the list has something to say ----

# bifrost: uncommitted local changes
printf 'work in progress\n' >> "$REPOS/work/backend/bifrost/README.md"
printf 'draft\n' > "$REPOS/work/backend/bifrost/notes.md"

# valhalla: local changes, on a feature branch
git -C "$REPOS/work/frontend/valhalla" checkout -q -b feat/rainbow-bridge
printf 'tweak\n' >> "$REPOS/work/frontend/valhalla/README.md"

# gjallarhorn: behind origin, someone else pushed
CLONE="$ROOT/.clone"
git clone -q "$ORIGINS/gjallarhorn.git" "$CLONE"
printf 'upstream change\n' >> "$CLONE/README.md"
git -C "$CLONE" add -A
git -C "$CLONE" commit -qm "Upstream work"
git -C "$CLONE" push -q origin main
rm -rf "$CLONE"
git -C "$REPOS/work/backend/gjallarhorn" fetch -q origin

# ragnarok: a commit of its own, not pushed
printf 'local only\n' > "$REPOS/work/frontend/ragnarok/local.md"
git -C "$REPOS/work/frontend/ragnarok" add -A
git -C "$REPOS/work/frontend/ragnarok" commit -qm "Local work not pushed"

# asgard: a few branches, so the details pane has something to show
git -C "$REPOS/work/backend/asgard" branch release/1.0
git -C "$REPOS/work/backend/asgard" branch fix/heimdall-horn

# yggdrasil: three worktrees, two of them unused, one holding changes
git -C "$REPOS/oss/yggdrasil" worktree add -q -b wt/review "$WT/yggdrasil-review" >/dev/null 2>&1
git -C "$REPOS/oss/yggdrasil" worktree add -q -b wt/spike  "$WT/yggdrasil-spike"  >/dev/null 2>&1
git -C "$REPOS/oss/yggdrasil" worktree add -q -b wt/keepme "$WT/yggdrasil-keepme" >/dev/null 2>&1
printf 'do not drop me\n' > "$WT/yggdrasil-keepme/keep.md"

# mjolnir: one unused worktree
git -C "$REPOS/oss/mjolnir" worktree add -q -b wt/hotfix "$WT/mjolnir-hotfix" >/dev/null 2>&1

cat > "$ROOT/heimdall.yml" <<YAML
work_dir: $REPOS

morning_routine:
  shell: bash
  commands: git fetch --all, git gc --quiet, git status --short
YAML

mkdir -p "$ROOT/logs"
: > "$ROOT/logs/heimdall.log"

echo "demo workspace ready in $ROOT"
