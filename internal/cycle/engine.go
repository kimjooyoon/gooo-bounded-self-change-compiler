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
		Invariants: append([]InvariantDecl(nil), meta.Invariants...),
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
	invariantResults := evaluateInvariants(ir, caseResults)
	proofs, indicators := invariantVectors(ir.Invariants, caseResults)
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
		"vector": vector, "invariant_count": len(invariantResults), "cases": caseResults, "invariants": invariantResults,
		"proofs": proofs, "indicator_classes": indicators, "claim": frontierClaim,
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
	report := renderReport(ir, decision, caseResults, invariantResults, proofs, indicators, execution, frontierClaim, mode)
	if err := writeText(filepath.Join(outputDir, "human-report.md"), report); err != nil { return nil, err }
	evidence, err := evidenceManifest(outputDir, ir, decision, vector, proofs, indicators, execution)
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
	if len(meta.Stages) != len(StageIDs) || len(meta.Edges) != len(StageIDs)-1 || len(meta.Cases) != FixedCaseCount || len(meta.Invariants) != FixedCaseCount || len(meta.Tools) != 5 || len(meta.Tests) == 0 {
		return fmt.Errorf("v2 meta has incomplete fixed semantic declarations")
	}
	for i, stage := range meta.Stages {
		if stage.Ordinal != i+1 || stage.ID != StageIDs[i] || stage.Input == "" || stage.Output == "" || stage.SemanticEdge == "" { return fmt.Errorf("invalid v2 stage %d", i+1) }
		if i > 0 && meta.Edges[i-1] != (EdgeDecl{Ordinal: i, From: StageIDs[i-1], To: StageIDs[i], Relation: "causes"}) { return fmt.Errorf("invalid v2 causal edge %d", i) }
	}
	if err := validateContract(contract); err != nil { return err }
	for i := range meta.Cases { if !reflect.DeepEqual(meta.Cases[i], contract.Cases[i]) { return fmt.Errorf("v2 case %d differs between .gooo and contract", i+1) } }
	for i := range meta.Invariants {
		if !reflect.DeepEqual(meta.Invariants[i], contract.Invariants[i]) { return fmt.Errorf("v2 invariant %d differs between .gooo and contract", i+1) }
		if meta.Invariants[i].ID != meta.Cases[i].ID || meta.Invariants[i].Ordinal != meta.Cases[i].Ordinal || meta.Cases[i].ProofChoice != meta.Invariants[i].ProofChoice || meta.Cases[i].IndicatorClass != meta.Invariants[i].IndicatorClass {
			return fmt.Errorf("v2 invariant %d is not bound to its canonical case", i+1)
		}
	}
	for i := range meta.Tools { if !reflect.DeepEqual(meta.Tools[i], contract.Tools[i]) { return fmt.Errorf("released tool lock %d differs between .gooo and contract", i+1) } }
	return nil
}

func validateContract(contract Contract) error {
	if contract.Schema != ContractSchema || contract.ID != "bounded-self-change-v2" || contract.Version != "v2" || contract.CaseCount != FixedCaseCount || contract.InvariantCount != FixedCaseCount || !contract.Fixed || !sameStrings(contract.RequiredOutputs, RequiredOutputs) || len(contract.Cases) != FixedCaseCount || len(contract.Invariants) != FixedCaseCount || len(contract.Tools) != 5 || contract.LiveLedger.Tag != "v0.50.0" || !contract.LiveLedger.Immutable { return fmt.Errorf("invalid v2 lock contract") }
	if err := validateToolLocks(contract.Tools); err != nil { return err }
	if contract.LiveLedger != fixedLiveLedger() { return fmt.Errorf("v2 live ledger lock differs from immutable v0.50 release") }
	if contract.Denominator["CLOSED"] != 4 || contract.Denominator["UNKNOWN"] != 4 || contract.Denominator["REFUTED"] != 4 { return fmt.Errorf("v2 denominator must be exactly 4/4/4") }
	counts := map[string]int{}
	for i, c := range contract.Cases { if c.Ordinal != i+1 || c.ID == "" { return fmt.Errorf("invalid v2 canonical case %d", i+1) }; counts[c.ExpectedState]++ }
	if counts["CLOSED"] != 4 || counts["UNKNOWN"] != 4 || counts["REFUTED"] != 4 { return fmt.Errorf("v2 canonical vector must be exactly 4/4/4") }
	if err := validateInvariants(contract.Invariants); err != nil { return err }
	if contract.ProofTotals["FOUNDATION"] != 4 || contract.ProofTotals["COHERENCE"] != 4 || contract.ProofTotals["REGRESSION"] != 4 { return fmt.Errorf("v2 invariant proof vector must be exactly 4/4/4") }
	if contract.IndicatorTotals["DRIVER"] != 4 || contract.IndicatorTotals["OUTCOME"] != 4 || contract.IndicatorTotals["GUARDRAIL"] != 4 { return fmt.Errorf("v2 invariant indicator vector must be exactly 4/4/4") }
	for i := range contract.Cases {
		if contract.Cases[i].ID != contract.Invariants[i].ID || contract.Cases[i].ProofChoice != contract.Invariants[i].ProofChoice || contract.Cases[i].IndicatorClass != contract.Invariants[i].IndicatorClass {
			return fmt.Errorf("v2 case %d does not bind to its invariant", i+1)
		}
	}
	return nil
}

