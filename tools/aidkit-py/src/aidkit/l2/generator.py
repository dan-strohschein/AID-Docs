"""L2 generator prompt builder with L1-guided file selection."""
from __future__ import annotations

import os
from pathlib import Path

from ..model import AidFile, Entry, Field


def build_generator_prompt(
    l1_aid: AidFile,
    source_dir: str,
    dep_aids: list[AidFile] | None = None,
) -> str:
    """Construct the prompt for a Layer 2 generator agent.

    Source file selection is guided by L1 metadata: only files containing
    documented functions/types (via @source_file) and their callees (via @calls)
    are listed. This typically reduces the file set by 40-60%.
    """
    dep_aids = dep_aids or []
    parts: list[str] = []

    parts.append("You are a Layer 2 AID Generator. Your job is to produce semantic documentation.\n\n")

    # L1 AID content
    parts.append("## Layer 1 AID (mechanical extraction)\n\n")
    parts.append(_read_aid_as_text(l1_aid))
    parts.append("\n\n")

    # Dependency AIDs
    if dep_aids:
        parts.append("## Related package AIDs (for cross-package context)\n\n")
        for dep in dep_aids:
            parts.append(_read_aid_as_text(dep))
            parts.append("\n\n---\n\n")

    # Source file listing -- guided by L1 metadata
    parts.append("## Source files to read\n\n")
    parts.append(f"Source directory: {source_dir}\n\n")

    files = extract_relevant_files(l1_aid)
    if not files:
        files = _list_source_files(source_dir)
        parts.append("(All source files listed -- L1 has no @source_file metadata)\n\n")
    else:
        parts.append("(Selected by L1 analysis -- contains all documented functions and their callees)\n\n")

    for f in files:
        parts.append(f"- {f}\n")
    parts.append("\n")

    # Instructions -- conditionally assembled
    parts.append(CORE_INSTRUCTIONS)

    if _has_error_fields(l1_aid):
        parts.append(ERROR_MAP_INSTRUCTIONS)
    if _detect_concurrency_primitives(source_dir, files):
        parts.append(LOCK_INSTRUCTIONS)

    parts.append(OUTPUT_FORMAT_INSTRUCTIONS)

    return "".join(parts)


def extract_relevant_files(l1_aid: AidFile) -> list[str]:
    """Analyze L1 AID entries to determine which source files the generator needs.

    Uses @source_file for direct references and @calls to include callee files.
    """
    file_set: set[str] = set()
    name_to_file: dict[str, str] = {}

    for e in l1_aid.entries:
        sf = e.fields.get("source_file")
        if sf and sf.inline_value:
            file_set.add(sf.inline_value)
            name_to_file[e.name] = sf.inline_value
            # Also index by short name for method resolution
            dot_idx = e.name.rfind(".")
            if dot_idx >= 0:
                name_to_file[e.name[dot_idx + 1:]] = sf.inline_value

    # Resolve callees to their source files
    for e in l1_aid.entries:
        calls = e.fields.get("calls")
        if calls:
            for callee in _parse_calls_list(calls.inline_value):
                if callee in name_to_file:
                    file_set.add(name_to_file[callee])

    return sorted(file_set)


def _parse_calls_list(calls_value: str) -> list[str]:
    """Split a @calls value like 'Validate, store.Push' into individual names."""
    calls_value = calls_value.strip("[]")
    return [p.strip() for p in calls_value.split(",") if p.strip()]


def _has_error_fields(l1_aid: AidFile) -> bool:
    return any("errors" in e.fields for e in l1_aid.entries)


def _detect_concurrency_primitives(source_dir: str, files: list[str]) -> bool:
    patterns = ["sync.Mutex", "sync.RWMutex", "chan struct{}", "atomic.",
                "threading.Lock", "asyncio.Lock", "multiprocessing.Lock"]
    for f in files:
        full_path = os.path.join(source_dir, f)
        try:
            content = Path(full_path).read_text(encoding="utf-8")
        except OSError:
            continue
        for pat in patterns:
            if pat in content:
                return True
    return False


