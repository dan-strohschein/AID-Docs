"""Incremental L2 update prompt builder."""
from __future__ import annotations

from ..model import AidFile, Entry, Field

from .diff import L1Diff
from .generator import (
    CORE_INSTRUCTIONS,
    OUTPUT_FORMAT_INSTRUCTIONS,
    _read_aid_as_text,
)
from .staleness import StaleClaim


def build_incremental_generator_prompt(
    new_l1: AidFile,
    existing_l2: AidFile | None,
    diff: L1Diff,
    source_dir: str,
    dep_aids: list[AidFile] | None = None,
) -> str:
    """Construct a prompt that generates L2 annotations for only NEW and MODIFIED entries."""
    dep_aids = dep_aids or []
    parts: list[str] = []

    parts.append("You are a Layer 2 AID Generator performing an INCREMENTAL update.\n")
    parts.append("Only generate L2 annotations for the new/changed entries listed below.\n\n")

    # Summary of changes
    parts.append("## Change summary\n\n")
    parts.append(f"- New entries: {len(diff.new)}\n")
    parts.append(f"- Modified entries: {len(diff.modified)}\n")
    parts.append(f"- Unchanged entries: {len(diff.unchanged)} (L2 annotations preserved, not shown)\n")
    parts.append(f"- Removed entries: {len(diff.removed)}\n\n")

    # New entries
    if diff.new:
        parts.append("## New entries (generate full L2 annotations)\n\n")
        for entry in diff.new:
            _write_entry(parts, entry)
        parts.append("\n")

    # Modified entries
    if diff.modified:
        parts.append("## Modified entries (update L2 annotations)\n\n")
        for pair in diff.modified:
            parts.append(f"### {pair.new.kind} {pair.new.name}\n\n")
            parts.append("**Updated L1:**\n")
            _write_entry(parts, pair.new)

            if existing_l2 is not None:
                existing_entry = _find_entry(existing_l2, pair.new.kind, pair.new.name)
                if existing_entry is not None:
                    parts.append("\n**Existing L2 annotations (may need updating):**\n")
                    _write_l2_fields(parts, existing_entry)
            parts.append("\n")

    # Source files
    parts.append("## Source files to read\n\n")
    parts.append(f"Source directory: {source_dir}\n\n")
    files = _collect_diff_source_files(diff)
    for f in files:
        parts.append(f"- {f}\n")
    parts.append("\n")

    # Dependency AIDs
    if dep_aids:
        parts.append("## Related package AIDs\n\n")
        for dep in dep_aids:
            parts.append(_read_aid_as_text(dep))
            parts.append("\n\n---\n\n")

    parts.append(CORE_INSTRUCTIONS)
    parts.append("""### Incremental generation rules

1. Generate L2 annotations (@purpose enhancement, @pre, @post, @errors, @thread_safety) for each NEW entry.
2. Update L2 annotations for each MODIFIED entry -- the existing L2 is shown for context.
3. Regenerate @workflow, @invariants, and @antipatterns blocks for the whole module (they cross-reference multiple functions).
4. Output ONLY the new/updated entries and module-level blocks. Do NOT output unchanged entries.
5. Preserve @fn name, @sig, @params, @calls, @source_file, @source_line exactly from the L1 above.

""")
    parts.append(OUTPUT_FORMAT_INSTRUCTIONS)

    return "".join(parts)


def build_incremental_prompt(
    aid_file: AidFile,
    stale_claims: list[StaleClaim],
    project_root: str,
) -> str:
    """Construct a prompt to re-generate only stale claims."""
    parts: list[str] = []

    parts.append("You are a Layer 2 AID Updater. Some claims in this AID file are stale because ")
    parts.append("the referenced source code has changed. Re-verify and update ONLY the stale claims listed below.\n\n")

    parts.append("## Current AID file\n\n")
    parts.append(f"Module: {aid_file.header.module}\n")
    parts.append(f"Code version: {aid_file.header.code_version}\n\n")

    parts.append("## Stale claims to update\n\n")
    parts.append(f"{len(stale_claims)} claim(s) need re-verification:\n\n")

    file_set: set[str] = set()
    for i, sc in enumerate(stale_claims, 1):
        parts.append(f"### Stale claim {i}\n")
        parts.append(f"- **Entry:** {sc.entry}\n")
        parts.append(f"- **Field:** {sc.field}\n")
        parts.append(f"- **Reference:** {sc.ref}\n")
        parts.append(f"- **Reason:** {sc.reason}\n")
        parts.append(f"- **Current claim:** {sc.claim_text}\n\n")
        file_set.add(sc.ref.file)

    parts.append("## Source files to read\n\n")
    parts.append(f"Project root: {project_root}\n\n")
    for file in file_set:
        parts.append(f"- {file}\n")
    parts.append("\n")

    parts.append(INCREMENTAL_INSTRUCTIONS)

    return "".join(parts)


INCREMENTAL_INSTRUCTIONS = """## Instructions

For each stale claim above:
1. Read the referenced source file at the current line numbers
2. Determine if the claim is still accurate, needs updating, or should be removed
3. Output the updated claim with corrected [src:] references

Output format:
```
### Claim 1: [UPDATED | UNCHANGED | REMOVED]
@field_name updated content here
  with continuation lines if needed [src: file:new-line-numbers]
```

Only output changes for claims that actually need updating. If a claim is still accurate
(the code changed but the invariant still holds), mark it UNCHANGED and update the line numbers.

DO NOT re-generate the entire AID file. Only update the specific stale claims listed above.
"""


def _collect_diff_source_files(diff: L1Diff) -> list[str]:
    file_set: set[str] = set()
    for entry in diff.new:
        sf = entry.fields.get("source_file")
        if sf and sf.inline_value:
            file_set.add(sf.inline_value)
    for pair in diff.modified:
        sf = pair.new.fields.get("source_file")
        if sf and sf.inline_value:
            file_set.add(sf.inline_value)
    return sorted(file_set)


def _find_entry(aid: AidFile, kind: str, name: str) -> Entry | None:
    for e in aid.entries:
        if e.kind == kind and e.name == name:
            return e
    return None


def _write_entry(parts: list[str], e: Entry) -> None:
    parts.append(f"@{e.kind} {e.name}\n")
    for name, field in e.fields.items():
        if name == e.kind:
            continue
        if field.inline_value:
            parts.append(f"@{name} {field.inline_value}\n")
        else:
            parts.append(f"@{name}\n")
        for line in field.lines:
            parts.append(f"  {line}\n")


def _write_l2_fields(parts: list[str], e: Entry) -> None:
    l2_fields = ["purpose", "pre", "post", "errors", "effects", "thread_safety", "complexity"]
    for name in l2_fields:
        field = e.fields.get(name)
        if field is None:
            continue
        if field.inline_value:
            parts.append(f"@{name} {field.inline_value}\n")
        elif field.lines:
            parts.append(f"@{name}\n")
            for line in field.lines:
                parts.append(f"  {line}\n")
