package compiler

import "strings"

const (
	SourceSchema       = "gooo/bounded-self-change/source/v1"
	MetaSchema         = "gooo/bounded-self-change/meta/v1"
	ContractSchema     = "gooo/bounded-self-change/contract/v1"
	IRSchema           = "gooo/bounded-self-change/semantic-ir/v1"
	GraphSchema        = "gooo/bounded-self-change/semantic-graph/v1"
	CandidateSchema    = "gooo/bounded-self-change/candidate/v1"
	ReceiptSchema      = "gooo/bounded-self-change/generation-receipt/v1"
	ExecutionSchema    = "gooo/bounded-self-change/execution/v1"
	ReportSchema       = "gooo/bounded-self-change/decision-dossier/v1"
	FixedCaseCount     = 9
	ToolchainIdentity  = "go1.27.0"
	RunnerIdentity     = "ubuntu-latest"
	ExpectedPrecedence = "REFUTED>UNKNOWN>CLOSED"
)

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	ApplyAuthority            int `json:"apply_authority"`
	CommitAuthority           int `json:"commit_authority"`
	MergeAuthority            int `json:"merge_authority"`
	TagAuthority              int `json:"tag_authority"`
	ReleaseAuthority          int `json:"release_authority"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type BaselineDecl struct {
	Rule string `json:"rule"`
}

type CounterexampleDecl struct {
	Input    int    `json:"input"`
	Expected string `json:"expected"`
	Label    string `json:"label"`
	Present  bool   `json:"present"`
}

type IntentDecl struct {
	Name string `json:"name"`
}

type EditSurfaceDecl struct {
	Field      string `json:"field"`
	Operator   string `json:"operator"`
	Constraint string `json:"constraint"`
}

type InvariantDecl struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
}

type GuardrailDecl struct {
	ID         string `json:"id"`
	Input      int    `json:"input"`
	Expected   string `json:"expected"`
	Expression string `json:"expression"`
}

type EvidenceDecl struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
	State    string `json:"state"`
}

type SourceDecl struct {
	Schema         string             `json:"schema"`
	Version        string             `json:"version"`
	Scenario       string             `json:"scenario"`
	Baseline       BaselineDecl       `json:"baseline"`
	Counterexample CounterexampleDecl `json:"counterexample"`
	Intent         IntentDecl         `json:"improvement_intent"`
	EditSurface    EditSurfaceDecl    `json:"permitted_edit_surface"`
	Invariants     []InvariantDecl    `json:"invariants"`
	Guardrails     []GuardrailDecl    `json:"guardrails"`
	Evidence       []EvidenceDecl     `json:"evidence_obligations"`
	Authority      Authority          `json:"authority"`
	SourceDigest   string             `json:"source_digest"`
}

type RuleSpec struct {
	Name      string `json:"name"`
	Predicate string `json:"predicate"`
}

type CanonicalCase struct {
	Ordinal       int      `json:"ordinal"`
	ID            string   `json:"id"`
	ExpectedState string   `json:"expected_state"`
	Probe         string   `json:"probe"`
	Fixture       string   `json:"fixture"`
	SemanticEdge  string   `json:"semantic_edge"`
	DependsOn     []string `json:"depends_on"`
	Reason        string   `json:"reason"`
}

type MetaDecl struct {
	Schema        string          `json:"schema"`
	Version       string          `json:"version"`
	Owner         string          `json:"owner"`
	Rules         []RuleSpec      `json:"rules"`
	Precedence    []string        `json:"precedence"`
	UnknownFields []string        `json:"unknown_fields"`
	Cases         []CanonicalCase `json:"canonical_cases"`
	MetaDigest    string          `json:"meta_digest"`
}

type Contract struct {
	Schema         string          `json:"schema"`
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	CaseCount      int             `json:"case_count"`
	Fixed          bool            `json:"fixed"`
	Cases          []CanonicalCase `json:"cases"`
	RequiredFiles  []string        `json:"required_files"`
	RequiredSource []string        `json:"required_source_fields"`
}

type GraphNode struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	SemanticEdge string `json:"semantic_edge"`
}

type GraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type SemanticGraph struct {
	Schema string      `json:"schema"`
	Nodes  []GraphNode `json:"nodes"`
	Edges  []GraphEdge `json:"edges"`
}

type SemanticIR struct {
	Schema         string             `json:"schema"`
	Version        string             `json:"version"`
	Scenario       string             `json:"scenario"`
	SourceDigest   string             `json:"source_digest"`
	MetaDigest     string             `json:"meta_digest"`
	ContractDigest string             `json:"contract_digest"`
	BaselineRule   string             `json:"baseline_rule"`
	CandidateRule  string             `json:"candidate_rule"`
	Counterexample CounterexampleDecl `json:"counterexample"`
	Intent         IntentDecl         `json:"improvement_intent"`
	EditSurface    EditSurfaceDecl    `json:"permitted_edit_surface"`
	Invariants     []InvariantDecl    `json:"invariants"`
	Guardrails     []GuardrailDecl    `json:"guardrails"`
	Evidence       []EvidenceDecl     `json:"evidence_obligations"`
	Authority      Authority          `json:"authority"`
	Precedence     []string           `json:"precedence"`
	UnknownFields  []string           `json:"unknown_fields"`
	Cases          []CanonicalCase    `json:"canonical_cases"`
	Graph          SemanticGraph      `json:"graph"`
	IRDigest       string             `json:"ir_digest,omitempty"`
}

type CandidateArtifact struct {
	Schema          string `json:"schema"`
	Scenario        string `json:"scenario"`
	BaselineRule    string `json:"baseline_rule"`
	CandidateRule   string `json:"candidate_rule"`
	SourceDigest    string `json:"source_digest"`
	MetaDigest      string `json:"meta_digest"`
	ContractDigest  string `json:"contract_digest"`
	IRDigest        string `json:"ir_digest"`
	EditSurface     string `json:"edit_surface"`
	ProposedChange  string `json:"proposed_change"`
	CandidateDigest string `json:"candidate_digest,omitempty"`
}

type PatchProposal struct {
	Schema           string `json:"schema"`
	Scenario         string `json:"scenario"`
	InputFile        string `json:"input_file"`
	EditSurface      string `json:"edit_surface"`
	Before           string `json:"before"`
	After            string `json:"after"`
	UnifiedDiff      string `json:"unified_diff"`
	SourceDigest     string `json:"source_digest"`
	CandidateDigest  string `json:"candidate_digest"`
	RepositoryWrites int    `json:"repository_writes"`
}

type GenerationReceipt struct {
	Schema                    string   `json:"schema"`
	SourceToIR                string   `json:"source_to_ir"`
	IRToCandidate             string   `json:"ir_to_candidate"`
	GeneratedFiles            []string `json:"generated_files"`
	Generated                 int      `json:"generated"`
	CallerOwnedTempOutput     bool     `json:"caller_owned_temp_output"`
	RepositoryWrites          int      `json:"repository_writes"`
	ApplyAuthority            int      `json:"apply_authority"`
	CommitAuthority           int      `json:"commit_authority"`
	MergeAuthority            int      `json:"merge_authority"`
	CrossProjectRequiredGates int      `json:"cross_project_required_gates"`
}

type ExecutionObservation struct {
	Variant  string `json:"variant"`
	Input    int    `json:"input"`
	Verdict  string `json:"verdict"`
	Accepted bool   `json:"accepted"`
}

type ExactBeforeAfter struct {
	Scenario       string `json:"scenario"`
	SourceDigest   string `json:"source_digest"`
	MetaDigest     string `json:"meta_digest"`
	ContractDigest string `json:"contract_digest"`
	Fixture        string `json:"fixture"`
	Toolchain      string `json:"toolchain"`
	Runner         string `json:"runner"`
	Metric         string `json:"metric"`
	Before         int    `json:"before"`
	After          int    `json:"after"`
}

type ExecutionReport struct {
	Schema         string                 `json:"schema"`
	Scenario       string                 `json:"scenario"`
	SourceDigest   string                 `json:"source_digest"`
	MetaDigest     string                 `json:"meta_digest"`
	ContractDigest string                 `json:"contract_digest"`
	IRDigest       string                 `json:"ir_digest"`
	Toolchain      string                 `json:"toolchain"`
	Runner         string                 `json:"runner"`
	Observations   []ExecutionObservation `json:"observations"`
	BeforeAfter    ExactBeforeAfter       `json:"before_after"`
}

type Claim struct {
	State         string   `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CaseResult struct {
	Ordinal       int    `json:"ordinal"`
	ID            string `json:"id"`
	ExpectedState string `json:"expected_state"`
	State         string `json:"state"`
	Probe         string `json:"probe"`
	Fixture       string `json:"fixture"`
	SemanticEdge  string `json:"semantic_edge"`
	Claim         Claim  `json:"claim"`
}

