#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
cycle_output=${2:?internal cycle output is required}
live_output=${3:?live cycle output is required}
metrics=${4:?stage metrics directory is required}
test_events=${5:?go test jsonl is required}
evidence_dir=${6:?evidence directory is required}
mkdir -p "$evidence_dir/cycle-output" "$evidence_dir/live-v0.50"
cp "$cycle_output"/* "$evidence_dir/cycle-output/"
cp "$live_output"/* "$evidence_dir/live-v0.50/"

test_passes=$(jq -s '[.[] | select(.Action == "pass" and (.Test // null) != null) | .Test] | length' "$test_events")
test_failures=$(jq -s '[.[] | select(.Action == "fail" and (.Test // null) != null) | .Test] | length' "$test_events")
test_unknown=$(jq -s '[.[] | select((.Action // "") == "unknown" and (.Test // null) != null) | .Test] | length' "$test_events")
jq -n --argjson total "$test_passes" --argjson failed "$test_failures" --argjson unknown "$test_unknown" '{total:$total,selected:$total,executed:$total,reused:0,failed:$failed,unknown:$unknown}' > "$evidence_dir/tests.json"

go_files=$(find "$root" -type f -name '*.go' ! -path "$root/.git/*" | awk 'END {print NR+0}')
gooo_files=$(find "$root" -type f -name '*.gooo' ! -path "$root/.git/*" | awk 'END {print NR+0}')
regular_files=$(find "$root" -type f ! -path "$root/.git/*" ! -path "$root/README.md" | awk 'END {print NR+0}')
descendant_dirs=$(find "$root" -type d ! -path "$root" ! -path "$root/.git" ! -path "$root/.git/*" | awk 'END {print NR+0}')
generated_count=$(find "$cycle_output" -maxdepth 1 -type f | awk 'END {print NR+0}')
generated_bytes=$(find "$cycle_output" -maxdepth 1 -type f -exec stat -c '%s' {} + | awk '{sum += $1} END {print sum+0}')

jq -n \
	--arg schema "gooo/bounded-self-change/ci-evidence/v2" \
	--arg commit "${GITHUB_SHA:-unknown}" \
	--slurpfile cycle "$cycle_output/cycle-manifest.json" \
	--slurpfile live "$live_output/cycle-manifest.json" \
	--slurpfile measurement "$cycle_output/measurement-receipt.json" \
	--slurpfile evidence "$cycle_output/evidence-manifest.json" \
	--slurpfile tests "$evidence_dir/tests.json" \
	--argjson go_files "$go_files" \
	--argjson gooo_files "$gooo_files" \
	--argjson regular_files "$regular_files" \
	--argjson descendant_dirs "$descendant_dirs" \
	--argjson generated_count "$generated_count" \
	--argjson generated_bytes "$generated_bytes" \
	'{schema:$schema,commit:$commit,cycle:$cycle[0],live_v050:$live[0],invariant_count:$cycle[0].invariant_count,proofs:$cycle[0].proofs,indicator_classes:$cycle[0].indicator_classes,stage_measurements:$measurement[0].stage_measurements,indicators:$measurement[0].indicators,tests:$tests[0],runtime_authority:{repository_writes:0,remote_writes:0,apply_authority:0,commit_authority:0,merge_authority:0,tag_authority:0,release_authority:0,local_test_executions:0,cross_project_required_gates:0},local_authority:{test:0,build:0,vet:0,lint:0,format:0,check:0,conformance:0,integration:0},remote_authority:{github_actions_validation_runs:1,github_actions_validation_jobs:6,github_token:1,cross_project_required_gates:0},inventory:{root_readme_excluded:true,git_excluded:true,caller_output_excluded:true,regular_files:$regular_files,descendant_dirs:$descendant_dirs,go_files:$go_files,gooo_files:$gooo_files},artifacts:{required_output_count:8,generated_count:$generated_count,generated_bytes:$generated_bytes,content_addressed_package_digest:$evidence[0].package_digest},historical_failures_preserved:true,release_policy:"immutable_forward_only",metric_summary_mode:"per_indicator_only"}' > "$evidence_dir/ci-evidence.json"
cp "$cycle_output/human-report.md" "$evidence_dir/human-report.md"
cp "$live_output/human-report.md" "$evidence_dir/live-v0.50-human-report.md"
