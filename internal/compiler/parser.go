package compiler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseSource(path string) (SourceDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceDecl{}, err
	}
	decl := SourceDecl{SourceDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fields := declarationFields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		values, err := parseKeyValues(fields[1:])
		if err != nil && fields[0] != "gooo" {
			return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		switch fields[0] {
		case "gooo":
			if len(fields) != 3 || fields[1] != "bounded_self_change" || fields[2] != "v1" {
				return SourceDecl{}, fmt.Errorf("line %d: invalid source header", lineNumber)
			}
			decl.Schema = SourceSchema
			decl.Version = fields[2]
		case "scenario":
			decl.Scenario = required(values, "id")
		case "baseline":
			decl.Baseline = BaselineDecl{Rule: required(values, "rule")}
		case "counterexample":
			input, parseErr := parseInt(values, "input")
			if parseErr != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, parseErr)
			}
			decl.Counterexample = CounterexampleDecl{Input: input, Expected: required(values, "expected"), Label: required(values, "label"), Present: true}
		case "improvement_intent":
			decl.Intent = IntentDecl{Name: required(values, "name")}
		case "edit_surface":
			decl.EditSurface = EditSurfaceDecl{Field: required(values, "field"), Operator: required(values, "operator"), Constraint: required(values, "constraint")}
		case "invariant":
			decl.Invariants = append(decl.Invariants, InvariantDecl{ID: required(values, "id"), Expression: required(values, "expression")})
		case "guardrail":
			input, parseErr := parseInt(values, "input")
			if parseErr != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, parseErr)
			}
			decl.Guardrails = append(decl.Guardrails, GuardrailDecl{ID: required(values, "id"), Input: input, Expected: required(values, "expected"), Expression: required(values, "expression")})
		case "evidence":
			requiredValue := required(values, "required")
			requiredBool, parseErr := strconv.ParseBool(requiredValue)
			if parseErr != nil {
				return SourceDecl{}, fmt.Errorf("line %d: invalid required boolean", lineNumber)
			}
			decl.Evidence = append(decl.Evidence, EvidenceDecl{ID: required(values, "id"), Required: requiredBool, State: required(values, "state")})
		case "authority":
			decl.Authority, err = parseAuthority(values)
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
		case "precedence", "unknown_fields":
			return SourceDecl{}, fmt.Errorf("line %d: %s belongs in meta source", lineNumber, fields[0])
		default:
			return SourceDecl{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceDecl{}, err
	}
	return decl, nil
}

func ParseMeta(path string) (MetaDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MetaDecl{}, err
	}
	decl := MetaDecl{MetaDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fields := declarationFields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "gooo" {
			if len(fields) != 3 || fields[1] != "bounded_self_change_meta" || fields[2] != "v1" {
				return MetaDecl{}, fmt.Errorf("line %d: invalid meta header", lineNumber)
			}
			decl.Schema = MetaSchema
			decl.Version = fields[2]
			continue
		}
		values, err := parseKeyValues(fields[1:])
		if err != nil {
			return MetaDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		switch fields[0] {
		case "semantic_owner":
			decl.Owner = required(values, "owner")
		case "rule":
			decl.Rules = append(decl.Rules, RuleSpec{Name: required(values, "name"), Predicate: required(values, "predicate")})
		case "precedence":
			decl.Precedence = strings.Split(required(values, "order"), ">")
		case "unknown_fields":
			decl.UnknownFields = strings.Split(required(values, "fields"), ",")
		case "case":
			ordinal, parseErr := parseInt(values, "ordinal")
			if parseErr != nil {
				return MetaDecl{}, fmt.Errorf("line %d: %w", lineNumber, parseErr)
			}
			decl.Cases = append(decl.Cases, CanonicalCase{
				Ordinal:       ordinal,
				ID:            required(values, "id"),
				ExpectedState: required(values, "expected"),
				Probe:         required(values, "probe"),
				Fixture:       required(values, "fixture"),
				SemanticEdge:  required(values, "edge"),
				DependsOn:     splitList(required(values, "depends_on")),
				Reason:        required(values, "reason"),
			})
		default:
			return MetaDecl{}, fmt.Errorf("line %d: unknown meta declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return MetaDecl{}, err
	}
	return decl, nil
}

func LoadContract(path string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	return contract, nil
}

func ContractDigest(contract Contract) (string, error) {
	return DigestValue(contract)
}

func declarationFields(line string) []string {
	line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
	line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
	if line == "" {
		return nil
	}
	return strings.Fields(line)
}

func parseKeyValues(fields []string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid key/value %q", field)
		}
		if _, exists := values[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate key %q", parts[0])
		}
		values[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return values, nil
}

func required(values map[string]string, key string) string {
	return values[key]
}

func parseInt(values map[string]string, key string) (int, error) {
	value, ok := values[key]
	if !ok || value == "" {
		return 0, fmt.Errorf("missing %s", key)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", key, value)
	}
	return n, nil
}

func parseAuthority(values map[string]string) (Authority, error) {
	keys := []string{"repository_writes", "apply_authority", "commit_authority", "merge_authority", "tag_authority", "release_authority", "local_test_executions", "cross_project_required_gates"}
	parsed := make([]int, len(keys))
	for index, key := range keys {
		value, err := parseInt(values, key)
		if err != nil {
			return Authority{}, err
		}
		parsed[index] = value
	}
	return Authority{
		RepositoryWrites:          parsed[0],
		ApplyAuthority:            parsed[1],
		CommitAuthority:           parsed[2],
		MergeAuthority:            parsed[3],
		TagAuthority:              parsed[4],
		ReleaseAuthority:          parsed[5],
		LocalTestExecutions:       parsed[6],
		CrossProjectRequiredGates: parsed[7],
	}, nil
}

func splitList(value string) []string {
	if value == "-" || value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
