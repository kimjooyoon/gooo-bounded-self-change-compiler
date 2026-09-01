#!/usr/bin/env bash
set -euo pipefail

bin=${1:?path to the compiler binary is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${CONFORMANCE_WORK_ROOT:?CONFORMANCE_WORK_ROOT is required}
counts=${CONFORMANCE_COUNTS_OUT:?CONFORMANCE_COUNTS_OUT is required}
source="$root/examples/bounded-self-change-v1/self-change.gooo"
meta="$root/.gooo/bounded-self-change-compiler.gooo"
contract="$root/contracts/denominator-v1.json"

mkdir -p "$work"
before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')

run_case() {
	label=$1
	out="$work/$label"
	mkdir -p "$out"
	"$bin" run --meta "$meta" --source "$source" --contract "$contract" --out "$out" >/dev/null
	go build -trimpath -o "$work/candidate-$label" "$out/candidate.go"
	"$work/candidate-$label" before 0 > "$out/before-counterexample.json"
	"$work/candidate-$label" after 0 > "$out/after-counterexample.json"
	"$work/candidate-$label" after -1 > "$out/after-negative.json"
	"$work/candidate-$label" after 1 > "$out/after-positive.json"
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
}

run_case first
run_case replay

for artifact in semantic-ir.json semantic-graph.json candidate-artifact.json candidate.gooo candidate.go proposal.patch.json verification-plan.json generation-receipt.json execution-evidence.json decision-dossier.json human-report.md; do
	cmp -s "$work/first/$artifact" "$work/replay/$artifact"
done

jq -e '
	.decision == "CANDIDATE_ACCEPTABLE" and
	.fixed_conformance_denominator == 9 and
	.summary == {total:9,closed:3,unknown:3,refuted:3} and
	.precedence == ["REFUTED","UNKNOWN","CLOSED"] and
	.improvement.state == "CLOSED" and
	.utility.state == "UNKNOWN" and
	.exact_before_after.before == 0 and .exact_before_after.after == 1 and
	.runtime_authority.repository_writes == 0 and
	.runtime_authority.apply_authority == 0 and
	.runtime_authority.commit_authority == 0 and
	.runtime_authority.merge_authority == 0 and
	.runtime_authority.tag_authority == 0 and
	.runtime_authority.release_authority == 0 and
	.runtime_authority.local_test_executions == 0 and
	.runtime_authority.cross_project_required_gates == 0 and
	([.cases[] | select(.state == "UNKNOWN") | (.claim.stage != "" and .claim.step != "" and .claim.reason != "" and .claim.unknown_class != "" and .claim.next_operation != "" and (.claim.blocked_by | length) > 0)] | all) and
	([.cases[] | select(.state == "CLOSED")] | length) == 3 and
	([.cases[] | select(.state == "UNKNOWN")] | length) == 3 and
	([.cases[] | select(.state == "REFUTED")] | length) == 3 and
	(has("score") | not) and (has("percentage") | not)
' "$work/first/decision-dossier.json" >/dev/null

for artifact in semantic-ir.json semantic-graph.json candidate-artifact.json candidate.gooo candidate.go proposal.patch.json verification-plan.json generation-receipt.json decision-dossier.json human-report.md; do
	test -f "$work/first/$artifact"
done

forbidden="$root/.gooo-bounded-self-change-forbidden-output"
if "$bin" run --meta "$meta" --source "$source" --contract "$contract" --out "$forbidden"; then
	echo "repository-owned output was accepted" >&2
	exit 1
fi
test ! -e "$forbidden"

after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"

jq -n \
	--arg schema "gooo/bounded-self-change/conformance/v1" \
	--slurpfile report "$work/first/decision-dossier.json" \
	'{schema:$schema,total:$report[0].fixed_conformance_denominator,selected:$report[0].fixed_conformance_denominator,executed:$report[0].fixed_conformance_denominator,reused:0,closed:$report[0].summary.closed,unknown:$report[0].summary.unknown,refuted:$report[0].summary.refuted}' > "$counts"
jq -e '.total == 9 and .selected == 9 and .executed == 9 and .reused == 0 and .closed == 3 and .unknown == 3 and .refuted == 3' "$counts" >/dev/null
