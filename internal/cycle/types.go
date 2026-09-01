package cycle

const (
	CycleSchema       = "gooo/bounded-self-change/cycle/v2"
	MetaSchema        = "gooo/bounded-self-change/meta/v2"
	SourceSchema      = "gooo/bounded-self-change/source/v2"
	ContractSchema    = "gooo/bounded-self-change/contract/v2"
	PreparationSchema = "gooo/bounded-self-change/preparation/v2"
	ExecutionSchema   = "gooo/bounded-self-change/execution/v2"
	FixedCaseCount    = 12
	ToolchainIdentity = "go1.27.0"
	RunnerIdentity    = "ubuntu-latest"
)

var StageIDs = []string{
	"OBSERVE_IMMUTABLE_LEDGER",
	"PROJECT_CAUSAL_FRONTIER",
	"REQUIRE_EXPLICIT_UNIQUE_ACTION",
	"GENERATE_BOUNDED_CHANGE",
	"APPLY_CALLER_TEMP_ONLY",
	"PROJECT_SEMANTIC_TEST_IMPACT",
	"MEASURE_COVERED_STAGE",
	"EVALUATE_PER_INDICATOR_VECTOR",
	"PACKAGE_CONTENT_ADDRESSED_EVIDENCE",
	"EMIT_NEXT_LEDGER_WAVE_PROPOSAL",
}

var RequiredOutputs = []string{
	"cycle-manifest.json",
	"frontier-receipt.json",
	"change-proposal.json",
	"test-impact-receipt.json",
	"measurement-receipt.json",
	"evidence-manifest.json",
	"next-wave-proposal.json",
	"human-report.md",
}

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	RemoteWrites              int `json:"remote_writes"`
	ApplyAuthority            int `json:"apply_authority"`
	CommitAuthority           int `json:"commit_authority"`
	MergeAuthority            int `json:"merge_authority"`
	TagAuthority              int `json:"tag_authority"`
	ReleaseAuthority          int `json:"release_authority"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type StageDecl struct {
	Ordinal    int    `json:"ordinal"`
	ID         string `json:"id"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	SemanticEdge string `json:"semantic_edge"`
}

