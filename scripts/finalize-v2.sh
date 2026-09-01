#!/usr/bin/env bash
set -euo pipefail

bin=${1:?path to the compiler binary is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${CYCLE_WORK_ROOT:?CYCLE_WORK_ROOT is required}
metrics=${METRICS_DIR:?METRICS_DIR is required}
integration=${CYCLE_INTEGRATION_OUT:?CYCLE_INTEGRATION_OUT is required}
output=${CYCLE_OUTPUT_DIR:?CYCLE_OUTPUT_DIR is required}
meta="$root/.gooo/bounded-self-change-cycle-v2.gooo"
source="$root/examples/bounded-self-change-v2/internal-safe-change.gooo"
contract="$root/contracts/v0.2-cycle-locks.json"

jq -n \
	--slurpfile base "$work/execution-base.json" \
	--slurpfile compile "$metrics/compile.json" \
	--slurpfile build "$metrics/build.json" \
	--slurpfile test "$metrics/test.json" \
	--slurpfile conformance "$metrics/conformance.json" \
	--slurpfile integration_metric "$metrics/integration.json" \
	--slurpfile integration "$integration" \
	'($base[0] + {stage_measurements:{compile:$compile[0],build:$build[0],test:$test[0],conformance:$conformance[0],integration:$integration_metric[0]},indicators:[{id:"counterexample_acceptance",unit:"accepted_inputs",baseline:0,candidate:1,same_job:true,explicit_budget:false},{id:"negative_guardrail_acceptance",unit:"accepted_inputs",baseline:0,candidate:0,same_job:true,explicit_budget:false},{id:"positive_behavior_acceptance",unit:"accepted_inputs",baseline:1,candidate:1,same_job:true,explicit_budget:false}],integration_state:$integration[0].state})' > "$work/execution-final.json"

mkdir -p "$output"
"$bin" cycle finalize --meta "$meta" --source "$source" --contract "$contract" --mode internal --prepared "$work/prep/prepared-change-proposal.json" --execution "$work/execution-final.json" --metrics "$metrics" --integration "$integration" --out "$output" > "$work/finalize-summary.json"

test "$(find "$output" -maxdepth 1 -type f | awk 'END {print NR+0}')" -eq 8
for artifact in cycle-manifest.json frontier-receipt.json change-proposal.json test-impact-receipt.json measurement-receipt.json evidence-manifest.json next-wave-proposal.json human-report.md; do test -f "$output/$artifact"; done
jq -e '.decision == "CLOSED" and .vector == {total:12,closed:4,unknown:4,refuted:4} and .runtime_local_validation_commands == 0 and ([.required_outputs[]] | length) == 8' "$output/cycle-manifest.json" >/dev/null
jq -e '.content_addressed == true and .vector == {total:12,closed:4,unknown:4,refuted:4} and .repository_writes == 0 and .remote_writes == 0' "$output/evidence-manifest.json" >/dev/null
