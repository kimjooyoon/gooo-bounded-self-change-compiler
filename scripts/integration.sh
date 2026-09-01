#!/usr/bin/env bash
set -euo pipefail

bin=${1:?path to the compiler binary is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
result=${INTEGRATION_RESULT_OUT:?INTEGRATION_RESULT_OUT is required}
work=${INTEGRATION_WORK_ROOT:?INTEGRATION_WORK_ROOT is required}
out="$work/first"
source="$root/examples/bounded-self-change-v1/self-change.gooo"
meta="$root/.gooo/bounded-self-change-compiler.gooo"
contract="$root/contracts/denominator-v1.json"

mkdir -p "$out"
before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
"$bin" run --meta "$meta" --source "$source" --contract "$contract" --out "$out" >/dev/null
go build -trimpath -o "$work/candidate" "$out/candidate.go"
"$work/candidate" before 0 > "$out/before-counterexample.json"
"$work/candidate" after 0 > "$out/after-counterexample.json"
"$work/candidate" after -1 > "$out/after-negative.json"
"$work/candidate" after 1 > "$out/after-positive.json"
source_digest=$(jq -r '.source_digest' "$out/semantic-ir.json")
meta_digest=$(jq -r '.meta_digest' "$out/semantic-ir.json")
contract_digest=$(jq -r '.contract_digest' "$out/semantic-ir.json")
ir_digest=$(jq -r '.ir_digest' "$out/semantic-ir.json")
jq -n \
	--arg schema "gooo/bounded-self-change/execution/v1" \
	--arg scenario bounded-zero-acceptance \
	--arg source_digest "$source_digest" \
	--arg meta_digest "$meta_digest" \
	--arg contract_digest "$contract_digest" \
	--arg ir_digest "$ir_digest" \
	--arg toolchain go1.27.0 \
	--arg runner ubuntu-latest \
	--slurpfile before "$out/before-counterexample.json" \
	--slurpfile after_zero "$out/after-counterexample.json" \
	--slurpfile after_negative "$out/after-negative.json" \
	--slurpfile after_positive "$out/after-positive.json" \
	'{schema:$schema,scenario:$scenario,source_digest:$source_digest,meta_digest:$meta_digest,contract_digest:$contract_digest,ir_digest:$ir_digest,toolchain:$toolchain,runner:$runner,observations:[$before[0],$after_zero[0],$after_negative[0],$after_positive[0]],before_after:{scenario:$scenario,source_digest:$source_digest,meta_digest:$meta_digest,contract_digest:$contract_digest,fixture:"zero_rejected",toolchain:$toolchain,runner:$runner,metric:"counterexample_acceptance",before:0,after:1}}' \
	> "$out/execution-evidence.json"
"$bin" verify --meta "$meta" --source "$source" --contract "$contract" --out "$out" --observations "$out/execution-evidence.json" >/dev/null
jq -e '.decision == "CANDIDATE_ACCEPTABLE" and .improvement.state == "CLOSED" and .exact_before_after.before == 0 and .exact_before_after.after == 1' "$out/decision-dossier.json" >/dev/null
after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"

jq -n \
	--arg schema "gooo/bounded-self-change/integration/v1" \
	--arg report_digest "sha256:$(sha256sum "$out/decision-dossier.json" | awk '{print $1}')" \
	'{schema:$schema,scenario:"bounded-zero-acceptance",state:"CLOSED",caller_owned_output:true,artifact_count:11,report_digest:$report_digest,ephemeral_compile:true,ephemeral_run:true,exact_before_after:true,repository_writes:0,apply_authority:0,commit_authority:0,merge_authority:0}' > "$result"
jq -e '.state == "CLOSED" and .caller_owned_output == true and .ephemeral_compile == true and .ephemeral_run == true and .exact_before_after == true and .repository_writes == 0 and .apply_authority == 0 and .commit_authority == 0 and .merge_authority == 0' "$result" >/dev/null
