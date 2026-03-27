# Using AID on a Greenfield Project

**How to use AID from day one on a new codebase.**

---

## The question

On an existing codebase, AID extracts knowledge that already exists in the code. But on a greenfield project, there IS no code yet. So when does AID enter the picture, and what does the development cycle look like?

The answer: **AID grows with the code.** It's not a one-time generation step — it's part of the build cycle, the same way tests are.

---

## Day 1: Project setup

Before writing any code, create the AID infrastructure:

```bash
mkdir -p .aidocs
touch .aidocs/manifest.aid
```

Add to your `CLAUDE.md` (or `.cursorrules`, etc.):

```markdown
## AID Documentation

This project uses AID files in .aidocs/ for AI-optimized documentation.

- Before modifying any package, check .aidocs/manifest.aid for relevant docs
- After creating a new package, generate its L1 AID
- After significant features, request L2 generation for the package
- AID files are committed alongside code — treat them as project artifacts
```

That's it. No AID files exist yet because no code exists yet.

---

## The greenfield development cycle

Here's how a typical feature cycle works with AID, from empty repo to mature codebase:

### Sprint 1: First features (no AID yet — and that's fine)

```
1. Plan    → AI and human design the architecture
2. Code    → AI writes the first packages
3. Test    → AI writes and runs tests
4. Commit  → Code ships

No AID yet. The codebase is small enough to fit in context.
The AI reads everything directly — no docs needed.
```

AID doesn't help here. With 5 files and 500 lines, the AI can read the whole codebase. Don't generate AID for a project this small — it's overhead.

**When to start AID:** When the codebase grows past what the AI can comfortably hold in context. Rule of thumb: **when you have 10+ files across 3+ packages, generate L1 AID.** That's usually sprint 2-3.

### Sprint 2-3: Codebase grows — generate L1

The project now has real structure: multiple packages, cross-package dependencies, non-obvious patterns emerging. This is when the AI starts wasting tokens exploring.

```
1. Plan    → AI reads source (getting slower as codebase grows)
2. Code    → AI writes new package
3. Test    → Tests pass
4. GENERATE L1 AID for new and changed packages
   └─ aid-gen-go -output .aidocs ./src/...
5. GENERATE MANIFEST
   └─ aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
6. Commit  → Code + .aidocs/ committed together
```

The new step (4-5) takes seconds. Now the next task benefits — the AI reads the manifest and L1 AID instead of opening every file.

### Sprint 4+: Architecture stabilizes — generate L2

The project now has established patterns: how data flows, what invariants matter, where the sharp edges are. This is when L2 becomes valuable.

```
1. Plan    → AI reads manifest + relevant AID files
   └─ Cross-package task? Load full L2 + @depends chain
   └─ Single-file fix? Load --summary only
2. Code    → AI reuses existing components (knows they exist from AID)
3. Test    → Tests pass
4. UPDATE L1 for changed packages
   └─ aid-gen-go -output .aidocs ./changed/package/...
5. CHECK L2 STALENESS
   └─ aid-gen-l2 stale --aid .aidocs/pkg-l2.aid --project-root .
6. IF STALE: regenerate only stale claims
   └─ aid-gen-l2 update --aid .aidocs/pkg-l2.aid --project-root .
   └─ Send prompt to AI, apply corrections
7. VALIDATE
   └─ aid-validate .aidocs/*.aid
8. Commit  → Code + updated .aidocs/ committed together
```

### Mature project: the full cycle

