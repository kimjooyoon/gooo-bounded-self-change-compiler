package cycle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseMeta(path string) (MetaDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MetaDecl{}, err
	}
	meta := MetaDecl{MetaDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	line := 0
	for scanner.Scan() {
		line++
		fields := fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "gooo" {
			if len(fields) != 3 || fields[1] != "bounded_self_change_cycle_meta" || fields[2] != "v2" {
				return MetaDecl{}, fmt.Errorf("line %d: invalid v2 meta header", line)
			}
			meta.Schema, meta.Version = MetaSchema, fields[2]
			continue
		}
		values, parseErr := keyValues(fields[1:])
		if parseErr != nil {
			return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
		}
		switch fields[0] {
		case "semantic_owner":
			meta.Owner = values["owner"]
		case "precedence":
			meta.Precedence = strings.Split(values["order"], ">")
		case "unknown_fields":
			meta.UnknownFields = strings.Split(values["fields"], ",")
		case "authority":
			meta.Authority, err = parseAuthority(values)
		case "live_action":
			meta.LiveFrontier, meta.LiveAction = values["frontier"], values["action"]
		case "stage":
			ordinal, parseErr := integer(values, "ordinal")
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
			}
			meta.Stages = append(meta.Stages, StageDecl{Ordinal: ordinal, ID: values["id"], Input: values["input"], Output: values["output"], SemanticEdge: values["edge"]})
		case "edge":
			ordinal, parseErr := integer(values, "ordinal")
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
			}
			meta.Edges = append(meta.Edges, EdgeDecl{Ordinal: ordinal, From: values["from"], To: values["to"], Relation: values["relation"]})
		case "test":
			meta.Tests = append(meta.Tests, TestImpactDecl{ID: values["id"], Root: values["root"], Edge: values["edge"]})
		case "case":
			canonical, parseErr := parseCase(values)
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
			}
			meta.Cases = append(meta.Cases, canonical)
		case "invariant":
			invariant, parseErr := parseInvariant(values)
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
			}
			meta.Invariants = append(meta.Invariants, invariant)
		case "tool":
			tool, parseErr := parseTool(values)
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
			}
			meta.Tools = append(meta.Tools, tool)
		default:
			return MetaDecl{}, fmt.Errorf("line %d: unknown v2 meta declaration %q", line, fields[0])
		}
		if err != nil {
			return MetaDecl{}, fmt.Errorf("line %d: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return MetaDecl{}, err
	}
	return meta, nil
}

func ParseSource(path string) (SourceDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceDecl{}, err
	}
	source := SourceDecl{SourceDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	line := 0
	for scanner.Scan() {
		line++
		fields := fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "gooo" {
			if len(fields) != 3 || fields[1] != "bounded_self_change_program" || fields[2] != "v2" {
				return SourceDecl{}, fmt.Errorf("line %d: invalid v2 source header", line)
			}
			source.Schema, source.Version = SourceSchema, fields[2]
			continue
		}
		if fields[0] == "mode" {
			if len(fields) != 2 || (fields[1] != "internal" && fields[1] != "live") {
				return SourceDecl{}, fmt.Errorf("line %d: invalid source mode", line)
			}
			source.Mode = fields[1]
			continue
		}
		values, parseErr := keyValues(fields[1:])
		if parseErr != nil {
			return SourceDecl{}, fmt.Errorf("line %d: %w", line, parseErr)
		}
		switch fields[0] {
		case "scenario":
			source.Scenario = values["id"]
		case "baseline":
			source.BaselineRule = values["rule"]
		case "candidate":
			source.CandidateRule = values["rule"]
		case "frontier":
			source.FrontierID, source.FrontierClass = values["id"], values["class"]
			source.FrontierActionable, err = boolean(values, "actionable")
			if err == nil {
				source.FrontierUnique, err = boolean(values, "unique")
			}
		case "action":
			source.ActionID, source.ActionAuthority = values["id"], values["authority"]
		case "parent_proof":
			source.ParentProofState, source.ParentProofDigest = values["state"], values["digest"]
		case "counterexample":
			source.CounterexampleInput, err = integer(values, "input")
			if err == nil {
				source.CounterexampleExpected, source.CounterexampleLabel = values["expected"], values["label"]
			}
		case "edit_surface":
			source.EditSurface = values["field"] + ":" + values["operator"] + ":" + values["constraint"]
		case "semantic_root":
			source.SemanticRoot = values["id"]
		case "test_impact":
			source.TestImpacts = splitList(values["ids"])
		case "counterexample_next_wave":
			source.CounterexampleNextWave, err = boolean(values, "accepted")
			if err == nil {
				source.RepositoryAuthority, err = integer(values, "repository_authority")
			}
		case "evidence_scope":
			source.EvidenceScope = values["source"] + ":" + values["meta"] + ":" + values["contract"] + ":" + values["runner"]
		case "authority":
			source.Authority, err = parseAuthority(values)
		default:
			return SourceDecl{}, fmt.Errorf("line %d: unknown v2 source declaration %q", line, fields[0])
		}
		if err != nil {
			return SourceDecl{}, fmt.Errorf("line %d: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceDecl{}, err
	}
	return source, nil
}

func LoadContract(path string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, fmt.Errorf("decode v2 contract: %w", err)
	}
	return contract, nil
}

func LoadExecution(path string) (ExecutionInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ExecutionInput{}, err
	}
	var execution ExecutionInput
	if err := json.Unmarshal(data, &execution); err != nil {
		return ExecutionInput{}, fmt.Errorf("decode v2 execution: %w", err)
	}
	return execution, nil
}

