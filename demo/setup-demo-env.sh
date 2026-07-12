#!/usr/bin/env bash
#
# setup-demo-env.sh — build rulem and provision a self-contained demo sandbox
# for the VHS tape scripts in this directory.
#
# The sample content lives as real files under demo/fixtures/ (tracked in git),
# so this script only wires them into a throwaway sandbox and generates a
# config.yaml. To change what the demos show, edit the fixtures, not this script.
#
# What it creates (everything under demo/sandbox/, all git-ignored):
#   demo/rulem-demo               the freshly built binary the tapes run
#   demo/sandbox/config/          an XDG_CONFIG_HOME dir containing rulem/config.yaml
#   demo/sandbox/rules-repos/*/   one directory per demo central repository
#   demo/sandbox/project/         a working dir with rule files to save/import
#
# The generated config.yaml points at the *absolute* sandbox paths, so it works
# no matter where the repo is checked out. The script is idempotent.
#
# Usage:
#   ./demo/setup-demo-env.sh
#   XDG_CONFIG_HOME="$PWD/demo/sandbox/config" ./demo/rulem-demo   # try it live
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURES="$SCRIPT_DIR/fixtures"
SANDBOX="$SCRIPT_DIR/sandbox"
CONFIG_DIR="$SANDBOX/config/rulem"
REPOS_DIR="$SANDBOX/rules-repos"
PROJECT_DIR="$SANDBOX/project"
BINARY="$SCRIPT_DIR/rulem-demo"

# Central repositories to register, as "fixture-dir|Display Name|type". The type
# is "local" or "github". Add a line here (and a matching
# demo/fixtures/repos/<dir>/) to grow the demo store.
#
# A "github" entry is provisioned as a real local git clone whose `origin` points
# at the URL below, but kept intentionally dirty. rulem's fetch short-circuits on
# a dirty working tree (see repository/git.go performFetch), so the demo shows a
# GitHub-typed repo with its full remote UI without ever touching the network.
REPOS=(
  "engineering-standards|Engineering Standards|local"
  "backend-standards|Backend Standards|github"
  "python-guide|Python Guide|local"
)

# Base for the fake-but-well-formed remote URL of github-typed demo repos.
GITHUB_OWNER="https://github.com/muhammadbassiony"

echo "==> Building demo binary -> $BINARY"
( cd "$REPO_ROOT" && go build -o "$BINARY" ./cmd/rulem )

echo "==> Provisioning sandbox -> $SANDBOX"
rm -rf "$SANDBOX"
mkdir -p "$CONFIG_DIR" "$REPOS_DIR"
cp -R "$FIXTURES/project" "$PROJECT_DIR"

# Fixed base timestamp keeps generated IDs/config deterministic across re-runs.
BASE_TS=1728756432

# provision_github_cache turns a copied fixture dir into a git repo whose origin
# matches its config URL, then leaves it dirty so rulem never fetches it.
provision_github_cache() {
  local dest="$1" url="$2"
  git -C "$dest" init -q -b main
  git -C "$dest" add -A
  GIT_AUTHOR_NAME="rulem demo" GIT_AUTHOR_EMAIL="demo@rulem.local" \
  GIT_COMMITTER_NAME="rulem demo" GIT_COMMITTER_EMAIL="demo@rulem.local" \
  GIT_AUTHOR_DATE="@${BASE_TS} +0000" GIT_COMMITTER_DATE="@${BASE_TS} +0000" \
    git -C "$dest" commit -q --no-gpg-sign -m "Import demo rules"
  git -C "$dest" remote add origin "$url"
  # Non-markdown, untracked -> keeps the tree dirty (fetch skipped) but stays out
  # of the file picker, which only lists markdown files.
  : > "$dest/.sync-cache"
}

echo "==> Generating config.yaml -> $CONFIG_DIR/config.yaml"
{
  echo 'version: "1.0"'
  echo "init_time: ${BASE_TS}"
  echo "repositories:"
  i=0
  for entry in "${REPOS[@]}"; do
    dir="${entry%%|*}"
    rest="${entry#*|}"
    name="${rest%|*}"
    type="${rest##*|}"
    ts=$((BASE_TS + i))
    dest="${REPOS_DIR}/${dir}"
    cp -R "$FIXTURES/repos/$dir" "$dest"

    echo "    - id: ${dir}-${ts}"
    echo "      name: ${name}"
    echo "      type: ${type}"
    echo "      created_at: ${ts}"
    echo "      path: ${dest}"
    if [ "$type" = "github" ]; then
      url="${GITHUB_OWNER}/rulem-demo-${dir}.git"
      provision_github_cache "$dest" "$url"
      echo "      remote_url: ${url}"
      echo "      branch: main"
      echo "      last_sync_time: ${ts}"
    fi
    i=$((i + 1))
  done
} > "$CONFIG_DIR/config.yaml"

echo "==> Done. ${#REPOS[@]} repositories registered."
echo "    Try it:  XDG_CONFIG_HOME=\"$SANDBOX/config\" \"$BINARY\""
