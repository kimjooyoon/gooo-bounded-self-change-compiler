# Gooo bounded self-change compiler

This repository implements the smallest useful loop for a Gooo program to
propose a bounded semantic change from an observed counterexample and an
improvement intent. The `.gooo` declarations own the semantics and meta
activity. Go is only the parser, semantic-IR/graph generator, evaluator, and
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

The denominator is fixed at nine canonical cases: three CLOSED, three
UNKNOWN, and three REFUTED. Resolution is always `REFUTED > UNKNOWN > CLOSED`.
UNKNOWN claims carry `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. No score or percentage is emitted.

## CI-only validation

The GitHub Actions workflow is the validation authority and uses Go 1.27. It
records wall time, peak RSS, test reuse/failure/unknown counts, repository
inventory, generated artifact counts and bytes, provenance digests, and the
zero-authority runtime receipt. The local workflow intentionally does not run
test, build, vet, lint, formatting, check, or conformance commands.

The executable interface used by CI is:

```text
gooo-bounded-self-change-compiler run --meta ... --source ... --contract ... --out ...
gooo-bounded-self-change-compiler verify --meta ... --source ... --contract ... --out ... --observations ...
```