Once L2 exists for core packages, this is the steady-state cycle:

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  1. PLAN                                                │
│     AI reads manifest → loads relevant AID              │
│     Knows: what exists, what the invariants are,        │
│     what NOT to do, WHY things are structured this way  │
│                                                         │
│  2. REFINE                                              │
│     AI references @decision blocks:                     │
│     "This was designed as X because Y — should we       │
│     change that decision or work within it?"            │
│                                                         │
│  3. CODE                                                │
│     AI writes implementation reusing existing           │
│     components from @methods, @related, @workflow       │
│     Respects @invariants and @antipatterns              │
│                                                         │
│  4. TEST                                                │
│     AI writes tests. AID's @errors and @pre/@post       │
│     inform what edge cases to test.                     │
│                                                         │
│  5. FIX                                                 │
│     AI adjusts until tests pass                         │
│                                                         │
│  6. UPDATE AID                                          │
│     a. Re-run L1 extractor on changed packages          │
│     b. Check L2 staleness                               │
│     c. If stale: re-verify stale claims only            │
│     d. If new package: generate L1, optionally L2       │
│     e. Update manifest                                  │
│     f. Validate all AID files                           │
│                                                         │
│  7. COMMIT                                              │
│     Code + tests + .aidocs/ committed together          │
│     PR includes AID diff alongside code diff            │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## When to generate L2 for a package

Not every package needs L2. L1 (type signatures and API surface) is sufficient for simple utility packages. L2 (workflows, invariants, antipatterns, decisions) is worth the investment when:

| Signal | Why L2 helps |
|--------|-------------|
| Package has 3+ consumers | Multiple callers need to understand the contract |
| Cross-package data flow | Workflows document how data moves between packages |
| Non-obvious invariants | "BRIN is lossy" isn't visible from the type signature |
| Historical design decisions | @decision blocks prevent undoing intentional tradeoffs |
| Complex state machines | State transitions, lifecycle management, ordering constraints |
| Error handling patterns | Which errors propagate, which are caught, which are fatal |

For a greenfield project, the first L2 candidates are usually:
1. The **core domain model** — the types everything else depends on
2. The **main data pipeline** — how requests flow from input to output
3. The **storage/persistence layer** — lifecycle, transactions, cleanup

---

## Automating the cycle

### Git hook (pre-commit)

```bash
#!/bin/bash
# .git/hooks/pre-commit — auto-generate L1 AID for changed Go packages

changed_dirs=$(git diff --cached --name-only --diff-filter=ACM -- '*.go' |
  xargs -I{} dirname {} | sort -u)

for dir in $changed_dirs; do
  aid-gen-go -output .aidocs "$dir"
done

aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
aid-validate .aidocs/*.aid

git add .aidocs/
```

### CI check

```yaml
# .github/workflows/aid.yml
- name: Validate AID docs
  run: |
    aid-validate .aidocs/*.aid
    aid-gen-l2 stale --aid .aidocs/*.aid --project-root . || echo "Stale AID detected"
```

### Claude Code hook

```json
{
  "hooks": {
    "post-commit": "aid-gen-go -output .aidocs ./src/... && aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid"
  }
}
```

---

## What NOT to do

1. **Don't generate AID before you have code.** AID documents what exists. On day 1, nothing exists. Start with code, add AID when the codebase outgrows what fits in context.

2. **Don't generate L2 for every package.** L2 costs tokens to generate and tokens to load. Only generate it for packages where the architectural knowledge is genuinely non-obvious. A `utils/strings` package doesn't need L2.

3. **Don't treat AID as a substitute for reading code.** AID is a map, not the territory. It tells the agent where to look and what to expect. The agent still reads source code — it just reads the right files first.

4. **Don't hand-write AID from scratch.** Always start from L1 extraction. Hand-editing L2 is fine (and valuable for critical invariants), but the structure should come from the extractor.

5. **Don't let AID go stale.** Stale AID is worse than no AID — it gives the agent false confidence. Use `aid-gen-l2 stale` in CI. If claims are stale, either update them or delete them.

---

## The minimal viable AID setup

If you want to start with the least effort:

```bash
# One-time setup (30 seconds)
mkdir .aidocs
echo "Read .aidocs/ before modifying code" >> CLAUDE.md

# After each feature (10 seconds)
aid-gen-go -output .aidocs ./src/...
aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
git add .aidocs/
```

That's it. L1 only, no L2, no validation, no staleness checks. Just mechanical extraction committed alongside code. Even this minimal setup gives the AI:
- A manifest showing all packages and their purposes
- Type signatures without reading every file
- Method lists showing what infrastructure exists

Add L2, validation, and staleness checks as the project grows and the payoff justifies the investment.