func fields(line string) []string {
	line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
	line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
	if line == "" {
		return nil
	}
	return strings.Fields(line)
}

func keyValues(items []string) (map[string]string, error) {
	values := make(map[string]string, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid key/value %q", item)
		}
		if _, exists := values[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate key %q", parts[0])
		}
		values[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return values, nil
}

func integer(values map[string]string, key string) (int, error) {
	value := values[key]
	number, err := strconv.Atoi(value)
	if value == "" || err != nil {
		return 0, fmt.Errorf("invalid integer %s=%q", key, value)
	}
	return number, nil
}

func int64Value(values map[string]string, key string) (int64, error) {
	value := values[key]
	number, err := strconv.ParseInt(value, 10, 64)
	if value == "" || err != nil {
		return 0, fmt.Errorf("invalid int64 %s=%q", key, value)
	}
	return number, nil
}

func boolean(values map[string]string, key string) (bool, error) {
	value, err := strconv.ParseBool(values[key])
	if err != nil {
		return false, fmt.Errorf("invalid boolean %s=%q", key, values[key])
	}
	return value, nil
}

func parseAuthority(values map[string]string) (Authority, error) {
	keys := []string{"repository_writes", "remote_writes", "apply_authority", "commit_authority", "merge_authority", "tag_authority", "release_authority", "local_test_executions", "cross_project_required_gates"}
	parsed := make([]int, len(keys))
	for i, key := range keys {
		value, err := integer(values, key)
		if err != nil {
			return Authority{}, err
		}
		parsed[i] = value
	}
	return Authority{RepositoryWrites: parsed[0], RemoteWrites: parsed[1], ApplyAuthority: parsed[2], CommitAuthority: parsed[3], MergeAuthority: parsed[4], TagAuthority: parsed[5], ReleaseAuthority: parsed[6], LocalTestExecutions: parsed[7], CrossProjectRequiredGates: parsed[8]}, nil
}

func parseCase(values map[string]string) (CanonicalCase, error) {
	ordinal, err := integer(values, "ordinal")
	if err != nil {
		return CanonicalCase{}, err
	}
	return CanonicalCase{Ordinal: ordinal, ID: values["id"], ExpectedState: values["expected"], Probe: values["probe"], Fixture: values["fixture"], SemanticEdge: values["edge"], DependsOn: splitList(values["depends_on"]), Reason: values["reason"], ProofChoice: values["proof_choice"], IndicatorClass: values["indicator_class"]}, nil
}

func parseInvariant(values map[string]string) (InvariantDecl, error) {
	ordinal, err := integer(values, "ordinal")
	if err != nil {
		return InvariantDecl{}, err
	}
	return InvariantDecl{
		Ordinal: ordinal,
		ID: values["id"],
		Activity: values["activity"],
		Stage: values["stage"],
		Step: values["step"],
		ProofChoice: values["proof_choice"],
		IndicatorClass: values["indicator_class"],
		DependsOn: splitList(values["depends_on"]),
	}, nil
}

func parseTool(values map[string]string) (ToolLock, error) {
	releaseID, err := int64Value(values, "release_id")
	if err != nil {
		return ToolLock{}, err
	}
	assetID, err := int64Value(values, "asset_id")
	if err != nil {
		return ToolLock{}, err
	}
	assetSize, err := int64Value(values, "asset_size_bytes")
	if err != nil {
		return ToolLock{}, err
	}
	immutable, err := boolean(values, "immutable")
	if err != nil {
		return ToolLock{}, err
	}
	return ToolLock{ID: values["id"], Repository: values["repository"], Tag: values["tag"], ReleaseID: releaseID, Immutable: immutable, TagObjectSHA: values["tag_object_sha"], TargetSHA: values["target_sha"], AssetName: values["asset_name"], AssetID: assetID, AssetSize: assetSize, AssetDigest: values["asset_digest"]}, nil
}

func splitList(value string) []string {
	if value == "" || value == "-" {
		return []string{}
	}
	return strings.Split(value, ",")
}