func fixedToolLocks() []ToolLock {
	return []ToolLock{
		{ID:"SELF_IMPROVEMENT_FRONTIER_PROJECTOR", Repository:"kimjooyoon/gooo-self-improvement-frontier-projector", Tag:"v0.2.0", ReleaseID:380832128, Immutable:true, TagObjectSHA:"042ca1bf7dfb432bd2ec0abef9e9884c9abe0286", TargetSHA:"98c3529013dad271337e424a7f07d4e5131d7edf", AssetName:"frontier-projector-evidence.tar.gz", AssetID:540161705, AssetSize:18846, AssetDigest:"sha256:112564378170baddfba44a1b3f5bd39216af65aefbc1f43f13732dbbbd5695a3"},
		{ID:"SEMANTIC_TEST_IMPACT_PROJECTOR", Repository:"kimjooyoon/gooo-semantic-test-impact-projector", Tag:"v0.1.2", ReleaseID:380755197, Immutable:true, TagObjectSHA:"6244b9c7115e10203a1472be2620497e9ac602e0", TargetSHA:"a2b1c7f5c24a20dfe44f25dfe50d3b2f60593ea7", AssetName:"gooo-semantic-test-impact-projector-evidence.tar.gz", AssetID:540004578, AssetSize:2654938, AssetDigest:"sha256:35dfe3921f82333fd9b44b02984a7e899dd937003802d3e423f760191cdf3a9c"},
		{ID:"MEASUREMENT_BOUNDARY_PROJECTOR", Repository:"kimjooyoon/gooo-measurement-boundary-projector", Tag:"v0.2.0", ReleaseID:380839207, Immutable:true, TagObjectSHA:"1bacf104da7ea9d6cf3ebd130801608b8e5afb14", TargetSHA:"1cff6318e748fec494dd9d28ec65db98b94293e0", AssetName:"gooo-measurement-boundary-projector-v0.2.0.tar.gz", AssetID:540176712, AssetSize:50378, AssetDigest:"sha256:90acd1f0a56ab38afe6c2b2b2033bd48b361b9dc43dc1257f606568f89076658"},
		{ID:"CONTENT_ADDRESSED_EVIDENCE_PROJECTOR", Repository:"kimjooyoon/gooo-content-addressed-evidence-projector", Tag:"v0.1.1", ReleaseID:380750147, Immutable:true, TagObjectSHA:"03dbbe7cd13549d4791e5e6086e036c81db3eac9", TargetSHA:"f3bfd2c6c05a45214fc7ed0732f2c3f0770bf463", AssetName:"gooo-content-addressed-evidence-projector-v0.1.1.tar.gz", AssetID:539995619, AssetSize:26063, AssetDigest:"sha256:a1d83f2503755bc6ea591d32cd4ef5d7a088e936da2a843d9d80af947acbe435"},
		{ID:"OPERATIONAL_PROVENANCE_PROJECTOR", Repository:"kimjooyoon/gooo-operational-provenance-projector", Tag:"v0.1.2", ReleaseID:380835618, Immutable:true, TagObjectSHA:"7f21cb959ab8d45c82a9790046a4eb86308c4622", TargetSHA:"36126b2a4b177d2b6f44713ffbf6908eb490af4b", AssetName:"gooo-operational-provenance-projector-evidence.tar.gz", AssetID:540170176, AssetSize:14601, AssetDigest:"sha256:23ed475552506ff279ae84210c1b825b44335db0b1ed258d407b88a59db1de16"},
	}
}

