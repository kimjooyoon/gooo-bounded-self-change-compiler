#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
stage_root=${1:?stage measurement directory is required}
test_events=${2:?go test json event file is required}
conformance_counts=${3:?conformance counts file is required}
integration_result=${4:?integration result file is required}
conformance_work=${5:?conformance work directory is required}
output=${6:?machine evidence output path is required}

for stage in compile build test conformance integration; do
	test -f "$stage_root/$stage.json"
	jq -e --arg stage "$stage" '.stage == $stage and (.wall_ms | type) == "number" and (.wall_ms | floor) == .wall_ms and (.wall_ms >= 0) and (.peak_rss_kib | type) == "number" and (.peak_rss_kib | floor) == .peak_rss_kib and (.peak_rss_kib >= 0)' "$stage_root/$stage.json" >/dev/null
done

jq -s '{total:([.[] | select(.Action == "run" and (.Test // "") != "")] | length),selected:([.[] | select(.Action == "run" and (.Test // "") != "")] | length),executed:([.[] | select(.Action == "pass" and (.Test // "") != "")] | length),reused:([.[] | select((.Test // "") != "" and .Cached == true)] | length),failed:([.[] | select(.Action == "fail" and (.Test // "") != "")] | length),unknown:0}' "$test_events" > "$stage_root/tests.json"
jq -e '.total == 3 and .selected == 3 and .executed == 3 and .reused == 0 and .failed == 0 and .unknown == 0' "$stage_root/tests.json" >/dev/null
jq -e '.total == 9 and .selected == 9 and .executed == 9 and .reused == 0 and .closed == 3 and .unknown == 3 and .refuted == 3' "$conformance_counts" >/dev/null
jq -e '.state == "CLOSED" and .caller_owned_output == true and .exact_before_after == true and .repository_writes == 0' "$integration_result" >/dev/null

report="$conformance_work/first/decision-dossier.json"
human_report="$conformance_work/first/human-report.md"
test -f "$report"
test -f "$human_report"
"$root/scripts/collect-inventory.sh" "$root" "$conformance_work/first" "$stage_root/inventory.json"
source_digest=$(jq -r '.source_digest' "$report")
meta_digest=$(jq -r '.meta_digest' "$report")
contract_digest=$(jq -r '.contract_digest' "$report")
ir_digest=$(jq -r '.ir_digest' "$report")
candidate_digest=$(jq -r '.candidate_digest' "$report")
report_digest="sha256:$(sha256sum "$report" | awk '{print $1}')"
evidence_dir="$(dirname "$output")/evidence"
mkdir -p "$evidence_dir"
cp "$report" "$evidence_dir/decision-dossier.json"
cp "$human_report" "$evidence_dir/human-report.md"
cp "$conformance_work/first/semantic-ir.json" "$evidence_dir/semantic-ir.json"
cp "$conformance_work/first/semantic-graph.json" "$evidence_dir/semantic-graph.json"
cp "$conformance_work/first/candidate.gooo" "$evidence_dir/candidate.gooo"
cp "$conformance_work/first/candidate.go" "$evidence_dir/candidate.go"
cp "$conformance_work/first/proposal.patch.json" "$evidence_dir/proposal.patch.json"

jq -n \
	--arg schema "gooo/bounded-self-change/ci-evidence/v1" \
	--arg commit "${GITHUB_SHA:-unknown}" \
	--arg source_digest "$source_digest" \
	--arg meta_digest "$meta_digest" \
	--arg contract_digest "$contract_digest" \
	--arg ir_digest "$ir_digest" \
	--arg candidate_digest "$candidate_digest" \
	--arg report_digest "$report_digest" \
	--slurpfile compile "$stage_root/compile.json" \
	--slurpfile build "$stage_root/build.json" \
	--slurpfile test "$stage_root/test.json" \
	--slurpfile conformance "$stage_root/conformance.json" \
	--slurpfile integration "$stage_root/integration.json" \
	--slurpfile tests "$stage_root/tests.json" \
	--slurpfile counts "$conformance_counts" \
	--slurpfile integration_result "$integration_result" \
	--slurpfile inventory "$stage_root/inventory.json" \
	--slurpfile report "$report" \
	'{schema:$schema,commit:$commit,source_digest:$source_digest,meta_digest:$meta_digest,contract_digest:$contract_digest,ir_digest:$ir_digest,candidate_digest:$candidate_digest,report_digest:$report_digest,stage_measurements:{compile:($compile[0]|{wall_ms,peak_rss_kib}),build:($build[0]|{wall_ms,peak_rss_kib}),test:($test[0]|{wall_ms,peak_rss_kib}),conformance:($conformance[0]|{wall_ms,peak_rss_kib}),integration:($integration[0]|{wall_ms,peak_rss_kib})},tests:$tests[0],conformance:$counts[0],integration:$integration_result[0],inventory:$inventory[0],runtime_authority:{repository_writes:0,apply_authority:0,commit_authority:0,merge_authority:0,tag_authority:0,release_authority:0,local_test_executions:0,cross_project_required_gates:0},local_authority:{test:0,build:0,vet:0,lint:0,format:0,check:0,conformance:0,integration:0},remote_authority:{github_actions_validation_runs:1,github_actions_validation_jobs:5,github_token:1,cross_project_required_gates:0},improvement:$report[0].improvement,utility:$report[0].utility,artifacts:{generated_count:$inventory[0].generated_artifacts.count,generated_bytes:$inventory[0].generated_artifacts.bytes,human_report:"evidence/human-report.md",machine_report:"evidence/decision-dossier.json"}}' > "$output"
cp "$output" "$evidence_dir/ci-evidence.json"
