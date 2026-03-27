# AID Greenfield Strategy

**What we built, what we proved, and where this goes.**

---

## What we built in one session

### The spec (complete)
- **format.md** — 13 sections, ~1100 lines. Four tiers: header, entries, module annotations, workflows.
- **fields.md** — 92 fields across all entry types, manifests, and annotations.
- **design-rationale.md** — 31 rationale entries, every design decision explained.
- **generation.md** — 3-layer pipeline (extract → AI infer → human review), per-language mappings.
- **layer2.md** — Source-linked claims, generator→reviewer multi-agent pipeline, versioning, staleness detection.

### The tooling (functional)

| Tool | Language | What it does |
|------|----------|-------------|
| `aid-gen` | Python | L1 extractor for Python (99 tests) |
| `aid-gen-go` | Go | L1 extractor for Go |
| `aid-gen-ts` | Go + Node.js | L1 extractor for TypeScript (.d.ts priority) |
| `aid-gen-cs` | Go + .NET/Roslyn | L1 extractor for C# |
| `aid-parse` | Go | Parse .aid files → JSON, `--summary` for annotations only |
| `aid-validate` | Go | 8 validation rules (45 tests across the toolkit) |
| `aid-discover` | Go | Find nearest .aidocs/ per spec discovery protocol |
| `aid-manifest-gen` | Go | Auto-generate manifest.aid from .aidocs/ contents |
| `aid-gen-l2 generate` | Go | Build L2 generator prompt from L1 + source |
| `aid-gen-l2 review` | Go | Build L2 reviewer prompt from draft + source refs |
| `aid-gen-l2 stale` | Go | Git-based staleness detection for [src:] references |
| `aid-gen-l2 update` | Go | Incremental update prompt for stale claims only |

### The benchmarks (validated)

| Benchmark | Codebase | Task | Key finding |
|-----------|----------|------|-------------|
| BM1 | Aria compiler (10K LoC Go) | Add spawn statement | AID: -47% tokens, 6x faster, same quality |
| BM2 | SyndrDB (219K LoC Go) | BRIN planner integration (L1 only) | AID: better architecture, but +24% tokens (loaded all 69 files) |
| BM3 | SyndrDB (219K LoC Go) | BRIN planner integration (L1+L2) | AID: **5x less code, 6 components reused, -10% tokens** |
| BM4 | SyndrDB flusher (single file) | Fix pending request accumulation | Full AID: +68% tokens. Slim AID: **-12% tokens, same quality** |

### The L2 pipeline (proven)

Ran the generator→reviewer pipeline on 4 SyndrDB packages:
- **Generator accuracy:** 83% first-pass (48/58 claims verified correct)
- **Reviewer catch rate:** 100% of inaccuracies caught, 4 missing claims added
- **Pipeline time:** ~16 minutes for a 26K LoC package
- **Bugs found by L2 generators:** totalDocsPending never decrements (flusher), stats race under RLock (brinindex)

---

## What we proved

### 1. AID makes AI agents write better code on novel codebases

Not marginally better — **fundamentally different solutions.** BM3 showed an agent without AID writing ~100 lines of custom filtering logic, while the agent with AID wrote ~20 lines reusing 6 existing components. Same bugs found, completely different (and better) architecture.

### 2. The value scales with task complexity

| Task type | AID effect |
|-----------|-----------|
| Pattern-following (BM1) | Efficiency only — same quality, fewer tokens |
| Cross-package feature (BM3) | **Quality AND efficiency** — the decisive win |
| Single-file fix (BM4) | Quality edge (context awareness), efficiency depends on loading strategy |

### 3. Selective loading is critical

Loading all AID files for a large project hurts more than it helps (BM2: +24% tokens). The fix:
- **Manifest** with `@purpose` and `@key_risks` per package for quick orientation
- **`@depends`** chain for loading only relevant packages
- **`--summary`** mode for single-file tasks (annotations only, skip per-entry detail)
- Match context depth to task scope: manifest → summary → full L2

### 4. Layer 2 is where the real value lives

Layer 1 (mechanical extraction) is table stakes — it gives you the API surface. Layer 2 (source-linked semantic docs) gives you what code alone can't express:
- **Workflows:** how data flows through the system
- **Invariants:** constraints that span multiple components
- **Antipatterns:** mistakes that aren't obvious from the API
- **Decisions:** why the code is structured this way (prevents "improvements" that break things)

### 5. The generator→reviewer pipeline works

Two separate AI agents — one creative (infers semantics), one critical (verifies claims) — with source references enabling mechanical verification. 83% accuracy on first pass, 100% after review. No human in the loop needed for correctness; human review is for critical systems where the stakes justify it.

---

## Where this goes

### Near-term (ready to use today)

1. **Adopt on a real project.** Generate L1+L2 AID for a project you work on daily. Use it for a week. See if it changes how your AI assistant handles tasks.

2. **Add to CI.** Run `aid-gen-l2 stale` in CI to catch when code changes invalidate AID claims. Run `aid-validate` to catch spec violations. Treat AID like code — committed, reviewed, tested.

3. **Publish the spec.** The format is stable enough for others to implement. The README has usage instructions for Claude Code, Cursor, Copilot, and raw API.

### Medium-term (next month)

4. **Consolidate extractors into one binary.** Currently: Python tool, Go tool, TS tool, CS tool. Should be: `aid-gen --lang go ./src/...` — one Go binary that handles all languages. The Go extractor uses native `go/ast`; TS and C# shell out to their respective compilers; Python could shell out to a bundled Python script.

5. **Auto-L2 in Claude Code hooks.** A Claude Code hook that runs after every commit: detect changed packages → check L2 staleness → re-generate stale claims → commit updated AID. Fully automated living documentation.

6. **Community generators.** Rust extractor (via `syn` crate → JSON → Go). Java extractor (via reflection or `javap` → JSON → Go). Same hybrid pattern, new languages.

### Long-term (vision)

7. **AID as the standard documentation format for AI agents.** Every library ships with `.aidocs/` the way they ship with README and API docs. Package managers index AID files for dependency documentation. AI agents load AID before source — it's their first read, not their last resort.

8. **AID-aware AI assistants.** The assistant doesn't just read AID — it maintains it. Every code change triggers staleness detection. Every new feature gets a workflow. Every bug fix gets an antipattern. The docs are always current because the agent that writes the code also writes the docs, and a separate agent verifies them.

9. **Cross-project AID.** When your project depends on a library, you load that library's AID from a registry — not just its source. The agent understands your dependencies' contracts, invariants, and antipatterns without reading their source code. Dependency management for documentation.

---

## The core insight

**AI agents can't reuse what they don't know exists.**

Every benchmark confirmed this. The no-docs agent found the same bugs, understood the same requirements, and wrote correct code. But it built from scratch — every time. The AID agent built on what was already there.

That's the difference between a contractor who's never seen your codebase and a senior developer who's been on the team for a year. AID gives the AI agent the senior developer's knowledge in a format that costs a fraction of the tokens it would take to acquire that knowledge by reading source code.

The format is simple. The tooling works. The benchmarks validate it. The next step is using it.
