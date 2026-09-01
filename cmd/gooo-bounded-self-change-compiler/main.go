package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-bounded-self-change-compiler/internal/compiler"
	"github.com/kimjooyoon/gooo-bounded-self-change-compiler/internal/cycle"
)

func main() {
	if len(os.Args) < 2 {
		fatal("command is required: run, verify, or cycle")
	}
	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	case "cycle":
		cycleCommand(os.Args[2:])
	default:
		fatal(fmt.Sprintf("unknown command %q", os.Args[1]))
	}
}

func cycleCommand(args []string) {
	if len(args) < 1 {
		fatal("cycle subcommand is required: prepare or finalize")
	}
	switch args[0] {
	case "prepare":
		cyclePrepare(args[1:])
	case "finalize":
		cycleFinalize(args[1:])
	default:
		fatal(fmt.Sprintf("unknown cycle subcommand %q", args[0]))
	}
}

func cyclePrepare(args []string) {
	set := flag.NewFlagSet("cycle prepare", flag.ExitOnError)
	metaPath := set.String("meta", "", "path to v2 .gooo semantic meta source")
	sourcePath := set.String("source", "", "path to v2 .gooo scenario source")
	contractPath := set.String("contract", "", "path to v2 immutable lock contract")
	mode := set.String("mode", "internal", "internal or live")
	outputDir := set.String("out", "", "caller-owned temporary preparation directory")
	set.Parse(args)
	if *metaPath == "" || *sourcePath == "" || *contractPath == "" || *outputDir == "" {
		fatal("cycle prepare requires --meta, --source, --contract, and --out")
	}
	meta, err := cycle.ParseMeta(*metaPath)
	if err != nil { fatal(err.Error()) }
	source, err := cycle.ParseSource(*sourcePath)
	if err != nil { fatal(err.Error()) }
	contract, err := cycle.LoadContract(*contractPath)
	if err != nil { fatal(err.Error()) }
	prepared, ir, err := cycle.Prepare(meta, source, contract, *mode, *outputDir)
	if err != nil { fatal(err.Error()) }
	data, err := json.Marshal(map[string]any{"decision": prepared.Claim.State, "mode": *mode, "source_digest": ir.SourceDigest, "meta_digest": ir.MetaDigest, "contract_digest": ir.ContractDigest, "ir_digest": ir.IRDigest, "candidate_digest": prepared.CandidateDigest})
	if err != nil { fatal(err.Error()) }
	fmt.Println(string(data))
}

func cycleFinalize(args []string) {
	set := flag.NewFlagSet("cycle finalize", flag.ExitOnError)
	metaPath := set.String("meta", "", "path to v2 .gooo semantic meta source")
	sourcePath := set.String("source", "", "path to v2 .gooo scenario source")
	contractPath := set.String("contract", "", "path to v2 immutable lock contract")
	mode := set.String("mode", "internal", "internal or live")
	preparedPath := set.String("prepared", "", "prepared-change-proposal.json")
	executionPath := set.String("execution", "", "same-job execution evidence JSON")
	metricsDir := set.String("metrics", "", "stage measurement directory")
	integrationPath := set.String("integration", "", "caller-owned integration receipt")
	outputDir := set.String("out", "", "caller-owned final output directory")
	set.Parse(args)
	if *metaPath == "" || *sourcePath == "" || *contractPath == "" || *preparedPath == "" || *executionPath == "" || *outputDir == "" {
		fatal("cycle finalize requires --meta, --source, --contract, --prepared, --execution, and --out")
	}
	meta, err := cycle.ParseMeta(*metaPath)
	if err != nil { fatal(err.Error()) }
	source, err := cycle.ParseSource(*sourcePath)
	if err != nil { fatal(err.Error()) }
	contract, err := cycle.LoadContract(*contractPath)
	if err != nil { fatal(err.Error()) }
	manifest, err := cycle.Finalize(meta, source, contract, *mode, *preparedPath, *executionPath, *metricsDir, *integrationPath, *outputDir)
	if err != nil { fatal(err.Error()) }
	data, err := json.Marshal(manifest)
	if err != nil { fatal(err.Error()) }
	fmt.Println(string(data))
}

func run(args []string) {
	set := flag.NewFlagSet("run", flag.ExitOnError)
	metaPath := set.String("meta", "", "path to .gooo semantic meta source")
	sourcePath := set.String("source", "", "path to .gooo scenario source")
	contractPath := set.String("contract", "", "path to fixed denominator contract")
	outputDir := set.String("out", "", "caller-owned temporary output directory")
	set.Parse(args)
	if *metaPath == "" || *sourcePath == "" || *contractPath == "" || *outputDir == "" {
		fatal("run requires --meta, --source, --contract, and --out")
	}
	meta, err := compiler.ParseMeta(*metaPath)
	if err != nil {
		fatal(err.Error())
	}
	source, err := compiler.ParseSource(*sourcePath)
	if err != nil {
		fatal(err.Error())
	}
	contract, err := compiler.LoadContract(*contractPath)
	if err != nil {
		fatal(err.Error())
	}
	ir, err := compiler.Generate(meta, source, contract, *outputDir)
	if err != nil {
		fatal(err.Error())
	}
	data, err := json.Marshal(map[string]any{"decision": "CANDIDATE_GENERATED", "ir_digest": ir.IRDigest, "candidate_rule": ir.CandidateRule})
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func verify(args []string) {
	set := flag.NewFlagSet("verify", flag.ExitOnError)
	metaPath := set.String("meta", "", "path to .gooo semantic meta source")
	sourcePath := set.String("source", "", "path to .gooo scenario source")
	contractPath := set.String("contract", "", "path to fixed denominator contract")
	outputDir := set.String("out", "", "caller-owned generated artifact directory")
	observationsPath := set.String("observations", "", "execution evidence JSON")
	set.Parse(args)
	if *metaPath == "" || *sourcePath == "" || *contractPath == "" || *outputDir == "" || *observationsPath == "" {
		fatal("verify requires --meta, --source, --contract, --out, and --observations")
	}
	report, err := compiler.Verify(*metaPath, *sourcePath, *contractPath, *outputDir, *observationsPath)
	if err != nil {
		fatal(err.Error())
	}
	data, err := json.Marshal(report.Summary)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