type EdgeDecl struct {
	Ordinal int    `json:"ordinal"`
	From    string `json:"from"`
	To      string `json:"to"`
	Relation string `json:"relation"`
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

type ToolLock struct {
	ID            string `json:"id"`
	Repository    string `json:"repository"`
	Tag           string `json:"tag"`
	ReleaseID     int64  `json:"release_id"`
	Immutable     bool   `json:"immutable"`
	TagObjectSHA  string `json:"tag_object_sha"`
	TargetSHA     string `json:"target_sha"`
	AssetName     string `json:"asset_name"`
	AssetID       int64  `json:"asset_id"`
	AssetSize     int64  `json:"asset_size_bytes"`
	AssetDigest   string `json:"asset_digest"`
}

type TestImpactDecl struct {
	ID   string `json:"id"`
	Root string `json:"semantic_root"`
	Edge string `json:"semantic_edge"`
}

type MetaDecl struct {
	Schema        string          `json:"schema"`
	Version       string          `json:"version"`
	Owner         string          `json:"owner"`
	Precedence    []string        `json:"precedence"`
	UnknownFields []string        `json:"unknown_fields"`
	Authority     Authority       `json:"authority"`
	LiveFrontier  string          `json:"live_frontier"`
	LiveAction    string          `json:"live_action"`
	Stages        []StageDecl     `json:"stages"`
	Edges         []EdgeDecl      `json:"causal_edges"`
	Tests         []TestImpactDecl `json:"semantic_tests"`
	Cases         []CanonicalCase `json:"canonical_cases"`
	Tools         []ToolLock      `json:"released_tools"`
	MetaDigest    string          `json:"meta_digest"`
}

type SourceDecl struct {
	Schema                string       `json:"schema"`
	Version               string       `json:"version"`
	Scenario              string       `json:"scenario"`
	Mode                  string       `json:"mode"`
	BaselineRule          string       `json:"baseline_rule"`
	CandidateRule         string       `json:"candidate_rule"`
	FrontierID            string       `json:"frontier_id"`
	FrontierClass         string       `json:"frontier_class"`
	FrontierActionable    bool         `json:"frontier_actionable"`
	FrontierUnique        bool         `json:"frontier_unique"`
	ActionID              string       `json:"action_id"`
	ActionAuthority       string       `json:"action_authority"`
	ParentProofState      string       `json:"parent_proof_state"`
	ParentProofDigest     string       `json:"parent_proof_digest"`
	CounterexampleInput   int          `json:"counterexample_input"`
	CounterexampleExpected string      `json:"counterexample_expected"`
	CounterexampleLabel   string       `json:"counterexample_label"`
	EditSurface           string       `json:"edit_surface"`
	SemanticRoot          string       `json:"semantic_root"`
	TestImpacts           []string     `json:"test_impacts"`
	CounterexampleNextWave bool        `json:"counterexample_next_wave"`
	RepositoryAuthority   int          `json:"repository_authority"`
	EvidenceScope         string       `json:"evidence_scope"`
	Authority             Authority    `json:"authority"`
	SourceDigest          string       `json:"source_digest"`
}

type Contract struct {
	Schema      string          `json:"schema"`
	ID          string          `json:"id"`
	Version     string          `json:"version"`
	CaseCount   int             `json:"case_count"`
	Fixed       bool            `json:"fixed"`
	Denominator map[string]int  `json:"denominator"`
	RequiredOutputs []string    `json:"required_outputs"`
	Cases       []CanonicalCase `json:"cases"`
	Tools       []ToolLock      `json:"tools"`
	LiveLedger  ToolLock        `json:"live_ledger"`
}

type SemanticIR struct {
	Schema            string          `json:"schema"`
	Version           string          `json:"version"`
	Scenario          string          `json:"scenario"`
	Mode              string          `json:"mode"`
	SourceDigest      string          `json:"source_digest"`
	MetaDigest        string          `json:"meta_digest"`
	ContractDigest    string          `json:"contract_digest"`
	BaselineRule      string          `json:"baseline_rule"`
	CandidateRule     string          `json:"candidate_rule"`
	FrontierID        string          `json:"frontier_id"`
	FrontierClass     string          `json:"frontier_class"`
	ActionID          string          `json:"action_id"`
	ActionAuthority   string          `json:"action_authority"`
	ParentProofState  string          `json:"parent_proof_state"`
	ParentProofDigest string          `json:"parent_proof_digest"`
	CounterexampleInput int           `json:"counterexample_input"`
	CounterexampleExpected string     `json:"counterexample_expected"`
	CounterexampleLabel string        `json:"counterexample_label"`
	EditSurface       string          `json:"edit_surface"`
	SemanticRoot      string          `json:"semantic_root"`
	TestImpacts       []string        `json:"test_impacts"`
	Stages            []StageDecl     `json:"stages"`
	Edges             []EdgeDecl      `json:"causal_edges"`
	Tests             []TestImpactDecl `json:"semantic_tests"`
	Cases             []CanonicalCase `json:"canonical_cases"`
	Tools             []ToolLock      `json:"released_tools"`
	Authority         Authority       `json:"authority"`
	IRDigest          string          `json:"ir_digest"`
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

type PreparedChange struct {
	Schema              string    `json:"schema"`
	Scenario            string    `json:"scenario"`
	Mode                string    `json:"mode"`
	SourceDigest        string    `json:"source_digest"`
	MetaDigest          string    `json:"meta_digest"`
	ContractDigest      string    `json:"contract_digest"`
	IRDigest            string    `json:"ir_digest"`
	Frontier            string    `json:"actionable_frontier"`
	Action              string    `json:"unique_action"`
	EditSurface         string    `json:"edit_surface"`
	CandidateSource     string    `json:"candidate_source"`
	GeneratedGo         string    `json:"generated_go"`
	CandidateDigest     string    `json:"candidate_digest"`
	CallerTempOnly      bool      `json:"caller_temp_only"`
	RepositoryWrites    int       `json:"repository_writes"`
	ParentProofDigest   string    `json:"parent_proof_digest"`
	ImpactedTests       []string  `json:"semantic_impacted_tests"`
	Claim               Claim     `json:"claim"`
}

type RuntimeObservation struct {
	Variant  string `json:"variant"`
	Input    int    `json:"input"`
	Verdict  string `json:"verdict"`
	Accepted bool   `json:"accepted"`
}

type IndicatorPair struct {
	ID             string `json:"id"`
	Unit           string `json:"unit"`
	Baseline       int64  `json:"baseline"`
	Candidate      int64  `json:"candidate"`
	SameJob        bool   `json:"same_job"`
	ExplicitBudget bool   `json:"explicit_budget"`
}

type StageMeasurement struct {
	Stage       string `json:"stage"`
	WallMS      int64  `json:"wall_ms"`
	PeakRSSKiB  int64  `json:"peak_rss_kib"`
}

type LiveLedgerObservation struct {
	Lock              ToolLock `json:"lock"`
	ActionableFrontier string   `json:"actionable_frontier"`
	AutomationCanProduce bool   `json:"automation_can_produce"`
	ExternalEvidence   bool     `json:"external_evidence"`
	Decision           string   `json:"decision"`
}

type ExecutionInput struct {
	Schema             string                         `json:"schema"`
	Scenario           string                         `json:"scenario"`
	Mode               string                         `json:"mode"`
	SourceDigest       string                         `json:"source_digest"`
	MetaDigest         string                         `json:"meta_digest"`
	ContractDigest     string                         `json:"contract_digest"`
	IRDigest           string                         `json:"ir_digest"`
	CandidateDigest    string                         `json:"candidate_digest"`
	Toolchain          string                         `json:"toolchain"`
	Runner             string                         `json:"runner"`
	Observations       []RuntimeObservation            `json:"observations"`
	BeforeAfter        map[string]int64               `json:"before_after"`
	StageMeasurements  map[string]StageMeasurement    `json:"stage_measurements"`
	Indicators         []IndicatorPair                `json:"indicators"`
	SameScope          bool                           `json:"same_scope"`
	SameJob            bool                           `json:"same_job"`
	PositiveWork       bool                           `json:"positive_work"`
	RepositoryWrites   int                            `json:"repository_writes"`
	RemoteWrites       int                            `json:"remote_writes"`
	LocalTestExecutions int                           `json:"local_test_executions"`
	CrossProjectRequiredGates int                     `json:"cross_project_required_gates"`
	IntegrationState   string                         `json:"integration_state"`
	LiveLedger         *LiveLedgerObservation         `json:"live_ledger,omitempty"`
}

type CaseResult struct {
	Ordinal       int      `json:"ordinal"`
	ID            string   `json:"id"`
	ExpectedState string   `json:"expected_state"`
	State         string   `json:"state"`
	Probe         string   `json:"probe"`
	Fixture       string   `json:"fixture"`
	SemanticEdge  string   `json:"semantic_edge"`
	Claim         Claim    `json:"claim"`
}

type Vector struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}