def _list_source_files(directory: str) -> list[str]:
    source_exts = {".go", ".py", ".ts", ".rs"}
    files: list[str] = []
    base = Path(directory)
    for path in sorted(base.rglob("*")):
        if path.is_file() and path.suffix in source_exts:
            if not path.name.endswith("_test.go"):
                files.append(str(path.relative_to(base)))
    return files


def _read_aid_as_text(f: AidFile) -> str:
    parts: list[str] = []
    parts.append(f"@module {f.header.module}\n")
    if f.header.lang:
        parts.append(f"@lang {f.header.lang}\n")
    if f.header.purpose:
        parts.append(f"@purpose {f.header.purpose}\n")

    for e in f.entries:
        parts.append(f"\n@{e.kind} {e.name}\n")
        for name, field in e.fields.items():
            if name == e.kind:
                continue
            if field.inline_value:
                parts.append(f"@{name} {field.inline_value}\n")
            else:
                parts.append(f"@{name}\n")
            for line in field.lines:
                parts.append(f"  {line}\n")
    return "".join(parts)


# ---------------------------------------------------------------------------
# Instruction templates
# ---------------------------------------------------------------------------

CORE_INSTRUCTIONS = """## Instructions

Read ONLY the source files listed above. For each @fn entry in the L1 AID, its source is at the @source_file and @source_line indicated. Focus on: (1) function bodies, (2) type definitions, (3) error sentinels. Do NOT read CLAUDE.md, README.md, or test files. If you discover a function calls into a file not listed, you may read it.

Produce an enriched AID file that **preserves ALL L1 content** and adds L2 semantic annotations.

### CRITICAL: Preserve L1 Content

Your output MUST include every @fn, @type, @trait, and @const entry from the L1 AID above -- with their @sig, @params, @returns, @calls, @source_file, and @source_line fields intact. Do NOT drop or rewrite L1 entries.

For each L1 entry, you MAY enhance:
- The @purpose field (explain WHY, not just WHAT)
- Add @pre/@post conditions
- Add @errors details
- Add @thread_safety notes

But you MUST keep: @fn name, @sig, @params, @calls, @source_file, @source_line unchanged.

### Add L2 Blocks

After the preserved L1 entries, add:

1. **@workflow blocks** -- major data flows with numbered steps
2. **@invariants with [src:] references** -- constraints that always hold
3. **@antipatterns with [src:] references** -- common mistakes to avoid

For EVERY semantic claim, include a [src: relative/path:LINE] or [src: relative/path:START-END] reference.

"""

ERROR_MAP_INSTRUCTIONS = """### @error_map format

If the module defines error sentinel values (e.g., ErrOutOfOrder, ErrNotFound), add @error_map blocks:

```
@error_map <name>
@purpose <what this error group covers>
@entries
  <ErrorName> -- <when it occurs> | <classification> | <metric> | <caller_behavior> [src: file:LINE]
```

Classification values: retriable, fatal, fatal_for_batch, silent_drop, logged_only

"""

LOCK_INSTRUCTIONS = """### @lock format

Document architecturally significant locks (skip trivial internal mutexes):

```
@lock <LockName>
@kind <sync.Mutex | sync.RWMutex | chan struct{} | atomic | sync.Cond>
@purpose <what data/invariant this lock protects>
@protects <specific fields or state guarded>
@acquired_by [<Function1>, <Function2>]
@ordering <lock ordering constraints>
@source_file <relative/path>
@source_line <line number>
```

"""

OUTPUT_FORMAT_INSTRUCTIONS = """### Output format

```
@module <module-name>
@lang <language>
@code_version git:<current-commit-hash>
@aid_status draft
@aid_generated_by layer2-generator
@depends [<dependency-packages>]
---
[ALL L1 entries preserved, with L2 enhancements on existing entries]
---
[NEW @workflow, @invariants, @antipatterns blocks]
```

Focus on the MOST IMPORTANT architectural knowledge -- the stuff that would take hours to figure out from reading code. Don't document trivial getters. Preserve every L1 entry even if you have nothing to add.
"""
