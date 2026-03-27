# AID — Agent Interface Document

**A documentation format designed for AI agents, not humans.**

AID is a structured, token-efficient documentation format that gives AI coding agents exactly what they need to use an API correctly on the first try: complete type signatures, exhaustive error conditions, parameter constraints, postconditions, thread safety guarantees, and workflow patterns.

Current documentation formats were designed for humans to browse. AI agents pay for every token they read. AID eliminates the prose, makes every constraint explicit, and introduces workflow blocks that describe how APIs fit together — something no existing doc format captures.

## Why AID exists

AI coding agents today spend 30-40% of their tokens on context discovery — reading documentation, searching for API signatures, figuring out error cases. Most of that is wasted on prose they don't need. When they miss a constraint or an error case buried in a paragraph, they generate bugs.

AID fixes this by providing:

- **Complete contracts** — every function's full behavior in ~100 tokens instead of ~800
- **Explicit constraints** — parameter ranges, preconditions, postconditions, invariants
- **Exhaustive errors** — every error variant and the exact condition that triggers it
- **Workflow patterns** — how APIs work together as sequences, not just individually
- **Zero prose** — every piece of information has a named field with defined semantics

## Quick example

```
@module http/client
@lang python
@version 2.31.0
@stability stable
@purpose HTTP client for making web requests

---

@fn get
@purpose Perform an HTTP GET request
@sig (url: str, opts?: RequestOpts) -> Response ! HttpError | TimeoutError
@params
  url: Full URL with scheme. Must match ^https?://. Required.
  opts: Request configuration.
    .timeout: Duration. Default 30s. Must be > 0.
    .redirects: int. Default 5. Range [0, 20].
@returns Response with status, headers, and body stream
@errors
  HttpError.DnsFailure — url domain cannot be resolved
  HttpError.ConnectionRefused — server not accepting connections
  TimeoutError — no response within opts.timeout
@post Response.body is open. Caller must close or read to completion.
@thread_safety Safe. Each call is independent.
@related post, put, request, Response

---

@workflow http_request_lifecycle
@purpose Make a request, handle response, clean up resources
@steps
  1. Configure: RequestOpts{} — set timeout, headers
  2. Execute: http.get(url, opts) — returns Response
  3. Consume: resp.json() or resp.text() or resp.bytes()
  4. Cleanup: resp.body.close() — automatic if fully consumed
@antipatterns
  - Don't read body twice. Stream is consumed on first read.
  - Don't skip close(). Leaks connection from pool.
```

---

## Getting started

### Install

```bash
# Clone the repo
git clone https://github.com/dan-strohschein/AID-Docs.git
cd AID-Docs

# Build the Go tools
cd tools/aid-gen-go && go build -o aid-gen-go . && cd ../..
cd tools/aidkit && go build ./cmd/... && cd ../..

# For Python projects, install the Python extractor
cd tools/aid-gen && pip install -e . && cd ../..
```

### Generate AID for your project

```bash
# Go project
./tools/aid-gen-go/aid-gen-go -output .aidocs ./src/...

# Python project
aid-gen path/to/package -o .aidocs

# Generate a manifest (index of all AID files)
./tools/aidkit/aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid

# Validate your AID files
./tools/aidkit/aid-validate .aidocs/*.aid
```

### Generate Layer 2 semantic docs (recommended)

Layer 1 gives you API surface (types, signatures). Layer 2 adds the high-value content: workflows, invariants, antipatterns, and architectural decisions — source-linked so they're verifiable.

```bash
# Build a generator prompt
./tools/aidkit/aid-gen-l2 generate \
  --l1 .aidocs/planner.aid \
  --source ./src/internal/query/planner/ \
  > /tmp/gen-prompt.txt

# Send the prompt to your AI assistant, save the output as L2 draft
# Then build a reviewer prompt to verify accuracy
./tools/aidkit/aid-gen-l2 review \
  --draft .aidocs/planner-l2.aid \
  --project-root . \
  > /tmp/review-prompt.txt

# After code changes, check for stale docs
./tools/aidkit/aid-gen-l2 stale --aid .aidocs/planner-l2.aid --project-root .

# If stale, build an incremental update prompt (only re-verifies changed claims)
./tools/aidkit/aid-gen-l2 update --aid .aidocs/planner-l2.aid --project-root .
```

---

## How AID fits into the AI coding cycle

A typical AI-assisted development cycle looks like this:

```
1. Plan      → Human describes task, AI proposes approach
2. Refine    → Human and AI iterate on the plan
3. Code      → AI writes the implementation
4. Test      → AI writes and runs tests
5. Fix       → AI adjusts code until tests pass
6. Document  → AI updates documentation
7. Submit    → AI commits and creates PR
```

AID changes steps 1, 3, and 6. Here's how:

### Step 1: Planning — AID makes the AI start smarter

