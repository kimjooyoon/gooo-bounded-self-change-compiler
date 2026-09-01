# Gooo bounded self-change compiler

This repository implements a bounded Gooo self-improvement cycle. The `.gooo`
declarations own semantics, meta stages, causal edges, authority, and
judgments. Go is only the parser, semantic-IR generator, evaluator, and
ephemeral candidate runtime.

The fixed end-to-end path is:

```text
.gooo source → semantic IR/graph → candidate .gooo + Go + patch proposal
→ ephemeral compile/run → verifier → human decision dossier
```

The example changes `input > 0` to `input >= 0` so the observed zero input is
accepted while the fixed negative guardrail remains rejected. The product
never edits, commits, merges, tags, or releases the input repository. All
generated artifacts must be written to an empty caller-owned directory outside
the repository.

The preserved v0.1 path remains available as `run` and `verify`, with its old
failed and immutable release history untouched. The v0.2 path is:

```text
immutable ledger → causal frontier → unique action → bounded change
→ caller temp only → semantic test impact → covered-stage measurement
→ per-indicator judgment → content-addressed evidence → next-wave proposal
```

The original v0.2.0 release remains an immutable historical release. The
authoritative release for the named-invariant contract is v0.2.1 and is
created only from a successful `main` Actions run.

The v0.2 denominator is fixed at exactly 12 canonical cases: 4 CLOSED, 4
UNKNOWN, and 4 REFUTED. Resolution is always `REFUTED > UNKNOWN > CLOSED`.
UNKNOWN claims always carry `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. No score, percentage, average, or
aggregate utility is emitted.

The same `.gooo` meta contract declares exactly 12 named invariants. Their
proof families are fixed at 4 FOUNDATION, 4 COHERENCE, and 4 REGRESSION; their
indicator classes are fixed at 4 DRIVER, 4 OUTCOME, and 4 GUARDRAIL. The
invariant declarations are bound one-to-one to the canonical cases and are
carried into the cycle manifest, evidence manifest, and human report.

Five released projector inputs are locked by immutable release, annotated tag,
asset ID, size, and digest in both the v2 `.gooo` meta source and
`contracts/v0.2-cycle-locks.json`. The optional live observation locks
`gooo-self-improvement-ledger` v0.50.0 the same way. Its frontier is
`EXTERNAL_UTILITY_EVIDENCE`; automation stops at
`UNKNOWN / HUMAN_EXTERNAL_EVIDENCE_REQUIRED` and never edits source.

The v0.2 final cycle output has exactly these eight files:

```text
cycle-manifest.json
frontier-receipt.json
change-proposal.json
test-impact-receipt.json
measurement-receipt.json
evidence-manifest.json
next-wave-proposal.json
human-report.md
```

## CI-only validation

The GitHub Actions workflow is the validation authority and uses Go 1.27. It
records wall time, peak RSS, test reuse/failure/unknown counts, repository
inventory, generated artifact counts and bytes, provenance digests, and the
zero-authority runtime receipt. The local workflow intentionally does not run
test, build, vet, lint, formatting, check, or conformance commands.

The preserved v0.1 interface is:

```text
gooo-bounded-self-change-compiler run --meta ... --source ... --contract ... --out ...
gooo-bounded-self-change-compiler verify --meta ... --source ... --contract ... --out ... --observations ...
```

The v0.2 interface is:

```text
gooo-bounded-self-change-compiler cycle prepare --meta ... --source ... --contract ... --mode internal --out ...
gooo-bounded-self-change-compiler cycle finalize --meta ... --source ... --contract ... --prepared ... --execution ... --metrics ... --integration ... --out ...
```
