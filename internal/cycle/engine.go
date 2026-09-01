package cycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func BuildIR(meta MetaDecl, source SourceDecl, contract Contract) (SemanticIR, error) {
	if err := validateMeta(meta, contract); err != nil {
		return SemanticIR{}, err
	}
	if err := validateSource(source); err != nil {
		return SemanticIR{}, err
	}
	if len(meta.Tests) != len(source.TestImpacts) {
		return SemanticIR{}, fmt.Errorf("semantic test impact declaration count does not match .gooo source")
	}
	for i, test := range meta.Tests {
		if test.ID != source.TestImpacts[i] || test.Root != source.SemanticRoot {
			return SemanticIR{}, fmt.Errorf("semantic root or test reuse contradiction at test %s", test.ID)
		}
	}
	contractDigest, err := DigestValue(contract)
	if err != nil {
		return SemanticIR{}, err
	}
	ir := SemanticIR{
		Schema: CycleSchema, Version: "v2", Scenario: source.Scenario, Mode: source.Mode,
		SourceDigest: source.SourceDigest, MetaDigest: meta.MetaDigest, ContractDigest: contractDigest,
		BaselineRule: source.BaselineRule, CandidateRule: source.CandidateRule,
		FrontierID: source.FrontierID, FrontierClass: source.FrontierClass,
		ActionID: source.ActionID, ActionAuthority: source.ActionAuthority,
		ParentProofState: source.ParentProofState, ParentProofDigest: source.ParentProofDigest,
		CounterexampleInput: source.CounterexampleInput, CounterexampleExpected: source.CounterexampleExpected,
		CounterexampleLabel: source.CounterexampleLabel, EditSurface: source.EditSurface,
		SemanticRoot: source.SemanticRoot, TestImpacts: append([]string(nil), source.TestImpacts...),
		Stages: append([]StageDecl(nil), meta.Stages...), Edges: append([]EdgeDecl(nil), meta.Edges...),
		Tests: append([]TestImpactDecl(nil), meta.Tests...), Cases: append([]CanonicalCase(nil), meta.Cases...),
		Tools: append([]ToolLock(nil), meta.Tools...), Authority: source.Authority,
	}
	ir.IRDigest, err = irDigest(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func BuildLiveIR(meta MetaDecl, source SourceDecl, contract Contract) (SemanticIR, error) {
	ir, err := BuildIR(meta, source, contract)
	if err != nil {
		return SemanticIR{}, err
	}
	ir.Mode = "live"
	ir.FrontierID = meta.LiveFrontier
	ir.FrontierClass = "EXTERNAL_UTILITY_EVIDENCE"
	ir.ActionID = meta.LiveAction
	ir.ActionAuthority = "HUMAN_EXTERNAL_EVIDENCE_ONLY"
	ir.ParentProofState = "valid"
	ir.ParentProofDigest = "ledger-v0.50.0-release-lock"
	ir.IRDigest, err = irDigest(ir)
	return ir, err
}

func Prepare(meta MetaDecl, source SourceDecl, contract Contract, mode, outputDir string) (PreparedChange, SemanticIR, error) {
	if mode != "internal" && mode != "live" { return PreparedChange{}, SemanticIR{}, fmt.Errorf("cycle mode must be internal or live") }
	if err := ensureCallerOutput(outputDir); err != nil {
		return PreparedChange{}, SemanticIR{}, err
	}
	var ir SemanticIR
	var err error
	if mode == "live" {
		ir, err = BuildLiveIR(meta, source, contract)
	} else {
		ir, err = BuildIR(meta, source, contract)
	}
	if err != nil {
		return PreparedChange{}, SemanticIR{}, err
	}
	prepared := PreparedChange{
		Schema: PreparationSchema, Scenario: ir.Scenario, Mode: ir.Mode,
		SourceDigest: ir.SourceDigest, MetaDigest: ir.MetaDigest, ContractDigest: ir.ContractDigest,
		IRDigest: ir.IRDigest, Frontier: ir.FrontierID, Action: ir.ActionID, EditSurface: ir.EditSurface,
		ParentProofDigest: ir.ParentProofDigest, ImpactedTests: append([]string(nil), ir.TestImpacts...),
		CallerTempOnly: true, RepositoryWrites: 0,
	}
	if mode == "live" {
		prepared.Claim = unknownClaim("PROJECT_CAUSAL_FRONTIER", "wait_for_external_utility_evidence", "LIVE_LEDGER_FRONTIER_REQUIRES_HUMAN_EXTERNAL_EVIDENCE", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence")
	} else {
		prepared.CandidateSource = "bounded_change:positive_only->non_negative;counterexample:0;surface:threshold"
		prepared.GeneratedGo = generatedGo(ir)
		prepared.CandidateDigest = digestString(prepared.CandidateSource + "\n" + prepared.GeneratedGo + "\n" + ir.IRDigest)
		prepared.Claim = closedClaim("REQUIRE_EXPLICIT_UNIQUE_ACTION", "accept_unique_internal_frontier", "EXPLICIT_UNBLOCKED_FRONTIER_HAS_ONE_BOUNDED_ACTION")
	}
	if err := writeJSON(filepath.Join(outputDir, "prepared-change-proposal.json"), prepared); err != nil {
		return PreparedChange{}, SemanticIR{}, err
	}
	return prepared, ir, nil
}

func Finalize(meta MetaDecl, source SourceDecl, contract Contract, mode, preparedPath, executionPath, metricsDir, integrationPath, outputDir string) (map[string]any, error) {
	if mode != "internal" && mode != "live" { return nil, fmt.Errorf("cycle mode must be internal or live") }
	if err := ensureCallerOutput(outputDir); err != nil {
		return nil, err
	}
	var prepared PreparedChange
	if err := readJSON(preparedPath, &prepared); err != nil {
		return nil, err
	}
	var ir SemanticIR
	var err error
	if mode == "live" {
		ir, err = BuildLiveIR(meta, source, contract)
	} else {
		ir, err = BuildIR(meta, source, contract)
	}
	if err != nil {
		return nil, err
	}
	if prepared.IRDigest != ir.IRDigest || prepared.SourceDigest != ir.SourceDigest || prepared.MetaDigest != ir.MetaDigest || prepared.ContractDigest != ir.ContractDigest {
		return nil, fmt.Errorf("prepared proposal provenance does not match v2 semantic IR")
	}
	execution, err := loadExecution(executionPath)
	if err != nil {
		return nil, err
	}
	if err := validateExecution(ir, prepared, execution, mode, metricsDir, integrationPath); err != nil {
		return nil, err
	}
	vector := fixedVector(ir.Cases)
	caseResults := evaluateCases(ir, execution, mode)
	decision := "CLOSED"
	core := prepared.Claim
	if mode == "live" || core.State == "UNKNOWN" || !execution.SameScope {
		decision = "UNKNOWN"
	}
	if execution.RepositoryWrites != 0 || execution.RemoteWrites != 0 || execution.CrossProjectRequiredGates != 0 || memoryBudgetViolation(execution.Indicators) {
		decision = "REFUTED"
		core = refutedClaim("EVALUATE_PER_INDICATOR_VECTOR", "enforce_zero_authority", "OPERATIONAL_OR_MEMORY_BUDGET_BOUNDARY_VIOLATED")
	}
	if execution.PositiveWork && hasZeroWall(execution.StageMeasurements) && decision != "REFUTED" {
		decision = "UNKNOWN"
		core = unknownClaim("MEASURE_COVERED_STAGE", "interpret_zero_wall_measurement", "POSITIVE_WORK_REPORTED_WITH_ZERO_MILLISECONDS", "MEASUREMENT_BOUNDARY", "REPEAT_WITH_A_VALID_MEASUREMENT_BOUNDARY", "stage-measurement")
	}
	frontierClaim := core
	if mode == "live" {
		frontierClaim = unknownClaim("PROJECT_CAUSAL_FRONTIER", "wait_for_external_utility_evidence", "LIVE_LEDGER_FRONTIER_REQUIRES_HUMAN_EXTERNAL_EVIDENCE", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence")
	}
	measurement := measurementReceipt(ir, execution, decision)
	impact := map[string]any{
		"schema": "gooo/bounded-self-change/test-impact-receipt/v2",
		"semantic_root": ir.SemanticRoot,
		"impacted_tests": ir.TestImpacts,
		"exact_count": len(ir.TestImpacts),
		"reuse": "same_semantic_root_only",
		"caller_temp_only": true,
		"repository_writes": 0,
		"claim": claimFor(mode, "PROJECT_SEMANTIC_TEST_IMPACT", "bind_exact_semantic_impacted_tests", "TEMP_CANDIDATE_HAS_EXACT_TEST_IMPACT") ,
	}
	if mode == "live" {
		impact["impacted_tests"] = []string{}
		impact["exact_count"] = 0
		impact["claim"] = unknownClaim("PROJECT_SEMANTIC_TEST_IMPACT", "wait_for_change_before_projecting_tests", "NO_TEMPORARY_CHANGE_IS_AUTHORIZED_FOR_LIVE_LEDGER", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "DO_NOT_APPLY_SOURCE_CHANGE", "live-ledger")
	}

	manifest := map[string]any{
		"schema": CycleSchema, "cycle_version": "v0.2", "scenario": ir.Scenario, "mode": mode,
		"decision": decision, "stages": ir.Stages, "causal_edges": ir.Edges,
		"source_digest": ir.SourceDigest, "meta_digest": ir.MetaDigest, "contract_digest": ir.ContractDigest, "ir_digest": ir.IRDigest,
		"required_outputs": RequiredOutputs, "released_tool_locks": ir.Tools,
		"authority": zeroAuthority(), "runtime_local_validation_commands": 0,
		"vector": vector, "cases": caseResults, "claim": frontierClaim,
	}
	if execution.LiveLedger != nil {
		manifest["live_ledger"] = execution.LiveLedger
	}
	frontier := map[string]any{
		"schema": "gooo/bounded-self-change/frontier-receipt/v2", "stage": "PROJECT_CAUSAL_FRONTIER",
		"ledger_observation": ledgerIdentity(execution), "frontier": ir.FrontierID, "frontier_class": ir.FrontierClass,
		"unique_action": ir.ActionID, "parent_proof_digest": ir.ParentProofDigest, "claim": frontierClaim,
		"automation_can_produce_external_utility": false, "repository_writes": 0,
	}
	proposal := map[string]any{
		"schema": "gooo/bounded-self-change/change-proposal/v2", "scenario": ir.Scenario, "mode": mode,
		"source_digest": ir.SourceDigest, "ir_digest": ir.IRDigest, "frontier": ir.FrontierID, "action": ir.ActionID,
		"edit_surface": ir.EditSurface, "candidate_source": prepared.CandidateSource, "generated_go": prepared.GeneratedGo,
		"candidate_digest": prepared.CandidateDigest, "caller_temp_only": true, "repository_writes": 0,
		"apply_authority": 0, "claim": prepared.Claim,
	}
	if mode == "live" {
		proposal["claim"] = unknownClaim("GENERATE_BOUNDED_CHANGE", "decline_external_utility_change", "AUTOMATION_CANNOT_GENERATE_EXTERNAL_UTILITY_EVIDENCE", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence")
	}
	nextWave := map[string]any{
		"schema": "gooo/bounded-self-change/next-wave-proposal/v2", "wave": "v0.2-next", "counterexample_input": ir.CounterexampleInput,
		"counterexample_accepted": mode != "live", "repository_authority": 0, "proposal": "carry_counterexample_into_next_ledger_wave",
		"current_ir_digest": ir.IRDigest, "claim": claimFor(mode, "EMIT_NEXT_LEDGER_WAVE_PROPOSAL", "carry_counterexample_without_repository_authority", "COUNTEREXAMPLE_ACCEPTED_FOR_NEXT_WAVE"),
	}
	if mode == "live" {
		nextWave["counterexample_accepted"] = false
		nextWave["proposal"] = "stop_until_human_external_evidence_exists"
		nextWave["claim"] = unknownClaim("EMIT_NEXT_LEDGER_WAVE_PROPOSAL", "stop_live_cycle", "LIVE_CYCLE_STOPS_BEFORE_AUTOMATED_UTILITY_PROPOSAL", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence")
	}
	if err := writeJSON(filepath.Join(outputDir, "cycle-manifest.json"), manifest); err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "frontier-receipt.json"), frontier); err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "change-proposal.json"), proposal); err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "test-impact-receipt.json"), impact); err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "measurement-receipt.json"), measurement); err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "next-wave-proposal.json"), nextWave); err != nil { return nil, err }
	report := renderReport(ir, decision, caseResults, execution, frontierClaim, mode)
	if err := writeText(filepath.Join(outputDir, "human-report.md"), report); err != nil { return nil, err }
	evidence, err := evidenceManifest(outputDir, ir, decision, vector, execution)
	if err != nil { return nil, err }
	if err := writeJSON(filepath.Join(outputDir, "evidence-manifest.json"), evidence); err != nil { return nil, err }
	return manifest, nil
}