**Without AID:** The AI spends the first 30-40% of planning time reading source files to understand your codebase. It opens file after file, building a mental model from scratch. It misses invariants buried in code, doesn't know which infrastructure already exists, and proposes plans that reinvent things you already have.

**With AID:** The AI reads the manifest (`@key_risks` tells it what matters in 1 line per package), loads the relevant AID files via `@depends`, and understands the architecture before reading a single source file. Plans reference existing components instead of proposing new ones.

```
┌─────────────────────────────────────────────────────────┐
│  PLANNING WITH AID                                      │
│                                                         │
│  1. Read .aidocs/manifest.aid                           │
│     → Identify relevant packages from @purpose fields   │
│     → Note @key_risks for each                          │
│                                                         │
│  2. For cross-package tasks:                            │
│     Load full L2 AID for target package + @depends      │
│                                                         │
│  3. For single-file fixes:                              │
│     Load --summary only (annotations + decisions)       │
│                                                         │
│  4. Now plan — with knowledge of:                       │
│     - What infrastructure exists (@methods, @related)   │
│     - What constraints apply (@invariants)              │
│     - What NOT to do (@antipatterns)                    │
│     - WHY things are the way they are (@decision)       │
└─────────────────────────────────────────────────────────┘
```

### Step 3: Coding — AID prevents the biggest waste

**Without AID:** The AI writes correct code that solves the problem — but builds parallel systems because it doesn't know what already exists. Our benchmarks showed an AI writing ~100 lines of custom filtering logic when a 2-line call to an existing FilterNode would have done the same thing, handling all edge cases the custom code missed.

**With AID:** The AI knows what components exist (`@methods`, `@related`), what their contracts are (`@sig`, `@pre`, `@post`), and how they fit together (`@workflow`). It writes 20 lines that reuse existing infrastructure instead of 100 lines that reinvent it.

**Measured results (BM3 — 219K LoC Go database):**

| Metric | Without AID | With AID |
|--------|-------------|----------|
| New code written | ~100 lines | ~20 lines |
| Existing components reused | 0 | 6 |
| Edge cases handled | Basic comparisons only | All predicates including subqueries |
| Architectural consistency | Built parallel system | Used existing patterns |

### Step 6: Documentation — AID updates itself

**Without AID:** The AI updates README or docstrings, which may or may not help the next AI that reads them.

**With AID:** After code changes, AID documents are kept current:

```
┌─────────────────────────────────────────────────────────┐
│  AFTER CODE CHANGES                                     │
│                                                         │
│  1. Re-run L1 extractor on changed packages             │
│     aid-gen-go -output .aidocs ./changed/package/...    │
│                                                         │
│  2. Check L2 staleness                                  │
│     aid-gen-l2 stale --aid .aidocs/pkg-l2.aid           │
│                                                         │
│  3. If stale claims found, re-verify only those         │
│     aid-gen-l2 update --aid .aidocs/pkg-l2.aid          │
│     → Produces focused prompt for stale claims only     │
│     → Send to AI reviewer, apply corrections            │
│                                                         │
│  4. Validate                                            │
│     aid-validate .aidocs/*.aid                           │
│                                                         │
│  5. Commit .aidocs/ alongside code changes              │
└─────────────────────────────────────────────────────────┘
```

### The full cycle with AID

```
 Without AID                          With AID
 ──────────                           ────────
 1. Plan                              1. Plan (read manifest + AID first)
    └─ 40% tokens on exploration         └─ AI starts with architectural knowledge
 2. Refine                            2. Refine (AI references @decisions)
 3. Code                              3. Code (AI reuses existing components)
    └─ Builds from scratch               └─ 5x less code, handles edge cases
 4. Test                              4. Test
 5. Fix                               5. Fix
 6. Update docs (maybe)               6. Update AID (L1 re-extract + L2 staleness check)
 7. Submit                            7. Submit (AID committed with code)
```

Steps 2, 4, 5, and 7 are unchanged. AID doesn't add steps — it makes steps 1, 3, and 6 dramatically more effective, and makes step 6 systematic instead of optional.

---

## Configuring your AI assistant

The key is making your assistant aware that `.aidocs/` exists and should be read before working on your code.

### Claude Code

Add this to your project's `CLAUDE.md`:

```markdown
## AID Documentation

This project uses AID (Agent Interface Document) files in `.aidocs/`.
Before modifying any package, read its AID file:

- Start with `.aidocs/manifest.aid` to find relevant packages
- For cross-package tasks: read full L2 AID + @depends chain
- For single-file fixes: use `aid-parse --summary` for just invariants and decisions
- Check @invariants and @antipatterns BEFORE making changes
- Check @decision blocks to understand WHY code is structured a certain way
```

### Cursor / Copilot / Other assistants

Add to `.cursorrules`, `.github/copilot-instructions.md`, or equivalent:

