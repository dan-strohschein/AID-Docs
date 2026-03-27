# AID — Agent Interface Document (CLAUDE.md)

## Project Overview

AID is a documentation format designed for AI coding agents. This repository contains the format specification — the spec is the product.

### Repository Structure

```
AID/
├── CLAUDE.md                  # This file — AI assistant guide
├── README.md                  # Project overview
├── spec/
│   ├── format.md              # The AID format specification
│   ├── fields.md              # Complete field reference
│   ├── design-rationale.md    # Why every design decision was made
│   └── generation.md          # How AID files are generated from source code
└── examples/
    ├── http-client.aid        # HTTP client module example
    ├── collections-hashmap.aid # Generic collections example
    └── events-emitter.aid     # Async/callback patterns example
```

### Purpose

Every file in this repo is a specification document or example AID file. The goal is to produce a complete, consistent, implementable format specification that tooling authors can use to build AID generators, validators, and consumers.

---

## Design Principles (Non-Negotiable)

### 1. Token Efficiency

Every character in an AID file should carry information. No prose, no filler, no formatting for aesthetics. Measure quality in information-per-token.

### 2. Completeness Over Brevity

When token efficiency and completeness conflict, completeness wins. A missing error case costs more (in debugging tokens) than the tokens to document it.

### 3. Language Agnostic

AID uses universal type notation that maps to any programming language. The `@lang` field tells tooling how to translate. Never assume a specific language.

### 4. Explicit Everything

If it's not written in the AID file, an agent must assume it's unknown. No implicit defaults, no "obvious" behavior, no "see the source code."

### 5. Parsability

The format must be trivially parseable with line-by-line processing. No context-dependent parsing. No complex grammar. An intern should be able to write a parser in an afternoon.

---

## Spec File Conventions

### Writing Rules

- Every design decision needs a rationale in `design-rationale.md`
- Syntax descriptions must be accompanied by examples
- Field tables must list: field name, required/optional, type, description
- Use `@field` notation in examples, not abstract descriptions
- Cross-references use relative paths within the repo
- Tone: direct, authoritative, no hedging

### What NOT to Propose

- JSON/YAML/XML as the base format (token-inefficient)
- Prose fields or paragraph-length descriptions within AID files
- Language-specific type notation in the spec (use universal types)
- Required fields beyond `@purpose` + `@sig`/`@fields` (partial docs must be valid)
- Deeply nested structures (flat is better)
- Features that require a complex parser

---

## Target Languages for Tooling

Priority order for generator/validator implementation:

1. **Python** — richest type annotation ecosystem, most immediate value
2. **Go** — strong typing, good stdlib documentation
3. **TypeScript** — `.d.ts` files provide type information

---

## Spiritual Connection to Aria

AID shares design philosophy with the Aria programming language: every token carries meaning, no implicit behavior, designed for AI from the ground up. Aria is the language built for agents. AID is the documentation format built for agents. They are separate projects with shared values.

Future goal: use AID to document Aria's standard library as the reference implementation of the format.

---

## Build & Verify

There is no compiler or tooling yet. Verification means:

1. **Consistency** — all examples conform to the rules in `spec/format.md`
2. **Completeness** — `spec/fields.md` covers every field used in examples
3. **Rationale coverage** — every major design choice is explained in `spec/design-rationale.md`
4. **Parsability test** — examples can be parsed by a simple line-by-line script
5. **Token efficiency test** — compare AID examples against equivalent human docs for token count