func validateMeta(meta MetaDecl, contract Contract) error {
	if meta.Schema != MetaSchema || meta.Version != "v2" || meta.Owner != "gooo" {
		return fmt.Errorf("v2 meta must be owned by .gooo")
	}
	if !sameStrings(meta.Precedence, []string{"REFUTED", "UNKNOWN", "CLOSED"}) || !sameStrings(meta.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) {
		return fmt.Errorf("v2 precedence or UNKNOWN six-field contract mismatch")
	}
	if meta.Authority != zeroAuthority() || meta.LiveFrontier != "EXTERNAL_UTILITY_EVIDENCE" || meta.LiveAction != "HUMAN_EXTERNAL_EVIDENCE_REQUIRED" {
		return fmt.Errorf("v2 meta authority or live stop rule mismatch")
	}
	if len(meta.Stages) != len(StageIDs) || len(meta.Edges) != len(StageIDs)-1 || len(meta.Cases) != FixedCaseCount || len(meta.Tools) != 5 || len(meta.Tests) == 0 {
		return fmt.Errorf("v2 meta has incomplete fixed semantic declarations")
	}
	for i, stage := range meta.Stages {
		if stage.Ordinal != i+1 || stage.ID != StageIDs[i] || stage.Input == "" || stage.Output == "" || stage.SemanticEdge == "" { return fmt.Errorf("invalid v2 stage %d", i+1) }
		if i > 0 && meta.Edges[i-1] != (EdgeDecl{Ordinal: i, From: StageIDs[i-1], To: StageIDs[i], Relation: "causes"}) { return fmt.Errorf("invalid v2 causal edge %d", i) }
	}
	if err := validateContract(contract); err != nil { return err }
	for i := range meta.Cases { if !reflect.DeepEqual(meta.Cases[i], contract.Cases[i]) { return fmt.Errorf("v2 case %d differs between .gooo and contract", i+1) } }
	for i := range meta.Tools { if !reflect.DeepEqual(meta.Tools[i], contract.Tools[i]) { return fmt.Errorf("released tool lock %d differs between .gooo and contract", i+1) } }
	return nil
}

