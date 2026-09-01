package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateMeta(meta MetaDecl, contract Contract) error {
	if meta.Schema != MetaSchema || meta.Version != "v1" || meta.Owner != "gooo" {
		return fmt.Errorf("meta declaration must be owned by .gooo")
	}
	if len(meta.Rules) != 2 || meta.Rules[0] != (RuleSpec{Name: "positive_only", Predicate: "input>0"}) || meta.Rules[1] != (RuleSpec{Name: "non_negative", Predicate: "input>=0"}) {
		return fmt.Errorf("meta declaration must fix baseline and candidate rules")
	}
	if strings.Join(meta.Precedence, ">") != ExpectedPrecedence {
		return fmt.Errorf("resolution precedence must be %s", ExpectedPrecedence)
	}
	if strings.Join(meta.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return fmt.Errorf("UNKNOWN six-field contract mismatch")
	}
	if len(meta.Cases) != FixedCaseCount || len(contract.Cases) != FixedCaseCount {
		return fmt.Errorf("canonical denominator must contain exactly %d cases", FixedCaseCount)
	}
	for index := 0; index < FixedCaseCount; index++ {
		if !sameCase(meta.Cases[index], contract.Cases[index]) || meta.Cases[index].Ordinal != index+1 {
			return fmt.Errorf("canonical case %d does not match the fixed contract", index+1)
		}
	}
	return nil
}

func ValidateSource(source SourceDecl, contract Contract) error {
	if source.Schema != SourceSchema || source.Version != "v1" || source.Scenario == "" {
		return fmt.Errorf("invalid bounded self-change source")
	}
	if source.Baseline.Rule != "positive_only" {
		return fmt.Errorf("unsupported baseline rule %q", source.Baseline.Rule)
	}
	if !source.Counterexample.Present || source.Counterexample.Input != 0 || source.Counterexample.Expected != "accept" || source.Counterexample.Label != "zero_rejected" {
		return fmt.Errorf("source must declare the fixed zero counterexample")
	}
	if source.Intent.Name != "accept_zero" {
		return fmt.Errorf("source must declare accept_zero improvement intent")
	}
	if source.EditSurface != (EditSurfaceDecl{Field: "threshold", Operator: "lower_bound", Constraint: "single_rule_only"}) {
		return fmt.Errorf("permitted edit surface is not fixed")
	}
	if len(source.Invariants) != 1 || source.Invariants[0] != (InvariantDecl{ID: "non_negative_only", Expression: "negative_inputs_rejected"}) {
		return fmt.Errorf("source must declare the non-negative invariant")
	}
	if len(source.Guardrails) != 1 || source.Guardrails[0] != (GuardrailDecl{ID: "reject_negative", Input: -1, Expected: "reject", Expression: "input=-1->reject"}) {
		return fmt.Errorf("source must declare the negative guardrail")
	}
	if len(source.Evidence) != 4 {
		return fmt.Errorf("source must declare four evidence obligations")
	}
	expectedEvidence := []EvidenceDecl{
		{ID: "source_digest", Required: true, State: "complete"},
		{ID: "meta_digest", Required: true, State: "complete"},
		{ID: "contract_digest", Required: true, State: "complete"},
		{ID: "runner_toolchain", Required: true, State: "complete"},
	}
	for index := range expectedEvidence {
		if source.Evidence[index] != expectedEvidence[index] {
			return fmt.Errorf("evidence obligation %d is not fixed", index+1)
		}
	}
	if source.Authority != zeroAuthority() {
		return fmt.Errorf("source authority must be zero and non-escalating")
	}
	if contract.Schema != ContractSchema || contract.ID != "bounded-self-change-v1" || contract.Version != "v1" || contract.CaseCount != FixedCaseCount || !contract.Fixed {
		return fmt.Errorf("invalid fixed contract")
	}
	return nil
}

