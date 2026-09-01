package compiler

import (
	"fmt"
	"path/filepath"
)

func Verify(metaPath, sourcePath, contractPath, outputDir, observationsPath string) (Report, error) {
	meta, err := ParseMeta(metaPath)
	if err != nil {
		return Report{}, err
	}
	source, err := ParseSource(sourcePath)
	if err != nil {
		return Report{}, err
	}
	contract, err := LoadContract(contractPath)
	if err != nil {
		return Report{}, err
	}
	ir, err := readIR(filepath.Join(outputDir, "semantic-ir.json"))
	if err != nil {
		return Report{}, err
	}
	if err := ValidateIR(ir); err != nil {
		return Report{}, err
	}
	if err := ValidateMeta(meta, contract); err != nil {
		return Report{}, err
	}
	if err := ValidateSource(source, contract); err != nil {
		return Report{}, err
	}
	contractDigest, err := ContractDigest(contract)
	if err != nil {
		return Report{}, err
	}
	if ir.SourceDigest != source.SourceDigest || ir.MetaDigest != meta.MetaDigest || ir.ContractDigest != contractDigest {
		return Report{}, fmt.Errorf("semantic IR provenance does not match declared inputs")
	}
	if err := validateGeneratedArtifacts(ir, outputDir); err != nil {
		return Report{}, err
	}
	execution, err := readExecution(observationsPath)
	if err != nil {
		return Report{}, err
	}
	if err := ValidateExecution(ir, execution); err != nil {
		return Report{}, err
	}
	cases, summary, err := evaluateCanonicalCases(ir, execution)
	if err != nil {
		return Report{}, err
	}
	coreClaim := evaluateCandidate(ir, execution)
	decision := "CANDIDATE_REJECTED"
	if coreClaim.State == "CLOSED" {
		decision = "CANDIDATE_ACCEPTABLE"
	}
	report := Report{
		Schema:                      ReportSchema,
		Decision:                    decision,
		FixedConformanceDenominator: FixedCaseCount,
		Precedence:                  append([]string(nil), ir.Precedence...),
		Scenario:                    ir.Scenario,
		SourceDigest:                ir.SourceDigest,
		MetaDigest:                  ir.MetaDigest,
		ContractDigest:              ir.ContractDigest,
		IRDigest:                    ir.IRDigest,
		CandidateDigest:             candidateDigest(ir, outputDir),
		CandidateRule:               ir.CandidateRule,
		Summary:                     summary,
		Cases:                       cases,
		Improvement:                 improvementClaim(coreClaim, execution.BeforeAfter),
		Utility: Claim{
			State: "UNKNOWN", Stage: "UTILITY", Step: "collect_external_user_evidence",
			Reason: "NO_EXTERNAL_USER_EVIDENCE_PROVIDED", UnknownClass: "MISSING_EXTERNAL_EVIDENCE",
			NextOperation: "COLLECT_EXTERNAL_USER_EVIDENCE", BlockedBy: []string{"external-user-evidence"},
		},
		ExactBeforeAfter: execution.BeforeAfter,
		RuntimeAuthority: Authority{
			RepositoryWrites:          0,
			ApplyAuthority:            0,
			CommitAuthority:           0,
			MergeAuthority:            0,
			TagAuthority:              0,
			ReleaseAuthority:          0,
			LocalTestExecutions:       0,
			CrossProjectRequiredGates: 0,
		},
		Evidence: map[string]string{
			"source_digest":   ir.SourceDigest,
			"meta_digest":     ir.MetaDigest,
			"contract_digest": ir.ContractDigest,
			"ir_digest":       ir.IRDigest,
			"candidate_digest": candidateDigest(ir, outputDir),
			"toolchain":       execution.Toolchain,
			"runner":          execution.Runner,
		},
	}
	if err := WriteJSON(filepath.Join(outputDir, "decision-dossier.json"), report); err != nil {
		return Report{}, err
	}
	if err := WriteText(filepath.Join(outputDir, "human-report.md"), RenderReport(report)); err != nil {
		return Report{}, err
	}
	return report, nil
}