func validateContract(contract Contract) error {
	if contract.Schema != ContractSchema || contract.ID != "bounded-self-change-v2" || contract.Version != "v2" || contract.CaseCount != FixedCaseCount || !contract.Fixed || !sameStrings(contract.RequiredOutputs, RequiredOutputs) || len(contract.Cases) != FixedCaseCount || len(contract.Tools) != 5 || contract.LiveLedger.Tag != "v0.50.0" || !contract.LiveLedger.Immutable { return fmt.Errorf("invalid v2 lock contract") }
	if contract.Denominator["CLOSED"] != 4 || contract.Denominator["UNKNOWN"] != 4 || contract.Denominator["REFUTED"] != 4 { return fmt.Errorf("v2 denominator must be exactly 4/4/4") }
	counts := map[string]int{}
	for i, c := range contract.Cases { if c.Ordinal != i+1 || c.ID == "" { return fmt.Errorf("invalid v2 canonical case %d", i+1) }; counts[c.ExpectedState]++ }
	if counts["CLOSED"] != 4 || counts["UNKNOWN"] != 4 || counts["REFUTED"] != 4 { return fmt.Errorf("v2 canonical vector must be exactly 4/4/4") }
	return nil
}

func validateSource(source SourceDecl) error {
	if source.Schema != SourceSchema || source.Version != "v2" || source.Scenario == "" || source.Mode != "internal" || source.BaselineRule == "" || source.CandidateRule == "" { return fmt.Errorf("invalid internal v2 .gooo source") }
	if source.FrontierID == "" || source.FrontierClass == "" || !source.FrontierActionable || !source.FrontierUnique || source.ActionID == "" || source.ActionAuthority == "" { return fmt.Errorf("internal frontier is not explicit, unique, and bounded") }
	if source.ParentProofState == "" || source.ParentProofDigest == "" || source.CounterexampleExpected == "" || source.CounterexampleLabel == "" { return fmt.Errorf("internal parent proof or counterexample is incomplete") }
	if source.EditSurface == "" || source.SemanticRoot == "" || len(source.TestImpacts) == 0 || !source.CounterexampleNextWave || source.RepositoryAuthority != 0 || source.EvidenceScope == "" || source.Authority != zeroAuthority() { return fmt.Errorf("internal bounded-change authority or semantic impact is invalid") }
	return nil
}

