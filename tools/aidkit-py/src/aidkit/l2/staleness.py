"""Git-based staleness checking for AID source references."""
from __future__ import annotations

import subprocess
from dataclasses import dataclass

from ..model import AidFile, Field, SourceRef


@dataclass
class StaleClaim:
    """A claim whose source reference may be outdated."""

    entry: str
    field: str
    ref: SourceRef
    reason: str  # "file changed", "lines changed", "file deleted"
    claim_text: str


def check_staleness(aid_file: AidFile, project_root: str) -> list[StaleClaim]:
    """Compare an AID file's @code_version against the current git HEAD.

    Reports which [src:] references point to changed code.

    Raises ValueError if @code_version is missing or malformed.
    """
    code_version = aid_file.header.code_version
    if not code_version:
        raise ValueError("no @code_version in AID file")

    if not code_version.startswith("git:"):
        raise ValueError(f"@code_version {code_version!r} doesn't start with 'git:'")

    commit_hash = code_version[4:]

    current_head = _git_head(project_root)
    if current_head.startswith(commit_hash) or commit_hash.startswith(current_head):
        return []

    changed_files = _git_changed_files(project_root, commit_hash, current_head)
    changed_set = set(changed_files)

    stale: list[StaleClaim] = []

    def check_fields(entry_name: str, fields: dict[str, Field]) -> None:
        for field in fields.values():
            for ref in field.source_refs:
                if ref.file in changed_set:
                    lines_changed = _git_lines_changed(
                        project_root, commit_hash, current_head,
                        ref.file, ref.start_line, ref.end_line,
                    )
                    if lines_changed:
                        stale.append(StaleClaim(
                            entry=entry_name,
                            field=field.name,
                            ref=ref,
                            reason="lines changed",
                            claim_text=_truncate(field.value(), 100),
                        ))

    for e in aid_file.entries:
        check_fields(e.name, e.fields)
    for w in aid_file.workflows:
        check_fields(f"workflow:{w.name}", w.fields)

    return stale


def _git_head(project_root: str) -> str:
    result = subprocess.run(
        ["git", "rev-parse", "--short", "HEAD"],
        cwd=project_root, capture_output=True, text=True, check=True,
    )
    return result.stdout.strip()


def _git_changed_files(project_root: str, from_hash: str, to_hash: str) -> list[str]:
    result = subprocess.run(
        ["git", "diff", "--name-only", from_hash, to_hash],
        cwd=project_root, capture_output=True, text=True, check=True,
    )
    return [l.strip() for l in result.stdout.strip().split("\n") if l.strip()]


def _git_lines_changed(
    project_root: str,
    from_hash: str,
    to_hash: str,
    file_path: str,
    start_line: int,
    end_line: int,
) -> bool:
    """Check if specific lines in a file were modified between two commits."""
    try:
        result = subprocess.run(
            ["git", "diff", from_hash, to_hash, "--", file_path],
            cwd=project_root, capture_output=True, text=True, check=True,
        )
    except subprocess.CalledProcessError:
        return True  # Assume changed on error

    if not result.stdout:
        return False

    for line in result.stdout.split("\n"):
        if not line.startswith("@@"):
            continue
        hunk_start, hunk_end = _parse_hunk_range(line)
        if hunk_start <= end_line and hunk_end >= start_line:
            return True

    return False


def _parse_hunk_range(hunk_line: str) -> tuple[int, int]:
    """Parse @@ -oldStart,oldCount +newStart,newCount @@ format."""
    for part in hunk_line.split():
        if part.startswith("+") and "," in part:
            part = part.lstrip("+")
            nums = part.split(",")
            if len(nums) == 2:
                s = int(nums[0])
                c = int(nums[1])
                return s, s + c - 1
        elif part.startswith("+"):
            part = part.lstrip("+")
            try:
                s = int(part)
                return s, s
            except ValueError:
                continue
    return 0, 0


def _truncate(s: str, max_len: int) -> str:
    s = s.replace("\n", " ")
    if len(s) > max_len:
        return s[: max_len - 3] + "..."
    return s
