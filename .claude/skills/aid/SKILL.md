---
name: aid
description: Use when generating, updating, or working with AID skeleton files (.aid). TRIGGER when user says "generate skeletons", "create .aidocs", "update AID files", "AID generation", "aid-gen", or when a project needs .aidocs/ created for chisel/cartograph.
---

# AID — Agent Interface Document Generation

AID is a documentation format for AI coding agents. Layer 1 generators extract structure from source code; Layer 2 adds semantic annotations via AI.

## Generators by Language

| Language | Tool | Location (relative to AID repo root) |
|----------|------|--------------------------------------|
| **Go** | `aid-gen-go` | `tools/aid-gen-go/` |
| **Python** | `aid-gen-py` | `tools/aid-gen/` |
| **TypeScript** | `aid-gen-ts` | `tools/aid-gen-ts/` |
| **C#** | `aid-gen-cs` | `tools/aid-gen-cs/` |

### Build Go generator
```bash
cd tools/aid-gen-go && go build -o aid-gen-go .
```

### Install Python generator
```bash
cd tools/aid-gen && pip install -e .
```

## Toolkit (aidkit)

Two implementations — use whichever matches your environment:

| Tool | Language | Location |
|------|----------|----------|
| **aidkit** (Go) | Go | `tools/aidkit/` (separate repo: github.com/dan-strohschein/aidkit) |
| **aidkit-py** | Python | `tools/aidkit-py/` |

Both provide: parser, validator, emitter, discovery, L2 pipeline (generator, reviewer, staleness, diff).

### Install Python aidkit
```bash
cd tools/aidkit-py && pip install -e .
```

### Build Go aidkit CLIs
```bash
cd tools/aidkit
go build -o aid-gen-l2 ./cmd/aid-gen-l2/
go build -o aid-parse ./cmd/aid-parse/
go build -o aid-validate ./cmd/aid-validate/
go build -o aid-manifest-gen ./cmd/aid-manifest-gen/
```

## Generating L1 Skeletons

### Go project
```bash
cd /path/to/project
aid-gen-go --output .aidocs --internal -v ./src/...
```

### Python project
```bash
cd /path/to/project
aid-gen-py --output .aidocs -v /path/to/source/
```

### Generate manifest (after L1)
```bash
# Go version
aid-manifest-gen --dir .aidocs/

# Python version
aid-manifest-gen-py --dir .aidocs/
```

## L2 Generation (AI-Assisted)

Build the L2 prompt, then feed it to an AI agent:

### Full generation
```bash
# Go version
aid-gen-l2 generate --l1 .aidocs/package.aid --source ./src/package/

# Python version
aid-gen-l2-py generate --l1 .aidocs/package.aid --source ./src/package/
```

### Incremental generation (after code changes — 91% fewer tokens)
```bash
aid-gen-l2 generate --l1 new.aid --old-l1 old.aid --existing-l2 package-l2.aid --source ./src/package/
```

### Review L2 output
```bash
aid-gen-l2 review --draft .aidocs/package.aid --project-root .
```

### Check staleness
```bash
aid-gen-l2 stale --aid .aidocs/package.aid --project-root .
```

## L1 Generator Flags

| Flag | Go | Python | Description |
|------|-----|--------|-------------|
| `--output` | Yes | Yes | Output directory (default: `.aidocs`) |
| `--stdout` | Yes | Yes | Print to stdout |
| `--module` | Yes | Yes | Override module name |
| `--version` | Yes | `--version-tag` | Library version |
| `--internal` | Yes | No | Include unexported functions |
| `--test` | Yes | No | Generate AID for test packages |
| `--exclude` | No | Yes | Glob patterns to skip |
| `-v` | Yes | Yes | Verbose output |

## Layer 1 Fields Extracted

Both Go and Python L1 generators extract:
- `@fn`, `@sig`, `@params`, `@returns` — function signatures
- `@type`, `@kind`, `@fields`, `@variants` — type definitions
- `@trait`, `@requires`, `@provided` — interfaces/protocols
- `@const`, `@value_type`, `@value` — constants
- `@calls` — function call graph (from AST analysis)
- `@source_file`, `@source_line` — exact source location
- `@extends`, `@implements` — inheritance and protocol detection

## Three-Layer Architecture

1. **Layer 1 (Mechanical)** — Generators extract structure deterministically from AST. Fast, no AI.
2. **Layer 2 (AI-Assisted)** — Agent reads skeleton + source to fill `@purpose`, `@errors`, `@pre`/`@post`, `@effects`, `@workflow`, `@invariants`, `@antipatterns`. Source-linked via `[src: file:line]`.
3. **Layer 3 (Human Review)** — Author verifies and adds domain knowledge.

## Common Workflow: Set Up a New Project

```bash
# 1. Generate L1 skeletons
cd /path/to/project
aid-gen-go --output .aidocs --internal -v ./src/...   # Go
# OR
aid-gen-py --output .aidocs -v ./src/                  # Python

# 2. Generate manifest
aid-manifest-gen --dir .aidocs/

# 3. Verify with cartograph
cartograph stats --dir .aidocs/

# 4. Generate L2 for a key package
aid-gen-l2 generate --l1 .aidocs/planner.aid --source ./src/internal/query/planner/ > /tmp/l2-prompt.txt
# Feed prompt to AI agent

# 5. Validate output
aid-validate .aidocs/planner.aid
```

## AID v0.2 File Format

```
@module package/name
@lang go
@version 0.1.0
@purpose Brief description
@depends [dep1, dep2]
@aid_version 0.2

---

@fn FunctionName
@purpose What it does
@sig (param: type) -> returnType ! errorType
@params
  param: type — description
@calls [OtherFunction, Type.Method]
@source_file path/to/file.go
@source_line 42
@thread_safety safe. No shared mutable state.

---

@type TypeName
@kind struct
@fields_visibility partial
@fields
  name: type — description
```

## Project-Level Documentation (v0.2)

For project architecture, create `.aidocs/project.aid`:
```
@project MyProject
@lang go
@aid_version 0.2

@layers
  server — HTTP handlers and routing
  service — Business logic
  repository — Data access

@boundaries
  repository -> server: FORBIDDEN

---

@convention error_wrapping
@purpose Error wrapping at layer boundaries
@rule Wrap with fmt.Errorf("[layer]: %w", err)
@rule Use errors.Is() for matching
```