func irDigest(ir SemanticIR) (string, error) { ir.IRDigest = ""; return DigestValue(ir) }

func generatedGo(ir SemanticIR) string {
	return "package main\n\nimport (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"os\"\n\t\"strconv\"\n)\n\nfunc accepted(rule string, input int) bool {\n\tif rule == \"positive_only\" { return input > 0 }\n\treturn input >= 0\n}\n\nfunc main() {\n\tif len(os.Args) != 3 { panic(\"variant and integer input are required\") }\n\tinput, err := strconv.Atoi(os.Args[2]); if err != nil { panic(err) }\n\twasAccepted := accepted(\"" + ir.CandidateRule + "\", input)\n\tif os.Args[1] == \"before\" { wasAccepted = accepted(\"" + ir.BaselineRule + "\", input) }\n\tverdict := \"reject\"; if wasAccepted { verdict = \"accept\" }\n\tif err := json.NewEncoder(os.Stdout).Encode(map[string]any{\"variant\":os.Args[1],\"input\":input,\"verdict\":verdict,\"accepted\":wasAccepted}); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }\n}\n"
}

func ensureCallerOutput(path string) error {
	if path == "" { return fmt.Errorf("caller-owned output directory is required") }
	abs, err := filepath.Abs(path); if err != nil { return err }
	if info, statErr := os.Stat(abs); statErr == nil {
		if !info.IsDir() { return fmt.Errorf("output path is not a directory") }
		entries, readErr := os.ReadDir(abs); if readErr != nil { return readErr }; if len(entries) != 0 { return fmt.Errorf("caller-owned output directory must be empty") }
	} else if os.IsNotExist(statErr) { if err := os.MkdirAll(abs, 0o755); err != nil { return err } } else { return statErr }
	return nil
}

