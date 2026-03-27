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

## Status

AID is in early design. The format spec is being developed in this repository.

## Repository structure

```
AID/
├── README.md                  # This file
├── CLAUDE.md                  # AI assistant guide for this project
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