func BuildIR(meta MetaDecl, source SourceDecl, contract Contract) (SemanticIR, error) {
	if err := ValidateMeta(meta, contract); err != nil {
		return SemanticIR{}, err
	}
	if err := ValidateSource(source, contract); err != nil {
		return SemanticIR{}, err
	}
	contractDigest, err := ContractDigest(contract)
	if err != nil {
		return SemanticIR{}, err
	}
	ir := SemanticIR{
		Schema:         IRSchema,
		Version:        "v1",
		Scenario:       source.Scenario,
		SourceDigest:   source.SourceDigest,
		MetaDigest:     meta.MetaDigest,
		ContractDigest: contractDigest,
		BaselineRule:   source.Baseline.Rule,
		CandidateRule:  "non_negative",
		Counterexample: source.Counterexample,
		Intent:         source.Intent,
		EditSurface:    source.EditSurface,
		Invariants:     append([]InvariantDecl(nil), source.Invariants...),
		Guardrails:     append([]GuardrailDecl(nil), source.Guardrails...),
		Evidence:       append([]EvidenceDecl(nil), source.Evidence...),
		Authority:      source.Authority,
		Precedence:     append([]string(nil), meta.Precedence...),
		UnknownFields:  append([]string(nil), meta.UnknownFields...),
		Cases:          append([]CanonicalCase(nil), contract.Cases...),
	}
	ir.Graph = buildGraph(ir)
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func ValidateIR(ir SemanticIR) error {
	if ir.Schema != IRSchema || ir.Version != "v1" || ir.Scenario == "" || ir.SourceDigest == "" || ir.MetaDigest == "" || ir.ContractDigest == "" || ir.IRDigest == "" {
		return fmt.Errorf("semantic IR identity is incomplete")
	}
	if ir.BaselineRule != "positive_only" || ir.CandidateRule != "non_negative" || len(ir.Cases) != FixedCaseCount || ir.Graph.Schema != GraphSchema {
		return fmt.Errorf("semantic IR is not fixed to the bounded loop")
	}
	expected, err := unsignedIRDigest(ir)
	if err != nil {
		return err
	}
	if expected != ir.IRDigest {
		return fmt.Errorf("semantic IR digest mismatch")
	}
	return nil
}

func Generate(meta MetaDecl, source SourceDecl, contract Contract, outputDir string) (SemanticIR, error) {
	if err := ensureCallerOutput(outputDir); err != nil {
		return SemanticIR{}, err
	}
	ir, err := BuildIR(meta, source, contract)
	if err != nil {
		return SemanticIR{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil {
		return SemanticIR{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-graph.json"), ir.Graph); err != nil {
		return SemanticIR{}, err
	}
	candidate, err := candidateArtifact(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "candidate-artifact.json"), candidate); err != nil {
		return SemanticIR{}, err
	}
	if err := WriteText(filepath.Join(outputDir, "candidate.gooo"), renderCandidateGooo(ir)); err != nil {
		return SemanticIR{}, err
	}
	if err := WriteText(filepath.Join(outputDir, "candidate.go"), renderCandidateGo()); err != nil {
		return SemanticIR{}, err
	}
	proposal := PatchProposal{
		Schema:           "gooo/bounded-self-change/patch-proposal/v1",
		Scenario:         ir.Scenario,
		InputFile:        "examples/bounded-self-change-v1/self-change.gooo",
		EditSurface:      ir.EditSurface.Field,
		Before:           "rule name=positive_only predicate=input>0",
		After:            "rule name=non_negative predicate=input>=0",
		UnifiedDiff:      "-rule name=positive_only predicate=input>0\n+rule name=non_negative predicate=input>=0",
		SourceDigest:     ir.SourceDigest,
		CandidateDigest:  candidate.CandidateDigest,
		RepositoryWrites: 0,
	}
	if err := WriteJSON(filepath.Join(outputDir, "proposal.patch.json"), proposal); err != nil {
		return SemanticIR{}, err
	}
	plan := map[string]any{
		"schema":                "gooo/bounded-self-change/verification-plan/v1",
		"scenario":              ir.Scenario,
		"source_digest":         ir.SourceDigest,
		"meta_digest":           ir.MetaDigest,
		"contract_digest":       ir.ContractDigest,
		"ir_digest":             ir.IRDigest,
		"runner":                RunnerIdentity,
		"toolchain":             ToolchainIdentity,
		"before":                map[string]any{"input": ir.Counterexample.Input, "expected": "reject"},
		"after":                 map[string]any{"input": ir.Counterexample.Input, "expected": "accept"},
		"guardrail":             map[string]any{"input": ir.Guardrails[0].Input, "expected": ir.Guardrails[0].Expected},
		"required_observations": []string{"before-counterexample", "after-counterexample", "after-negative", "after-positive"},
	}
	if err := WriteJSON(filepath.Join(outputDir, "verification-plan.json"), plan); err != nil {
		return SemanticIR{}, err
	}
	receipt := GenerationReceipt{
		Schema:                    ReceiptSchema,
		SourceToIR:                "GOOO_DECLARATIONS_TO_SEMANTIC_IR_AND_GRAPH",
		IRToCandidate:             "SEMANTIC_IR_TO_CALLER_OWNED_CANDIDATE_ARTIFACTS",
		GeneratedFiles:            []string{"semantic-ir.json", "semantic-graph.json", "candidate-artifact.json", "candidate.gooo", "candidate.go", "proposal.patch.json", "verification-plan.json", "generation-receipt.json"},
		Generated:                 8,
		CallerOwnedTempOutput:     true,
		RepositoryWrites:          0,
		ApplyAuthority:            0,
		CommitAuthority:           0,
		MergeAuthority:            0,
		CrossProjectRequiredGates: 0,
	}
	if err := WriteJSON(filepath.Join(outputDir, "generation-receipt.json"), receipt); err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func candidateArtifact(ir SemanticIR) (CandidateArtifact, error) {
	candidate := CandidateArtifact{
		Schema:         CandidateSchema,
		Scenario:       ir.Scenario,
		BaselineRule:   ir.BaselineRule,
		CandidateRule:  ir.CandidateRule,
		SourceDigest:   ir.SourceDigest,
		MetaDigest:     ir.MetaDigest,
		ContractDigest: ir.ContractDigest,
		IRDigest:       ir.IRDigest,
		EditSurface:    ir.EditSurface.Field,
		ProposedChange: "threshold:strict_positive_to_non_negative",
	}
	digest, err := unsignedCandidateDigest(candidate)
	if err != nil {
		return CandidateArtifact{}, err
	}
	candidate.CandidateDigest = digest
	return candidate, nil
}

func buildGraph(ir SemanticIR) SemanticGraph {
	nodes := []GraphNode{
		{ID: "baseline", Kind: "baseline_semantics", SemanticEdge: "source:baseline->rule"},
		{ID: "counterexample", Kind: "observed_counterexample", SemanticEdge: "counterexample:input->baseline"},
		{ID: "intent", Kind: "improvement_intent", SemanticEdge: "intent:accept_zero->edit"},
		{ID: "edit_surface", Kind: "permitted_edit_surface", SemanticEdge: "edit:threshold->candidate"},
		{ID: "invariant", Kind: "invariant", SemanticEdge: "invariant:negative_inputs_rejected->candidate"},
		{ID: "guardrail", Kind: "guardrail", SemanticEdge: "guardrail:-1->reject"},
		{ID: "evidence", Kind: "evidence_obligations", SemanticEdge: "evidence:provenance->decision"},
		{ID: "candidate", Kind: "candidate_rule", SemanticEdge: "candidate:non_negative->execution"},
		{ID: "decision", Kind: "decision", SemanticEdge: "execution->CLOSED_UNKNOWN_REFUTED"},
	}
	edges := []GraphEdge{
		{From: "baseline", To: "counterexample", Relation: "observed_by"},
		{From: "counterexample", To: "intent", Relation: "motivates"},
		{From: "intent", To: "edit_surface", Relation: "constrains"},
		{From: "edit_surface", To: "candidate", Relation: "permits"},
		{From: "invariant", To: "candidate", Relation: "guards"},
		{From: "guardrail", To: "candidate", Relation: "checks"},
		{From: "candidate", To: "evidence", Relation: "provenance"},
		{From: "evidence", To: "decision", Relation: "supports"},
	}
	return SemanticGraph{Schema: GraphSchema, Nodes: nodes, Edges: edges}
}

func renderCandidateGooo(ir SemanticIR) string {
	return fmt.Sprintf("gooo bounded_self_change_candidate v1\nrule name=%s predicate=input>=0\nprovenance source_digest=%s meta_digest=%s contract_digest=%s ir_digest=%s\nchange field=%s operator=%s before=%s after=%s\n", ir.CandidateRule, ir.SourceDigest, ir.MetaDigest, ir.ContractDigest, ir.IRDigest, ir.EditSurface.Field, ir.EditSurface.Operator, ir.BaselineRule, ir.CandidateRule)
}

func renderCandidateGo() string {
	return `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func baseline(input int) bool {
	return input > 0
}

func candidate(input int) bool {
	return input >= 0
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: candidate.go before|after integer")
		os.Exit(2)
	}
	input, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var accepted bool
	switch os.Args[1] {
	case "before":
		accepted = baseline(input)
	case "after":
		accepted = candidate(input)
	default:
		fmt.Fprintln(os.Stderr, "variant must be before or after")
		os.Exit(2)
	}
	verdict := "reject"
	if accepted {
		verdict = "accept"
	}
	result := map[string]any{"variant": os.Args[1], "input": input, "accepted": accepted, "verdict": verdict}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`
}

func sameCase(a, b CanonicalCase) bool {
	return a.Ordinal == b.Ordinal && a.ID == b.ID && a.ExpectedState == b.ExpectedState && a.Probe == b.Probe && a.Fixture == b.Fixture && a.SemanticEdge == b.SemanticEdge && sameStrings(a.DependsOn, b.DependsOn) && a.Reason == b.Reason
}

func zeroAuthority() Authority {
	return Authority{}
}

func ensureCallerOutput(path string) error {
	if path == "" {
		return fmt.Errorf("caller-owned output path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if repoRoot := findRepoRoot(); repoRoot != "" && isWithin(repoRoot, abs) {
		return fmt.Errorf("caller-owned output must be outside repository: %s", repoRoot)
	}
	info, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("caller-owned output must be a directory")
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("caller-owned output must be empty")
	}
	return nil
}

func findRepoRoot() string {
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if info, err := os.Stat(filepath.Join(current, ".git")); err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func isWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