func fixedLiveLedger() ToolLock {
	return ToolLock{ID:"SELF_IMPROVEMENT_LEDGER_V0_50", Repository:"kimjooyoon/gooo-self-improvement-ledger", Tag:"v0.50.0", ReleaseID:380866481, Immutable:true, TagObjectSHA:"9e3263ea902bef64fa31c05ca7c1ab038ef962ef", TargetSHA:"e93768f4204e8a88214026ffa22febad7ecedcbd", AssetName:"gooo-self-improvement-ledger-e93768f4204e8a88214026ffa22febad7ecedcbd", AssetID:540246273, AssetSize:55178070, AssetDigest:"sha256:80575837d8ebb8d838bab912ff7802946fb37b2d90d923e8a9cec27bdf543e25"}
}

func validateToolLocks(tools []ToolLock) error {
	expected := fixedToolLocks()
	if len(tools) != len(expected) { return fmt.Errorf("v2 released tool lock count is not fixed at 5") }
	for i := range expected {
		if tools[i] != expected[i] { return fmt.Errorf("v2 released tool lock %d differs from immutable identity", i+1) }
	}
	return nil
}

func validateInvariants(invariants []InvariantDecl) error {
	if len(invariants) != FixedCaseCount { return fmt.Errorf("v2 invariant denominator must contain exactly 12 named invariants") }
	proofs := map[string]int{}
	indicators := map[string]int{}
	ids := map[string]bool{}
	validStages := map[string]bool{}
	for _, stage := range StageIDs { validStages[stage] = true }
	for i, invariant := range invariants {
		if invariant.Ordinal != i+1 || invariant.ID == "" || ids[invariant.ID] || invariant.Activity == "" || !validStages[invariant.Stage] || invariant.Step == "" {
			return fmt.Errorf("invalid named v2 invariant %d", i+1)
		}
		if invariant.ProofChoice != "FOUNDATION" && invariant.ProofChoice != "COHERENCE" && invariant.ProofChoice != "REGRESSION" { return fmt.Errorf("invalid proof choice for invariant %s", invariant.ID) }
		if invariant.IndicatorClass != "DRIVER" && invariant.IndicatorClass != "OUTCOME" && invariant.IndicatorClass != "GUARDRAIL" { return fmt.Errorf("invalid indicator class for invariant %s", invariant.ID) }
		ids[invariant.ID] = true
		proofs[invariant.ProofChoice]++
		indicators[invariant.IndicatorClass]++
	}
	if proofs["FOUNDATION"] != 4 || proofs["COHERENCE"] != 4 || proofs["REGRESSION"] != 4 { return fmt.Errorf("v2 named invariant proof vector must be exactly 4/4/4") }
	if indicators["DRIVER"] != 4 || indicators["OUTCOME"] != 4 || indicators["GUARDRAIL"] != 4 { return fmt.Errorf("v2 named invariant indicator vector must be exactly 4/4/4") }
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
		results = append(results, CaseResult{Ordinal:c.Ordinal, ID:c.ID, ExpectedState:c.ExpectedState, State:c.ExpectedState, Probe:c.Probe, Fixture:c.Fixture, SemanticEdge:c.SemanticEdge, Claim:claim, ProofChoice:c.ProofChoice, IndicatorClass:c.IndicatorClass})
	}
	return results
}