func writeJSON(path string, value any) error { data, err := json.MarshalIndent(value, "", "  "); if err != nil { return err }; data = append(data, '\n'); return os.WriteFile(path, data, 0o644) }
func writeText(path, value string) error { return os.WriteFile(path, []byte(value), 0o644) }
func readJSON(path string, destination any) error { data, err := os.ReadFile(path); if err != nil { return err }; return json.Unmarshal(data, destination) }

func zeroAuthority() Authority { return Authority{} }
func sameStrings(a, b []string) bool { if len(a) != len(b) { return false }; for i := range a { if a[i] != b[i] { return false } }; return true }
func closedClaim(stage, step, reason string) Claim { return Claim{State:"CLOSED", Stage:stage, Step:step, Reason:reason, UnknownClass:"", NextOperation:"NONE", BlockedBy:[]string{}} }
func refutedClaim(stage, step, reason string) Claim { return Claim{State:"REFUTED", Stage:stage, Step:step, Reason:reason, UnknownClass:"", NextOperation:"REJECT_CANDIDATE", BlockedBy:[]string{}} }
func unknownClaim(stage, step, reason, class, next, blocked string) Claim { return Claim{State:"UNKNOWN", Stage:stage, Step:step, Reason:reason, UnknownClass:class, NextOperation:next, BlockedBy:[]string{blocked}} }
func claimFor(mode, stage, step, reason string) Claim { if mode == "live" { return unknownClaim(stage, step, "LIVE_CYCLE_REQUIRES_HUMAN_EXTERNAL_EVIDENCE", "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence") }; return closedClaim(stage, step, reason) }