type CaseSummary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type Report struct {
	Schema                      string            `json:"schema"`
	Decision                    string            `json:"decision"`
	FixedConformanceDenominator int               `json:"fixed_conformance_denominator"`
	Precedence                  []string          `json:"precedence"`
	Scenario                    string            `json:"scenario"`
	SourceDigest                string            `json:"source_digest"`
	MetaDigest                  string            `json:"meta_digest"`
	ContractDigest              string            `json:"contract_digest"`
	IRDigest                    string            `json:"ir_digest"`
	CandidateDigest             string            `json:"candidate_digest"`
	CandidateRule               string            `json:"candidate_rule"`
	Summary                     CaseSummary       `json:"summary"`
	Cases                       []CaseResult      `json:"cases"`
	Improvement                 Claim             `json:"improvement"`
	Utility                     Claim             `json:"utility"`
	ExactBeforeAfter            ExactBeforeAfter  `json:"exact_before_after"`
	RuntimeAuthority            Authority         `json:"runtime_authority"`
	Evidence                    map[string]string `json:"evidence"`
}

func (c Claim) HasUnknownTuple() bool {
	if c.State != "UNKNOWN" || c.Stage == "" || c.Step == "" || c.Reason == "" || c.UnknownClass == "" || c.NextOperation == "" || len(c.BlockedBy) == 0 {
		return false
	}
	for _, blocker := range c.BlockedBy {
		if strings.TrimSpace(blocker) == "" {
			return false
		}
	}
	return true
}
