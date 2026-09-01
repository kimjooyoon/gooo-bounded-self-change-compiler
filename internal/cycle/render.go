package cycle

import (
	"fmt"
	"strings"
)

func renderReport(ir SemanticIR, decision string, cases []CaseResult, execution ExecutionInput, core Claim, mode string) string {
	var b strings.Builder
	b.WriteString("# Gooo deterministic bounded self-improvement cycle v0.2\n\n")
	fmt.Fprintf(&b, "Decision: `%s`\n\n", decision)
	fmt.Fprintf(&b, "Scenario: `%s`\n\n", ir.Scenario)
	fmt.Fprintf(&b, "Mode: `%s`\n\n", mode)
	b.WriteString("The semantic owner is the `.gooo` meta/source pair. Go generated only the evaluator and ephemeral candidate runtime.\n\n")
	b.WriteString("## Fixed 12-case judgment vector\n\n")
	b.WriteString("The denominator is exactly 12: 4 CLOSED, 4 UNKNOWN, 4 REFUTED. Resolution precedence is `REFUTED > UNKNOWN > CLOSED`; no score, percentage, average, or aggregate utility is emitted.\n\n")
	b.WriteString("| # | case | expected | state | semantic edge | claim |\n|---:|---|---|---|---|---|\n")
	for _, result := range cases {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n", result.Ordinal, result.ID, result.ExpectedState, result.State, result.SemanticEdge, result.Claim.Reason)
	}
	b.WriteString("\n## Cycle authority\n\n")
	b.WriteString("- repository writes: `0`\n- remote writes: `0`\n- apply/commit/merge/tag/release authority: `0`\n- local test executions: `0`\n- cross-project required gates: `0`\n- runtime local validation commands: `0`\n\n")
	b.WriteString("## Internal candidate and live boundary\n\n")
	if mode == "internal" {
		b.WriteString("The internal source exposes one explicit unblocked frontier, proposes one bounded threshold change, applies it only in caller-owned temporary space, binds exactly three semantic impacted tests, and carries the counterexample into the next wave without repository authority.\n\n")
		fmt.Fprintf(&b, "Exact before/after counterexample pair: `%d -> %d`.\n\n", execution.BeforeAfter["before"], execution.BeforeAfter["after"])
	} else {
		b.WriteString("The immutable ledger v0.50 observation exposes `EXTERNAL_UTILITY_EVIDENCE`. Automation cannot produce that evidence, so the live cycle stops at `UNKNOWN / HUMAN_EXTERNAL_EVIDENCE_REQUIRED`; no source change or fabricated proposal is emitted.\n\n")
	}
	fmt.Fprintf(&b, "Core judgment: `%s` — %s\n\n", core.State, core.Reason)
	b.WriteString("## Provenance\n\n")
	fmt.Fprintf(&b, "- source digest: `%s`\n- meta digest: `%s`\n- contract digest: `%s`\n- semantic IR digest: `%s`\n- toolchain: `%s`\n- runner: `%s`\n\n", ir.SourceDigest, ir.MetaDigest, ir.ContractDigest, ir.IRDigest, execution.Toolchain, execution.Runner)
	b.WriteString("All final artifacts are caller-owned and content-addressed. Historical failed runs, tags, drafts, and releases remain preserved; this cycle is forward-only.\n")
	return b.String()
}