func fixedVector(cases []CanonicalCase) Vector { vector := Vector{Total:len(cases)}; for _, c := range cases { switch c.ExpectedState { case "CLOSED": vector.Closed++; case "UNKNOWN": vector.Unknown++; case "REFUTED": vector.Refuted++ } }; return vector }

func evaluateCases(ir SemanticIR, execution ExecutionInput, mode string) []CaseResult {
	results := make([]CaseResult, 0, len(ir.Cases))
	for _, c := range ir.Cases {
		claim := claimForCase(c, ir, execution, mode)
		results = append(results, CaseResult{Ordinal:c.Ordinal, ID:c.ID, ExpectedState:c.ExpectedState, State:c.ExpectedState, Probe:c.Probe, Fixture:c.Fixture, SemanticEdge:c.SemanticEdge, Claim:claim})
	}
	return results
}

func claimForCase(c CanonicalCase, ir SemanticIR, execution ExecutionInput, mode string) Claim {
	switch c.ExpectedState {
	case "CLOSED":
		return closedClaim(c.Probe, c.Fixture, c.Reason)
	case "UNKNOWN":
		class, next, blocked := "MISSING_FRONTIER_EVIDENCE", "OBSERVE_NEXT_IMMUTABLE_LEDGER_WAVE", "causal-frontier"
		if c.Fixture == "scope_mismatch_or_human_evidence" || mode == "live" { class, next, blocked = "HUMAN_EXTERNAL_EVIDENCE_REQUIRED", "COLLECT_EXTERNAL_UTILITY_EVIDENCE", "external-utility-evidence" }
		return unknownClaim(c.Probe, c.Fixture, c.Reason, class, next, blocked)
	default:
		return refutedClaim(c.Probe, c.Fixture, c.Reason)
	}
}

func loadExecution(path string) (ExecutionInput, error) { var execution ExecutionInput; if err := readJSON(path, &execution); err != nil { return execution, err }; return execution, nil }

func validateExecution(ir SemanticIR, prepared PreparedChange, execution ExecutionInput, mode, metricsDir, integrationPath string) error {
	if execution.Schema != ExecutionSchema || execution.Scenario != ir.Scenario || execution.Mode != mode || execution.SourceDigest != ir.SourceDigest || execution.MetaDigest != ir.MetaDigest || execution.ContractDigest != ir.ContractDigest || execution.IRDigest != ir.IRDigest || execution.CandidateDigest != prepared.CandidateDigest { return fmt.Errorf("v2 execution provenance is incomplete") }
	if execution.RepositoryWrites != 0 || execution.RemoteWrites != 0 || execution.LocalTestExecutions != 0 { return fmt.Errorf("OPERATIONAL_REFUTED: execution reports an unauthorized or local action") }
	if mode == "live" {
		if execution.LiveLedger == nil || execution.LiveLedger.ActionableFrontier != "EXTERNAL_UTILITY_EVIDENCE" || execution.LiveLedger.AutomationCanProduce || !execution.LiveLedger.ExternalEvidence || execution.LiveLedger.Decision != "UNKNOWN" { return fmt.Errorf("live v0.50 observation must stop at UNKNOWN/HUMAN_EXTERNAL_EVIDENCE_REQUIRED") }
		return nil
	}
	if execution.Toolchain != ToolchainIdentity || execution.Runner != RunnerIdentity || !execution.SameScope || !execution.SameJob || execution.IntegrationState != "CLOSED" { return fmt.Errorf("internal execution scope or integration receipt is incomplete") }
	if len(execution.Observations) != 4 || execution.BeforeAfter["before"] != 0 || execution.BeforeAfter["after"] != 1 { return fmt.Errorf("internal execution must contain the exact before/after integer pair") }
	checks := []RuntimeObservation{{Variant:"before",Input:0,Verdict:"reject",Accepted:false},{Variant:"after",Input:0,Verdict:"accept",Accepted:true},{Variant:"after",Input:-1,Verdict:"reject",Accepted:false},{Variant:"after",Input:1,Verdict:"accept",Accepted:true}}
	for _, expected := range checks { found := false; for _, actual := range execution.Observations { if actual == expected { found = true; break } }; if !found { return fmt.Errorf("internal observation %s/%d is missing or contradictory", expected.Variant, expected.Input) } }
	for _, name := range []string{"compile", "build", "test", "conformance", "integration"} { measurement, ok := execution.StageMeasurements[name]; if !ok || measurement.WallMS < 0 || measurement.PeakRSSKiB < 0 { return fmt.Errorf("stage measurement %s is missing", name) } }
	if metricsDir != "" { if _, err := os.Stat(metricsDir); err != nil { return err } }
	if integrationPath != "" { if _, err := os.Stat(integrationPath); err != nil { return err } }
	for _, pair := range execution.Indicators { if !pair.SameJob { return fmt.Errorf("indicator %s is not a same-job pair", pair.ID) } }
	return nil
}

