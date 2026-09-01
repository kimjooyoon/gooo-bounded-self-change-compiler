package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-bounded-self-change-compiler/internal/compiler"
)

func main() {
	if len(os.Args) < 2 {
		fatal("command is required: run or verify")
	}
	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		fatal(fmt.Sprintf("unknown command %q", os.Args[1]))
	}
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
