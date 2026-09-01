#!/usr/bin/env bash
set -euo pipefail

bin=${1:?path to the compiler binary is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${CYCLE_WORK_ROOT:?CYCLE_WORK_ROOT is required}
mkdir -p "$work/prep" "$work/runtime"
meta="$root/.gooo/bounded-self-change-cycle-v2.gooo"
source="$root/examples/bounded-self-change-v2/internal-safe-change.gooo"
contract="$root/contracts/v0.2-cycle-locks.json"

before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
"$bin" cycle prepare --meta "$meta" --source "$source" --contract "$contract" --mode internal --out "$work/prep" > "$work/prepare-summary.json"

jq -e '.decision == "CLOSED" and (.candidate_digest | strings | startswith("sha256:"))' "$work/prepare-summary.json" >/dev/null
jq -r '.generated_go' "$work/prep/prepared-change-proposal.json" > "$work/runtime/candidate.go"
go build -trimpath -o "$work/runtime/candidate" "$work/runtime/candidate.go"

"$work/runtime/candidate" before 0 > "$work/runtime/before-zero.json"
"$work/runtime/candidate" after 0 > "$work/runtime/after-zero.json"
"$work/runtime/candidate" after -1 > "$work/runtime/after-negative.json"
"$work/runtime/candidate" after 1 > "$work/runtime/after-positive.json"

jq -e '.verdict == "reject" and .accepted == false' "$work/runtime/before-zero.json" >/dev/null
jq -e '.verdict == "accept" and .accepted == true' "$work/runtime/after-zero.json" >/dev/null
jq -e '.verdict == "reject" and .accepted == false' "$work/runtime/after-negative.json" >/dev/null
jq -e '.verdict == "accept" and .accepted == true' "$work/runtime/after-positive.json" >/dev/null

after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"

source_digest=$(jq -r '.source_digest' "$work/prep/prepared-change-proposal.json")
meta_digest=$(jq -r '.meta_digest' "$work/prep/prepared-change-proposal.json")
contract_digest=$(jq -r '.contract_digest' "$work/prep/prepared-change-proposal.json")
ir_digest=$(jq -r '.ir_digest' "$work/prep/prepared-change-proposal.json")
candidate_digest=$(jq -r '.candidate_digest' "$work/prep/prepared-change-proposal.json")
jq -n \
	--arg schema "gooo/bounded-self-change/execution/v2" \
	--arg scenario "deterministic-safe-self-improvement" \
	--arg mode internal \
	--arg source_digest "$source_digest" \
	--arg meta_digest "$meta_digest" \
	--arg contract_digest "$contract_digest" \
	--arg ir_digest "$ir_digest" \
	--arg candidate_digest "$candidate_digest" \
	--slurpfile before "$work/runtime/before-zero.json" \
	--slurpfile after_zero "$work/runtime/after-zero.json" \
	--slurpfile after_negative "$work/runtime/after-negative.json" \
	--slurpfile after_positive "$work/runtime/after-positive.json" \
	'{schema:$schema,scenario:$scenario,mode:$mode,source_digest:$source_digest,meta_digest:$meta_digest,contract_digest:$contract_digest,ir_digest:$ir_digest,candidate_digest:$candidate_digest,toolchain:"go1.27.0",runner:"ubuntu-latest",observations:[$before[0],$after_zero[0],$after_negative[0],$after_positive[0]],before_after:{before:0,after:1},stage_measurements:{},indicators:[],same_scope:true,same_job:true,positive_work:true,repository_writes:0,remote_writes:0,local_test_executions:0,cross_project_required_gates:0,integration_state:"PENDING"}' > "$work/execution-base.json"
