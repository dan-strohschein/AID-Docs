"""L2 reviewer prompt builder."""
from __future__ import annotations

import os
from pathlib import Path

from ..model import AidFile, SourceRef

from .generator import _read_aid_as_text


def build_reviewer_prompt(l2_draft: AidFile, project_root: str) -> str:
    """Construct the prompt for a Layer 2 reviewer agent.

    The reviewer reads the L2 draft and checks every [src:] reference against source.
    """
    parts: list[str] = []

    parts.append("You are a Layer 2 AID Reviewer. Verify the accuracy of every source-linked claim.\n\n")

    # L2 draft content
    parts.append("## Layer 2 AID Draft to Review\n\n")
    parts.append(_read_aid_as_text(l2_draft))
    parts.append("\n\n")

    # Collect all source refs and list files to read
    refs = _collect_all_source_refs(l2_draft)
    if refs:
        parts.append("## Source files to verify against\n\n")
        parts.append(f"Project root: {project_root}\n\n")

        file_set: set[str] = {ref.file for ref in refs}
        parts.append("Read ONLY these files (the ones referenced by [src:] links):\n\n")
        for file in file_set:
            full_path = os.path.join(project_root, file)
            if os.path.exists(full_path):
                parts.append(f"- {full_path}\n")
            else:
                parts.append(f"- {full_path} (WARNING: file not found)\n")
        parts.append("\n")

    parts.append(REVIEWER_INSTRUCTIONS)

    return "".join(parts)


def _collect_all_source_refs(f: AidFile) -> list[SourceRef]:
    """Gather all [src:] references from an AID file."""
    refs: list[SourceRef] = []
    for e in f.entries:
        for field in e.fields.values():
            refs.extend(field.source_refs)
    for w in f.workflows:
        for field in w.fields.values():
            refs.extend(field.source_refs)
    return refs


REVIEWER_INSTRUCTIONS = """## Instructions

For each claim with a [src: file:line] reference:
1. Read the referenced source file at the specified lines
2. Verify the claim matches what the code actually does
3. Record your findings

## Output format

```
## Verification Report

### Verified claims (accurate)
- [claim summary] -- [src: file:line] -- VERIFIED: [brief confirmation]

### Corrected claims (inaccurate)
- [claim summary] -- [src: file:line] -- CORRECTED: [what was wrong] -> [what it should say]

### Missing claims (reviewer additions)
- [new claim] -- [src: file:line] -- ADDED: [why this matters]

### Stale references (line numbers wrong)
- [claim summary] -- [src: file:line] -- STALE: [correct location]

### Summary
- Total claims checked: N
- Verified accurate: N
- Corrected: N
- Added: N
- Stale references: N
```

Focus on the MOST IMPORTANT claims first: workflow steps, invariants, antipatterns.
DO NOT read CLAUDE.md or README.md.
"""
