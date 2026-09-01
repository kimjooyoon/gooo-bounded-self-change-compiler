#!/usr/bin/env bash
set -euo pipefail

bin=${1:?path to the compiler binary is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${LIVE_WORK_ROOT:?LIVE_WORK_ROOT is required}
output=${LIVE_OUTPUT_DIR:?LIVE_OUTPUT_DIR is required}
mkdir -p "$work/prep" "$work/api"
meta="$root/.gooo/bounded-self-change-cycle-v2.gooo"
source="$root/examples/bounded-self-change-v2/internal-safe-change.gooo"
contract="$root/contracts/v0.2-cycle-locks.json"
before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')

# Read only immutable release metadata. The ledger payload/source tree is never copied.
gh api 'repos/kimjooyoon/gooo-self-improvement-ledger/releases/tags/v0.50.0' > "$work/api/release.json"
gh api 'repos/kimjooyoon/gooo-self-improvement-ledger/git/ref/tags/v0.50.0' > "$work/api/ref.json"
tag_object=$(jq -r '.object.sha' "$work/api/ref.json")
gh api "repos/kimjooyoon/gooo-self-improvement-ledger/git/tags/$tag_object" > "$work/api/tag.json"
jq -e '.immutable == true and .tag_name == "v0.50.0" and ([.assets[] | select(.digest == "sha256:80575837d8ebb8d838bab912ff7802946fb37b2d90d923e8a9cec27bdf543e25")] | length) == 1' "$work/api/release.json" >/dev/null
jq -e '.object.type == "tag"' "$work/api/ref.json" >/dev/null
jq -e '.object.type == "commit" and .object.sha == "e93768f4204e8a88214026ffa22febad7ecedcbd"' "$work/api/tag.json" >/dev/null

"$bin" cycle prepare --meta "$meta" --source "$source" --contract "$contract" --mode live --out "$work/prep" > "$work/prepare-summary.json"
source_digest=$(jq -r '.source_digest' "$work/prep/prepared-change-proposal.json")
meta_digest=$(jq -r '.meta_digest' "$work/prep/prepared-change-proposal.json")
contract_digest=$(jq -r '.contract_digest' "$work/prep/prepared-change-proposal.json")
ir_digest=$(jq -r '.ir_digest' "$work/prep/prepared-change-proposal.json")
asset=$(jq -c '[.assets[] | select(.digest == "sha256:80575837d8ebb8d838bab912ff7802946fb37b2d90d923e8a9cec27bdf543e25")][0]' "$work/api/release.json")
jq -n \
	--arg source_digest "$source_digest" --arg meta_digest "$meta_digest" --arg contract_digest "$contract_digest" --arg ir_digest "$ir_digest" \
	--argjson asset "$asset" \
	'{schema:"gooo/bounded-self-change/execution/v2",scenario:"deterministic-safe-self-improvement",mode:"live",source_digest:$source_digest,meta_digest:$meta_digest,contract_digest:$contract_digest,ir_digest:$ir_digest,candidate_digest:"",toolchain:"not-run",runner:"ubuntu-latest",observations:[],before_after:null,stage_measurements:{},indicators:[],same_scope:false,same_job:false,positive_work:false,repository_writes:0,remote_writes:0,local_test_executions:0,cross_project_required_gates:0,integration_state:"STOPPED",live_ledger:{lock:{id:"SELF_IMPROVEMENT_LEDGER_V0_50",repository:"kimjooyoon/gooo-self-improvement-ledger",tag:"v0.50.0",release_id:380866481,immutable:true,tag_object_sha:"9e3263ea902bef64fa31c05ca7c1ab038ef962ef",asset_name:$asset.name,asset_id:$asset.id,asset_size_bytes:$asset.size,asset_digest:$asset.digest},actionable_frontier:"EXTERNAL_UTILITY_EVIDENCE",automation_can_produce:false,external_evidence:true,decision:"UNKNOWN"}}' > "$work/execution-live.json"

mkdir -p "$output"
"$bin" cycle finalize --meta "$meta" --source "$source" --contract "$contract" --mode live --prepared "$work/prep/prepared-change-proposal.json" --execution "$work/execution-live.json" --out "$output" > "$work/finalize-summary.json"
after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
test "$(find "$output" -maxdepth 1 -type f | awk 'END {print NR+0}')" -eq 8
jq -e '.decision == "UNKNOWN" and .claim.state == "UNKNOWN" and .claim.unknown_class == "HUMAN_EXTERNAL_EVIDENCE_REQUIRED" and .runtime_local_validation_commands == 0' "$output/cycle-manifest.json" >/dev/null
jq -e '.counterexample_accepted == false and .claim.state == "UNKNOWN"' "$output/next-wave-proposal.json" >/dev/null
