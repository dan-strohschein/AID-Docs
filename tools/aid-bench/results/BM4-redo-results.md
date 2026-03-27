# Benchmark 4 Redo: Slim AID Loading Strategy

**Date:** 2026-03-27
**Task:** Fix AdaptiveFlusher pending request accumulation bug (same as BM4)
**Change:** Summary-only AID loading instead of full L2

---

## The change: `--summary` mode + `@key_risks`

Instead of loading the full 27K char L2 AID file, Agent B got:
- `@key_risks` inline in prompt: 130 chars (one line from the manifest)
- Summary AID via `aid-parse --summary`: 306 lines / 17.5K chars (annotations + decisions only, no per-entry detail)

## Results

| Metric | No Docs | Full L2 AID (BM4) | Slim AID (BM4 redo) |
|--------|---------|-------------------|---------------------|
| Tokens | 26,038 | 43,653 (+68%) | **22,912 (-12%)** |
| Tool calls | 6 | 8 | **3** |
| Duration | 62s | 76s | **66s** |
| Fix quality | Good | Good+ | **Good+** |

## What changed

The slim agent:
- Read the summary AID (1 tool call) instead of full L2 + L1 bundlestore + manifest (3 tool calls)
- Got the same key insights: lock ordering invariant, shutdown antipattern, pending request lifecycle
- Skipped 500+ lines of per-entry type descriptions that were irrelevant to the bug fix
- Produced the same quality solution as the full AID agent

## Updated cumulative results

| Benchmark | Task type | AID tokens | AID quality |
|-----------|-----------|-----------|-------------|
| BM1 | Simple, small codebase | **-47%** | Same |
| BM3 | Cross-package feature | **-10%** | **Much better** (5x less code) |
| BM4 original | Single-file, full L2 | +68% | Slightly better |
| **BM4 redo** | **Single-file, slim AID** | **-12%** | **Same as full AID** |

## Conclusion

The slim loading strategy (`--summary` + `@key_risks`) eliminates the token overhead penalty for simple tasks while preserving all quality advantages. AID now wins or ties on BOTH efficiency and quality across ALL tested task types.

The key insight: **match context depth to task scope.**
- Cross-package → full L2 with `@depends` chain
- Single-file → summary only (annotations + decisions)
- Quick orientation → `@key_risks` from manifest (one line)