```
This project has AID documentation in .aidocs/. Before modifying code:
1. Read .aidocs/manifest.aid for package index and @key_risks
2. Read the relevant .aid file for the package you're changing
3. Check @invariants, @antipatterns, and @workflow blocks
4. AID files have [src: file:line] references — verify claims if unsure
```

### Any LLM via API

```python
# Load relevant AID based on task
manifest = open(".aidocs/manifest.aid").read()
# Parse manifest, match task to @purpose fields, load @depends chain
relevant_aids = select_relevant_packages(manifest, task_description)
context = "\n---\n".join(open(f".aidocs/{f}").read() for f in relevant_aids)
prompt = f"## Project Documentation\n{context}\n## Task\n{task_description}"
```

---

## Benchmark results

Tested against real codebases (10K-219K LoC) that the AI had never seen:

| Benchmark | Task type | Tokens | Quality |
|-----------|-----------|--------|---------|
| BM1 (Aria compiler, 10K LoC) | Simple, pattern-following | **-47%** | Same |
| BM3 (SyndrDB, 219K LoC) | Cross-package feature | **-10%** | **5x less code, 6 components reused** |
| BM4 (SyndrDB flusher) | Single-file bug fix, slim loading | **-12%** | Better API compat, invariant awareness |

Full benchmark reports in `tools/aid-bench/results/`.

---

## Tools

| Command | Purpose |
|---------|---------|
| `aid-gen-go` | Generate Layer 1 AID from Go source code |
| `aid-gen` | Generate Layer 1 AID from Python source code |
| `aid-parse [--summary]` | Parse .aid files to JSON, or extract annotations only |
| `aid-validate` | Check .aid files against 8 spec validation rules |
| `aid-discover` | Find nearest .aidocs/ directory |
| `aid-manifest-gen` | Auto-generate manifest.aid from .aidocs/ contents |
| `aid-gen-l2 generate` | Build Layer 2 generator prompt |
| `aid-gen-l2 review` | Build Layer 2 reviewer prompt |
| `aid-gen-l2 stale` | Check for stale [src:] references |
| `aid-gen-l2 update` | Build incremental update prompt for stale claims |

## Repository structure

```
AID/
├── README.md                  # This file
├── CLAUDE.md                  # AI assistant guide for this project
├── spec/
│   ├── format.md              # The AID format specification (13 sections)
│   ├── fields.md              # Complete field reference (92 fields)
│   ├── design-rationale.md    # Why every design decision was made (31 entries)
│   ├── generation.md          # How AID files are generated from source code
│   └── layer2.md              # Layer 2: AI-generated semantic docs with verification
├── examples/
│   ├── http-client.aid        # HTTP client module example
│   ├── collections-hashmap.aid # Generic collections example
│   └── events-emitter.aid     # Async/callback patterns example
└── tools/
    ├── aid-gen/               # Python L1 extractor (99 tests)
    ├── aid-gen-go/            # Go L1 extractor
    ├── aidkit/                # Go toolkit: parser (19 tests), validator (20 tests),
    │                          #   emitter (6 tests), discovery, L2 pipeline
    └── aid-bench/             # Benchmark harness + results (BM1-BM4)
```

## Design principles

1. **Every field is optional except `@purpose` and `@sig`/`@fields`.** Partial docs beat no docs.
2. **No prose paragraphs.** Every piece of information has a named field.
3. **Constraints are machine-checkable.** `Range [0, 20]`, `Must match ^regex$`, not "should be reasonable."
4. **One file per module.** Not per-function (too fragmented) or per-library (too large).
5. **Flat, not nested.** Minimal indentation. Linear field sequences. Low token cost.
6. **Cross-references by name.** `@related post, Response` not `@related ./post.aid#L45`.
7. **Source-linked claims.** Every L2 semantic assertion has a `[src: file:line]` reference.
8. **Living documents.** `@code_version` tracks which commit the AID describes. Staleness is detectable.

## Format overview

AID files (`.aid`) are structured plain text. One file per module. Four tiers:

| Tier | Purpose | Example |
|------|---------|---------|
| **Module Header** | Identity, version, stability | `@module http/client` |
| **Entries** | Individual API contracts | `@fn get` with signature, errors, constraints |
| **Module Annotations** | Cross-cutting concerns | `@invariants`, `@antipatterns`, `@decision` blocks |
| **Workflows** | Multi-step usage patterns | Step-by-step data flows with error mapping |

## Target languages

AID is language-agnostic. Current tooling supports:

- **Go** — full L1 extractor + all toolkit tools
- **Python** — L1 extractor (99 tests)
- **TypeScript** — planned

## Spiritual connection

AID shares its design philosophy with the [Aria programming language](https://github.com/dan-strohschein/aria): every token carries meaning, no implicit behavior, designed for AI from the ground up. Aria is the language built for agents. AID is the documentation format built for agents.

## License

TBD
