package compiler

import (
	"fmt"
	"strings"
)

func RenderReport(report Report) string {
	var b strings.Builder
	b.WriteString("# Gooo bounded self-change decision dossier\n\n")
	fmt.Fprintf(&b, "Decision: `%s`\n\n", report.Decision)
	fmt.Fprintf(&b, "Scenario: `%s`\n\n", report.Scenario)
	fmt.Fprintf(&b, "Fixed semantic denominator: %d canonical cases\n\n", report.FixedConformanceDenominator)
	b.WriteString("Resolution precedence: `REFUTED > UNKNOWN > CLOSED`\n\n")
	fmt.Fprintf(&b, "- CLOSED: %d\n- UNKNOWN: %d\n- REFUTED: %d\n\n", report.Summary.Closed, report.Summary.Unknown, report.Summary.Refuted)
	b.WriteString("## Exact before/after evidence\n\n")
	fmt.Fprintf(&b, "- fixture: `%s`\n- metric: `%s`\n- before: `%d`\n- after: `%d`\n- toolchain: `%s`\n- runner: `%s`\n\n", report.ExactBeforeAfter.Fixture, report.ExactBeforeAfter.Metric, report.ExactBeforeAfter.Before, report.ExactBeforeAfter.After, report.ExactBeforeAfter.Toolchain, report.ExactBeforeAfter.Runner)
	fmt.Fprintf(&b, "Improvement claim: `%s` — %s\n\n", report.Improvement.State, report.Improvement.Reason)
	b.WriteString("## Canonical cases\n\n")
	b.WriteString("| ordinal | case | expected | actual | fixture | semantic edge | reason |\n")
	b.WriteString("|---:|---|---|---|---|---|---|\n")
	for _, result := range report.Cases {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s |\n", result.Ordinal, result.ID, result.ExpectedState, result.State, result.Fixture, result.SemanticEdge, result.Claim.Reason)
	}
	b.WriteString("\n## Provenance and authority\n\n")
	fmt.Fprintf(&b, "- source digest: `%s`\n- meta digest: `%s`\n- contract digest: `%s`\n- semantic IR digest: `%s`\n- candidate digest: `%s`\n- runtime repository writes: `%d`\n- runtime apply authority: `%d`\n- runtime commit authority: `%d`\n- runtime merge authority: `%d`\n- runtime tag authority: `%d`\n- runtime release authority: `%d`\n- local test executions: `%d`\n- cross-project required gates: `%d`\n\n", report.SourceDigest, report.MetaDigest, report.ContractDigest, report.IRDigest, report.CandidateDigest, report.RuntimeAuthority.RepositoryWrites, report.RuntimeAuthority.ApplyAuthority, report.RuntimeAuthority.CommitAuthority, report.RuntimeAuthority.MergeAuthority, report.RuntimeAuthority.TagAuthority, report.RuntimeAuthority.ReleaseAuthority, report.RuntimeAuthority.LocalTestExecutions, report.RuntimeAuthority.CrossProjectRequiredGates)
	fmt.Fprintf(&b, "Utility claim: `%s` — %s\n\n", report.Utility.State, report.Utility.Reason)
	b.WriteString("The candidate is a proposal only. Applying, committing, merging, tagging, and releasing remain outside the runtime.\n")
	return b.String()
}
