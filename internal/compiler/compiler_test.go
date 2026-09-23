package compiler

import "testing"

func TestResolutionPrecedenceFailsClosed(t *testing.T) {
	if got := resolveClaim(Claim{State: "UNKNOWN"}); got != "REFUTED" {
		t.Fatalf("incomplete UNKNOWN claim resolved to %q", got)
	}
	if got := resolveClaim(Claim{State: "REFUTED"}); got != "REFUTED" {
		t.Fatalf("REFUTED claim resolved to %q", got)
	}
	if got := resolveClaim(Claim{State: "CLOSED"}); got != "CLOSED" {
		t.Fatalf("CLOSED claim resolved to %q", got)
	}
}

func TestGraphHasBoundedDecisionPath(t *testing.T) {
	ir := SemanticIR{Graph: SemanticGraph{Schema: GraphSchema}}
	ir.Graph = buildGraph(ir)
	if len(ir.Graph.Nodes) != 9 {
		t.Fatalf("graph nodes = %d, want 9", len(ir.Graph.Nodes))
	}
	if len(ir.Graph.Edges) != 8 {
		t.Fatalf("graph edges = %d, want 8", len(ir.Graph.Edges))
	}
}

func TestExactPairClosesImprovement(t *testing.T) {
	claim := improvementClaim(Claim{State: "CLOSED"}, ExactBeforeAfter{Before: 0, After: 1})
	if claim.State != "CLOSED" || claim.Reason != "EXACT_BEFORE_AFTER_INTEGER_PAIR_PROVES_COUNTEREXAMPLE_REPAIR" {
		t.Fatalf("unexpected improvement claim: %#v", claim)
	}
}

func TestParseKeyValuesRejectsDuplicateKeys(t *testing.T) {
	if _, err := parseKeyValues([]string{"id=first", "id=second"}); err == nil {
		t.Fatal("duplicate declaration key was accepted")
	}
}

func TestValidateMetaRejectsMutatedCanonicalCase(t *testing.T) {
	cases := fixedCanonicalCases()
	meta := MetaDecl{
		Schema: MetaSchema, Version: "v1", Owner: "gooo",
		Rules: []RuleSpec{{Name: "positive_only", Predicate: "input>0"}, {Name: "non_negative", Predicate: "input>=0"}},
		Precedence: []string{"REFUTED", "UNKNOWN", "CLOSED"},
		UnknownFields: []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"},
		Cases: append([]CanonicalCase(nil), cases...),
	}
	contract := Contract{
		Schema: ContractSchema, ID: "bounded-self-change-v1", Version: "v1", CaseCount: FixedCaseCount, Fixed: true,
		Cases: append([]CanonicalCase(nil), cases...),
	}
	contract.Cases[0].Fixture = "counterexample_persists"
	meta.Cases[0] = contract.Cases[0]
	if err := ValidateMeta(meta, contract); err == nil {
		t.Fatal("mutated canonical case was accepted")
	}
}

func TestValidateIRRejectsRehashedGraphMutation(t *testing.T) {
	ir := SemanticIR{
		Schema: IRSchema, Version: "v1", Scenario: "graph-integrity", SourceDigest: "source", MetaDigest: "meta", ContractDigest: "contract",
		BaselineRule: "positive_only", CandidateRule: "non_negative", Cases: make([]CanonicalCase, FixedCaseCount),
	}
	ir.Graph = buildGraph(ir)
	digest, err := unsignedIRDigest(ir)
	if err != nil {
		t.Fatal(err)
	}
	ir.Graph.Edges[0].Relation = "tampered"
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		t.Fatal(err)
	}
	if digest == ir.IRDigest {
		t.Fatal("graph mutation did not change IR digest")
	}
	if err := ValidateIR(ir); err == nil {
		t.Fatal("rehashed graph mutation was accepted")
	}
}