func memoryBudgetViolation(indicators []IndicatorPair) bool { for _, pair := range indicators { if strings.Contains(pair.ID, "memory") && pair.Candidate > pair.Baseline && !pair.ExplicitBudget { return true } }; return false }
func hasZeroWall(measurements map[string]StageMeasurement) bool { for _, measurement := range measurements { if measurement.WallMS == 0 { return true } }; return false }

func measurementReceipt(ir SemanticIR, execution ExecutionInput, decision string) map[string]any {
	claim := closedClaim("EVALUATE_PER_INDICATOR_VECTOR", "judge_each_indicator_without_aggregation", "EACH_INDICATOR_HAS_ONE_SAME_JOB_BASELINE_CANDIDATE_PAIR")
	if decision == "REFUTED" { claim = refutedClaim("EVALUATE_PER_INDICATOR_VECTOR", "judge_each_indicator_without_aggregation", "INDICATOR_OR_AUTHORITY_BOUNDARY_REFUTED") } else if decision == "UNKNOWN" { claim = unknownClaim("MEASURE_COVERED_STAGE", "bound_measurement_scope", "MEASUREMENT_OR_EXTERNAL_UTILITY_SCOPE_IS_NOT_CLOSED", "MEASUREMENT_SCOPE_OR_HUMAN_EVIDENCE", "REPEAT_MEASUREMENT_OR_COLLECT_EXTERNAL_EVIDENCE", "measurement-scope") }
	return map[string]any{"schema":"gooo/bounded-self-change/measurement-receipt/v2", "scope":"same_job_baseline_candidate_per_indicator", "scenario":ir.Scenario, "toolchain":execution.Toolchain, "runner":execution.Runner, "stage_measurements":execution.StageMeasurements, "indicators":execution.Indicators, "metric_summary_mode":"per_indicator_only", "claim":claim}
}

func ledgerIdentity(execution ExecutionInput) any { if execution.LiveLedger == nil { return map[string]any{"kind":"internal_parent_receipt","immutable":true,"digest":"immutable-ledger-parent-v0.50"} }; return execution.LiveLedger.Lock }

func evidenceManifest(outputDir string, ir SemanticIR, decision string, vector Vector, execution ExecutionInput) (map[string]any, error) {
	entries := make([]map[string]any, 0, 7)
	for _, name := range []string{"cycle-manifest.json", "frontier-receipt.json", "change-proposal.json", "test-impact-receipt.json", "measurement-receipt.json", "next-wave-proposal.json", "human-report.md"} {
		data, err := os.ReadFile(filepath.Join(outputDir, name)); if err != nil { return nil, err }; entries = append(entries, map[string]any{"name":name,"size_bytes":len(data),"digest":DigestBytes(data)})
	}
	packageDigest, err := DigestValue(entries); if err != nil { return nil, err }
	return map[string]any{"schema":"gooo/bounded-self-change/evidence-manifest/v2", "content_addressed":true, "package_digest":packageDigest, "entries":entries, "scenario":ir.Scenario, "decision":decision, "vector":vector, "released_tool_digests":toolDigests(ir.Tools), "runtime_local_validation_commands":0, "repository_writes":0, "remote_writes":0, "cross_project_required_gates":execution.CrossProjectRequiredGates}, nil
}

func toolDigests(tools []ToolLock) []string { values := make([]string, len(tools)); for i, tool := range tools { values[i] = tool.ID + "=" + tool.Tag + "@" + tool.AssetDigest }; sort.Strings(values); return values }