func readIR(path string) (SemanticIR, error) {
	var ir SemanticIR
	if err := ReadJSON(path, &ir); err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func readExecution(path string) (ExecutionReport, error) {
	var execution ExecutionReport
	if err := ReadJSON(path, &execution); err != nil {
		return ExecutionReport{}, err
	}
	return execution, nil
}

func validateGeneratedArtifacts(ir SemanticIR, outputDir string) error {
	var candidate CandidateArtifact
	if err := ReadJSON(filepath.Join(outputDir, "candidate-artifact.json"), &candidate); err != nil {
		return err
	}
	expectedCandidate, err := candidateArtifact(ir)
	if err != nil {
		return err
	}
	if candidate != expectedCandidate {
		return fmt.Errorf("candidate artifact provenance or semantics mismatch")
	}
	var proposal PatchProposal
	if err := ReadJSON(filepath.Join(outputDir, "proposal.patch.json"), &proposal); err != nil {
		return err
	}
	if proposal.SourceDigest != ir.SourceDigest || proposal.CandidateDigest != candidate.CandidateDigest || proposal.RepositoryWrites != 0 || proposal.EditSurface != ir.EditSurface.Field {
		return fmt.Errorf("patch proposal violates the zero-write edit boundary")
	}
	return nil
}

func ValidateExecution(ir SemanticIR, execution ExecutionReport) error {
	if execution.Schema != ExecutionSchema || execution.Scenario != ir.Scenario || execution.SourceDigest != ir.SourceDigest || execution.MetaDigest != ir.MetaDigest || execution.ContractDigest != ir.ContractDigest || execution.IRDigest != ir.IRDigest || execution.Toolchain != ToolchainIdentity || execution.Runner != RunnerIdentity {
		return fmt.Errorf("execution evidence provenance is incomplete")
	}
	if len(execution.Observations) != 4 {
		return fmt.Errorf("execution evidence must contain exactly four observations")
	}
	checks := []struct {
		variant string
		input   int
		verdict string
	}{
		{variant: "before", input: 0, verdict: "reject"},
		{variant: "after", input: 0, verdict: "accept"},
		{variant: "after", input: -1, verdict: "reject"},
		{variant: "after", input: 1, verdict: "accept"},
	}
	for _, check := range checks {
		observation, ok := findObservation(execution.Observations, check.variant, check.input)
		if !ok || observation.Verdict != check.verdict || observation.Accepted != (check.verdict == "accept") {
			return fmt.Errorf("execution observation %s/%d does not satisfy the fixed fixture", check.variant, check.input)
		}
	}
	if execution.BeforeAfter != (ExactBeforeAfter{
		Scenario:       ir.Scenario,
		SourceDigest:   ir.SourceDigest,
		MetaDigest:     ir.MetaDigest,
		ContractDigest: ir.ContractDigest,
		Fixture:        "zero_rejected",
		Toolchain:      ToolchainIdentity,
		Runner:         RunnerIdentity,
		Metric:         "counterexample_acceptance",
		Before:         0,
		After:          1,
	}) {
		return fmt.Errorf("exact before/after integer pair is not fixed")
	}
	return nil
}

func evaluateCanonicalCases(ir SemanticIR, execution ExecutionReport) ([]CaseResult, CaseSummary, error) {
	results := make([]CaseResult, 0, len(ir.Cases))
	summary := CaseSummary{Total: len(ir.Cases)}
	for _, canonical := range ir.Cases {
		claim := evaluateFixture(ir, canonical, execution)
		state := resolveClaim(claim)
		if state != canonical.ExpectedState {
			return nil, CaseSummary{}, fmt.Errorf("canonical case %s resolved to %s, expected %s", canonical.ID, state, canonical.ExpectedState)
		}
		result := CaseResult{Ordinal: canonical.Ordinal, ID: canonical.ID, ExpectedState: canonical.ExpectedState, State: state, Probe: canonical.Probe, Fixture: canonical.Fixture, SemanticEdge: canonical.SemanticEdge, Claim: claim}
		results = append(results, result)
		switch state {
		case "CLOSED":
			summary.Closed++
		case "UNKNOWN":
			summary.Unknown++
		case "REFUTED":
			summary.Refuted++
		default:
			return nil, CaseSummary{}, fmt.Errorf("unknown canonical resolution %q", state)
		}
	}
	return results, summary, nil
}

func evaluateCandidate(ir SemanticIR, execution ExecutionReport) Claim {
	if ir.Authority != zeroAuthority() {
		return refutedClaim("AUTHORITY", "authority_boundary", "CANDIDATE_EXCEEDS_ZERO_WRITE_AUTHORITY")
	}
	if !allEvidenceComplete(ir.Evidence) {
		return unknownClaim("EVIDENCE", "verify_provenance", "REQUIRED_EVIDENCE_IS_INCOMPLETE", "MISSING_EVIDENCE", "RESTORE_REQUIRED_EVIDENCE", "evidence-obligations")
	}
	afterZero, _ := findObservation(execution.Observations, "after", 0)
	if afterZero.Verdict != "accept" {
		return refutedClaim("COUNTEREXAMPLE", "resolve_counterexample", "ORIGINAL_COUNTEREXAMPLE_PERSISTS")
	}
	afterNegative, _ := findObservation(execution.Observations, "after", -1)
	if afterNegative.Verdict != "reject" {
		return refutedClaim("GUARDRAIL", "verify_non_regression", "FIXED_GUARDRAIL_REGRESSED")
	}
	afterPositive, _ := findObservation(execution.Observations, "after", 1)
	if afterPositive.Verdict != "accept" {
		return refutedClaim("GUARDRAIL", "verify_positive_preservation", "POSITIVE_BEHAVIOR_REGRESSED")
	}
	return Claim{State: "CLOSED", Stage: "CANDIDATE", Step: "accept_verified_candidate", Reason: "COUNTEREXAMPLE_RESOLVED_GUARDRAILS_PRESERVED_PROVENANCE_COMPLETE", UnknownClass: "", NextOperation: "NONE", BlockedBy: []string{}}
}

func evaluateFixture(ir SemanticIR, canonical CanonicalCase, execution ExecutionReport) Claim {
	switch canonical.Fixture {
	case "counterexample_resolved":
		return evaluateCandidate(ir, execution)
	case "negative_guardrail_preserved":
		observation, ok := findObservation(execution.Observations, "after", -1)
		if !ok {
			return unknownClaim("GUARDRAIL", "observe_negative_guardrail", "GUARDRAIL_OBSERVATION_MISSING", "MISSING_EXECUTION_EVIDENCE", "COLLECT_NEGATIVE_GUARDRAIL_OBSERVATION", "after-negative")
		}
		if observation.Verdict != "reject" {
			return refutedClaim("GUARDRAIL", "observe_negative_guardrail", "FIXED_GUARDRAIL_REGRESSED")
		}
		return closedClaim("GUARDRAIL", "observe_negative_guardrail", "FIXED_NEGATIVE_GUARDRAIL_PRESERVED")
	case "positive_preserved":
		observation, ok := findObservation(execution.Observations, "after", 1)
		if !ok {
			return unknownClaim("GUARDRAIL", "observe_positive_behavior", "POSITIVE_OBSERVATION_MISSING", "MISSING_EXECUTION_EVIDENCE", "COLLECT_POSITIVE_OBSERVATION", "after-positive")
		}
		if observation.Verdict != "accept" {
			return refutedClaim("GUARDRAIL", "observe_positive_behavior", "POSITIVE_BEHAVIOR_REGRESSED")
		}
		return closedClaim("GUARDRAIL", "observe_positive_behavior", "POSITIVE_BEHAVIOR_PRESERVED")
	case "missing_counterexample":
		return unknownClaim("INPUT", "read_counterexample", "COUNTEREXAMPLE_DECLARATION_MISSING", "MISSING_INPUT_EVIDENCE", "DECLARE_OBSERVED_COUNTEREXAMPLE", "counterexample")
	case "unfixed_edit_surface":
		return unknownClaim("EDIT_SURFACE", "bound_permitted_edit", "PERMITTED_EDIT_SURFACE_NOT_FIXED", "MISSING_EDIT_BOUNDARY", "FIX_EDIT_SURFACE_BEFORE_GENERATION", "edit-surface")
	case "incomplete_evidence":
		return unknownClaim("EVIDENCE", "verify_obligations", "EVIDENCE_OBLIGATION_INCOMPLETE", "MISSING_EVIDENCE", "PROVIDE_ALL_EVIDENCE_OBLIGATIONS", "evidence-obligations")
	case "counterexample_persists":
		return refutedClaim("COUNTEREXAMPLE", "resolve_counterexample", "ORIGINAL_COUNTEREXAMPLE_PERSISTS")
	case "guardrail_regresses":
		return refutedClaim("GUARDRAIL", "verify_non_regression", "FIXED_GUARDRAIL_REGRESSED")
	case "authority_overreach":
		return refutedClaim("AUTHORITY", "verify_edit_authority", "CANDIDATE_REQUESTS_REPOSITORY_WRITE")
	default:
		return refutedClaim("FIXTURE", "resolve_fixture", "UNSUPPORTED_CANONICAL_FIXTURE")
	}
}

func resolveClaim(claim Claim) string {
	if claim.State == "REFUTED" {
		return "REFUTED"
	}
	if claim.State == "UNKNOWN" {
		if !claim.HasUnknownTuple() {
			return "REFUTED"
		}
		return "UNKNOWN"
	}
	if claim.State == "CLOSED" {
		return "CLOSED"
	}
	return "REFUTED"
}

func allEvidenceComplete(evidence []EvidenceDecl) bool {
	if len(evidence) == 0 {
		return false
	}
	for _, obligation := range evidence {
		if !obligation.Required || obligation.State != "complete" {
			return false
		}
	}
	return true
}

func findObservation(observations []ExecutionObservation, variant string, input int) (ExecutionObservation, bool) {
	for _, observation := range observations {
		if observation.Variant == variant && observation.Input == input {
			return observation, true
		}
	}
	return ExecutionObservation{}, false
}

func closedClaim(stage, step, reason string) Claim {
	return Claim{State: "CLOSED", Stage: stage, Step: step, Reason: reason, UnknownClass: "", NextOperation: "NONE", BlockedBy: []string{}}
}

func refutedClaim(stage, step, reason string) Claim {
	return Claim{State: "REFUTED", Stage: stage, Step: step, Reason: reason, UnknownClass: "", NextOperation: "REJECT_CANDIDATE", BlockedBy: []string{}}
}

func unknownClaim(stage, step, reason, unknownClass, nextOperation, blockedBy string) Claim {
	return Claim{State: "UNKNOWN", Stage: stage, Step: step, Reason: reason, UnknownClass: unknownClass, NextOperation: nextOperation, BlockedBy: []string{blockedBy}}
}

func improvementClaim(core Claim, pair ExactBeforeAfter) Claim {
	if core.State == "CLOSED" && pair.Before == 0 && pair.After == 1 {
		return Claim{State: "CLOSED", Stage: "IMPROVEMENT", Step: "compare_exact_before_after", Reason: "EXACT_BEFORE_AFTER_INTEGER_PAIR_PROVES_COUNTEREXAMPLE_REPAIR", UnknownClass: "", NextOperation: "NONE", BlockedBy: []string{}}
	}
	if core.State == "REFUTED" {
		return refutedClaim("IMPROVEMENT", "compare_exact_before_after", "CANDIDATE_PROOF_REFUTED")
	}
	return unknownClaim("IMPROVEMENT", "compare_exact_before_after", "EXACT_BEFORE_AFTER_INTEGER_PAIR_NOT_AVAILABLE", "MISSING_EXACT_PAIR", "PROVIDE_MATCHED_BEFORE_AFTER_PAIR", "scenario-source-contract-fixture-toolchain-runner")
}

func candidateDigest(ir SemanticIR, outputDir string) string {
	var candidate CandidateArtifact
	if err := ReadJSON(filepath.Join(outputDir, "candidate-artifact.json"), &candidate); err != nil {
		return ""
	}
	return candidate.CandidateDigest
}
