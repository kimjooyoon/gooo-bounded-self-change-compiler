#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${CYCLE_WORK_ROOT:?CYCLE_WORK_ROOT is required}
result=${CYCLE_INTEGRATION_OUT:?CYCLE_INTEGRATION_OUT is required}
candidate="$work/runtime/candidate"
before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test -x "$candidate"
"$candidate" after 0 > "$work/runtime/integration-zero.json"
jq -e '.verdict == "accept" and .accepted == true' "$work/runtime/integration-zero.json" >/dev/null
after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"

mkdir -p "$(dirname "$result")"
jq -n '{schema:"gooo/bounded-self-change/integration/v2",state:"CLOSED",caller_owned_output:true,temporary_candidate:true,repository_writes:0,remote_writes:0,apply_authority:0,commit_authority:0,merge_authority:0,semantic_root:"bounded-self-change-tests"}' > "$result"
jq -e '.state == "CLOSED" and .caller_owned_output == true and .temporary_candidate == true and .repository_writes == 0 and .apply_authority == 0 and .commit_authority == 0 and .merge_authority == 0' "$result" >/dev/null
