#!/bin/zsh
# Re-bench all registered models at max + one-below-max (or top pair / plain),
# publishing web/data.json + gh-pages after EACH model. data.json = last run per model.
set -u
cd /Users/ankityadav/agent/opencode-bench
mkdir -p results
BOARD=results/board-runs.json
LOG=results/bench-all.log

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG"; echo "$*"; }

MODELS=(
  "opencode-go/deepseek-v4-flash|max|-"
  "opencode-go/deepseek-v4-flash-vision-exp|max|-"
  "opencode-go/deepseek-v4-pro|max|-"
  "google/gemini-3.7-flash|medium|-"
  "opencode-go/gpt-5.6-luna|max|-"
  "opencode/muse-spark-1.2|xhigh|-"
)

log "=== START bench-all $(date) ==="
i=0
for spec in $MODELS; do
  i=$((i+1))
  m=${spec%%|*}; rest=${spec#*|}; v1=${rest%%|*}; v2=${rest#*|}
  if [ "$v1" = "-" ]; then mlist="$m"; vs="plain"
  elif [ "$v2" = "-" ]; then mlist="$m@$v1"; vs="$v1"
  else mlist="$m@$v1,$m@$v2"; vs="$v1+$v2"; fi
  log "=== [$i/${#MODELS[@]}] BENCH $m ($vs) ==="
  RUNLOG="results/bench-$i-$(echo $m | tr '/' '_').log"
  if ! ./ocbench -models "$mlist" -runs 3 -parallel 8 > "$RUNLOG" 2>&1; then
    log "!!! BENCH FAILED $m ($vs) - skipping publish"
    continue
  fi
  NEW=$(grep -o 'results/run-[0-9-]*\.json' "$RUNLOG" | head -1)
  if [ -z "$NEW" ]; then log "!!! no run file for $m - skipping publish"; continue; fi
  python3 - "$BOARD" "$NEW" <<'PY'
import json,sys,os
board,newf=sys.argv[1],sys.argv[2]
runs=json.load(open(board)) if os.path.exists(board) else []
new=json.load(open(newf))
keys={(r['model'],r.get('variant','')) for r in new}
runs=[r for r in runs if (r['model'],r.get('variant','')) not in keys]
runs.extend(new)
json.dump(runs,open(board,'w'))
PY
  BOARD_TOTAL=$(python3 -c "import json;print(len(json.load(open('$BOARD'))))")
  STATS=$(python3 - "$BOARD" "$m" <<'PY'
import json,sys
runs=json.load(open(sys.argv[1])); m=sys.argv[2]
sel=[r for r in runs if r['model']==m and r['passed']]
allsel=[r for r in runs if r['model']==m]
print(f"{len(sel)}/{len(allsel)}")
PY
  )
  log "board runs: $BOARD_TOTAL | $m $STATS"
  ./ocbench -report "$BOARD" > /dev/null 2>&1 || { log "!!! report failed for $m"; continue; }
  git add web/data.json
  git commit -q -m "bench: $m ($vs): $STATS" || log "commit no-op"
  git push -q origin main 2>&1 | tail -1 >> "$LOG"
  ./sync_ghpages.sh "sync: $m ($vs) $STATS" >> "$LOG" 2>&1
  log "=== published $m ($vs) $STATS ==="
done
log "=== DONE bench-all $(date) ==="
