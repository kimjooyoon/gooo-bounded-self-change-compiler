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
