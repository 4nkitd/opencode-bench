#!/bin/zsh
# Sync web/* from main to gh-pages branch and push. Usage: sync_ghpages.sh "commit msg"
set -e
cd /Users/ankityadav/agent/opencode-bench
msg="$1"
git fetch origin -q
git checkout -q -B gh-pages-sync origin/gh-pages
git show main:web/data.json > data.json
git show main:web/index.html > index.html
git show main:web/REDESIGN.md > REDESIGN.md
git add data.json index.html REDESIGN.md
if git diff --cached --quiet; then
  echo "gh-pages: no changes"
else
  git commit -q -m "$msg"
  git push -q origin HEAD:refs/heads/gh-pages
  echo "gh-pages: pushed ($msg)"
fi
git checkout -q main
git branch -q -D gh-pages-sync