func evaluateInvariants(ir SemanticIR, cases []CaseResult) []InvariantResult {
	results := make([]InvariantResult, 0, len(ir.Invariants))
	byID := make(map[string]CaseResult, len(cases))
	for _, result := range cases { byID[result.ID] = result }
	for _, invariant := range ir.Invariants {
		caseResult := byID[invariant.ID]
		results = append(results, InvariantResult{Ordinal: invariant.Ordinal, ID: invariant.ID, Activity: invariant.Activity, Stage: invariant.Stage, Step: invariant.Step, ProofChoice: invariant.ProofChoice, IndicatorClass: invariant.IndicatorClass, DependsOn: append([]string(nil), invariant.DependsOn...), State: caseResult.State, Claim: caseResult.Claim})
	}
	return results
}

func invariantVectors(invariants []InvariantDecl, cases []CaseResult) ([]map[string]any, []map[string]any) {
	proofs := []map[string]any{}
	for _, choice := range []string{"FOUNDATION", "COHERENCE", "REGRESSION"} {
		entry := map[string]any{"choice": choice, "total": 0, "closed": 0, "unknown": 0, "refuted": 0}
		for _, invariant := range invariants {
			if invariant.ProofChoice != choice { continue }
			entry["total"] = entry["total"].(int) + 1
			for _, result := range cases {
				if result.ID != invariant.ID { continue }
				state := strings.ToLower(result.State)
				entry[state] = entry[state].(int) + 1
			}
		}
		proofs = append(proofs, entry)
	}
	indicators := []map[string]any{}
	for _, class := range []string{"DRIVER", "OUTCOME", "GUARDRAIL"} {
		entry := map[string]any{"class": class, "total": 0, "closed": 0, "unknown": 0, "refuted": 0}
		for _, invariant := range invariants {
			if invariant.IndicatorClass != class { continue }
			entry["total"] = entry["total"].(int) + 1
			for _, result := range cases {
				if result.ID != invariant.ID { continue }
				state := strings.ToLower(result.State)
				entry[state] = entry[state].(int) + 1
			}
		}
		indicators = append(indicators, entry)
	}
	return proofs, indicators
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
		if execution.LiveLedger == nil || execution.LiveLedger.Lock.ID != "SELF_IMPROVEMENT_LEDGER_V0_50" || !execution.LiveLedger.Lock.Immutable || execution.LiveLedger.Lock.Tag != "v0.50.0" || execution.LiveLedger.ActionableFrontier != "EXTERNAL_UTILITY_EVIDENCE" || execution.LiveLedger.AutomationCanProduce || !execution.LiveLedger.ExternalEvidence || execution.LiveLedger.Decision != "UNKNOWN" { return fmt.Errorf("live v0.50 observation must stop at UNKNOWN/HUMAN_EXTERNAL_EVIDENCE_REQUIRED") }
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

func evidenceManifest(outputDir string, ir SemanticIR, decision string, vector Vector, proofs, indicators []map[string]any, execution ExecutionInput) (map[string]any, error) {
	entries := make([]map[string]any, 0, 7)
	for _, name := range []string{"cycle-manifest.json", "frontier-receipt.json", "change-proposal.json", "test-impact-receipt.json", "measurement-receipt.json", "next-wave-proposal.json", "human-report.md"} {
		data, err := os.ReadFile(filepath.Join(outputDir, name)); if err != nil { return nil, err }; entries = append(entries, map[string]any{"name":name,"size_bytes":len(data),"digest":DigestBytes(data)})
	}
	packageDigest, err := DigestValue(entries); if err != nil { return nil, err }
	return map[string]any{"schema":"gooo/bounded-self-change/evidence-manifest/v2", "content_addressed":true, "package_digest":packageDigest, "entries":entries, "scenario":ir.Scenario, "decision":decision, "vector":vector, "invariant_count":len(ir.Invariants), "proofs":proofs, "indicator_classes":indicators, "released_tool_digests":toolDigests(ir.Tools), "runtime_local_validation_commands":0, "repository_writes":0, "remote_writes":0, "cross_project_required_gates":execution.CrossProjectRequiredGates}, nil
}

func toolDigests(tools []ToolLock) []string { values := make([]string, len(tools)); for i, tool := range tools { values[i] = tool.ID + "=" + tool.Tag + "@" + tool.AssetDigest }; sort.Strings(values); return values }
