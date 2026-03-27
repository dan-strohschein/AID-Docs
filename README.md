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

## Format overview

AID files (`.aid`) are structured plain text. One file per module. Three tiers:

| Tier | Purpose | Example |
|------|---------|---------|
| **Module Header** | Identity, version, stability | `@module http/client` |
| **Entries** | Individual API contracts (functions, types, traits, constants) | `@fn get` with full signature, errors, constraints |
| **Workflows** | Multi-step usage patterns | "Make request, consume body, close connection" |

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

## Using AID with your AI code assistant

AID files live in a `.aidocs/` directory at your project root. Any AI coding agent can use them — the format is plain text, language-agnostic, and designed for exactly this purpose.

### Step 1: Generate AID files for your project

```bash
# For Go projects
aid-gen-go -output .aidocs ./src/...

# For Python projects
aid-gen path/to/package -o .aidocs
```

This creates Layer 1 (mechanical extraction) AID files — type signatures, function contracts, struct fields, interface definitions.

### Step 2: Generate Layer 2 semantic docs (optional but recommended)

Layer 2 adds the high-value content: workflows, invariants, antipatterns, and source-linked constraints. Use the `aid-gen-l2` tool to build prompts for your AI assistant:

```bash
# Generate a prompt for the L2 generator
aid-gen-l2 generate --l1 .aidocs/planner.aid --source ./src/internal/query/planner/ > /tmp/gen-prompt.txt

# Send the prompt to your AI assistant, save the output as a draft
# Then generate a reviewer prompt
aid-gen-l2 review --draft .aidocs/planner-l2.aid --project-root . > /tmp/review-prompt.txt

# Check for stale docs after code changes
aid-gen-l2 stale --aid .aidocs/planner-l2.aid --project-root .
```

### Step 3: Tell your AI assistant to use the AID files

The key is making your assistant aware that `.aidocs/` exists and should be read before working on your code. Here's how to do it with different tools:

#### Claude Code

Add this to your project's `CLAUDE.md` file:

```markdown
## AID Documentation

This project uses AID (Agent Interface Document) files for AI-optimized documentation.
Before modifying any package, read its AID file from the `.aidocs/` directory:

- `.aidocs/` contains Layer 1 (API surface) and Layer 2 (semantic) docs
- Layer 2 files have `[src: file:line]` references linking claims to code
- Read the relevant AID files BEFORE reading source code — they give you
  the architectural context that takes hours to figure out from code alone
- When AID docs describe workflows, invariants, or antipatterns — follow them

To find the right AID file for a package:
1. Check the `@depends` field to find related AID files
2. Read the `@workflow` blocks to understand data flows
3. Check `@invariants` and `@antipatterns` before making changes
```

#### Cursor / Copilot / Other assistants

Add a similar instruction to whatever project-level config your assistant reads (`.cursorrules`, `.github/copilot-instructions.md`, etc.):

```
This project has AID documentation in .aidocs/. Before modifying code in any package,
read the corresponding .aid file. AID files contain:
- @workflow blocks showing how components interact
- @invariants with [src: file:line] references to enforcing code
- @antipatterns documenting known pitfalls
- @sig with full type signatures and error types

Read AID files before source code — they're designed to give you the context you need
in fewer tokens than reading the source directly.
```

#### Any LLM via API

When building agentic workflows, include relevant AID files in the system prompt or context window. Use the `@depends` field for selective loading — only include AID files for packages the task touches:

```python
# Load only relevant AID files based on the task
relevant_packages = ["planner", "brinindex", "models"]
context = ""
for pkg in relevant_packages:
    aid_path = f".aidocs/{pkg}.aid"
    if os.path.exists(aid_path):
        context += open(aid_path).read() + "\n---\n"

# Include in the prompt
prompt = f"## Project Documentation\n{context}\n## Task\n{task_description}"
```

### What to expect

Based on our benchmarks against a 219K LoC Go database server:

| Metric | Without AID | With AID (L1+L2) |
|--------|-------------|------------------|
| Token usage | Baseline | **10% fewer** |
| Tool calls / file reads | Baseline | **39% fewer** |
| Time to complete | Baseline | **14% faster** |
| Code reuse (existing infra) | 0 components | **6 components reused** |
| New code written | ~100 lines | **~20 lines** |
| Solution quality | Correct but reinvents | **Correct and idiomatic** |

The biggest win isn't speed — it's **codebase fluency**. Without AID, AI agents find bugs but solve them by building parallel systems because they don't know what infrastructure already exists. With AID, they write the solution a senior developer who knows the codebase would write.

## Status

AID is in active development. The format spec is stable (v0.1). Tooling is functional:

| Tool | Language | Status |
|------|----------|--------|
| `aid-gen` | Python | Layer 1 extractor (99 tests) |
| `aid-gen-go` | Go | Layer 1 extractor for Go |
| `aidkit` | Go | Parser, validator, L2 pipeline |

## Repository structure

```
AID/
├── README.md                  # This file
├── CLAUDE.md                  # AI assistant guide for this project
├── spec/
│   ├── format.md              # The AID format specification
│   ├── fields.md              # Complete field reference
│   ├── design-rationale.md    # Why every design decision was made
│   ├── generation.md          # How AID files are generated from source code
│   └── layer2.md              # Layer 2: AI-generated semantic docs with verification
├── examples/
│   ├── http-client.aid        # HTTP client module example
│   ├── collections-hashmap.aid # Generic collections example
│   └── events-emitter.aid     # Async/callback patterns example
└── tools/
    ├── aid-gen/               # Python L1 extractor
    ├── aid-gen-go/            # Go L1 extractor
    ├── aidkit/                # Go toolkit (parser, validator, L2 pipeline)
    └── aid-bench/             # Benchmark harness + results
```

## Design principles

1. **Every field is optional except `@purpose` and `@sig`/`@fields`.** Partial docs beat no docs.
2. **No prose paragraphs.** Every piece of information has a named field.
3. **Constraints are machine-checkable.** `Range [0, 20]`, `Must match ^regex$`, not "should be reasonable."
4. **One file per module.** Not per-function (too fragmented) or per-library (too large).
5. **Flat, not nested.** Minimal indentation. Linear field sequences. Low token cost.
6. **Cross-references by name.** `@related post, Response` not `@related ./post.aid#L45`.

## Target languages

AID is language-agnostic. Priority targets for initial tooling:

- **Python** — richest existing type annotation ecosystem, most immediate value
- **Go** — strong typing, good for generator tooling
- **TypeScript** — large ecosystem, `.d.ts` files provide type information

## Spiritual connection

AID shares its design philosophy with the [Aria programming language](https://github.com/dan-strohschein/aria): every token carries meaning, no implicit behavior, designed for AI from the ground up. Aria is the language built for agents. AID is the documentation format built for agents.

## License

TBD
