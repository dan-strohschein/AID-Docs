"""L1 diff with reverse-call propagation for incremental L2 generation."""
from __future__ import annotations

from dataclasses import dataclass, field

from ..model import AidFile, Entry

from .generator import _parse_calls_list


@dataclass
class EntryPair:
    """Old and new versions of a modified entry."""

    old: Entry
    new: Entry


@dataclass
class L1Diff:
    """Categorized entries from comparing old L1 against new L1."""

    new: list[Entry] = field(default_factory=list)
    modified: list[EntryPair] = field(default_factory=list)
    unchanged: list[Entry] = field(default_factory=list)
    removed: list[Entry] = field(default_factory=list)


def diff_l1_aids(old_l1: AidFile, new_l1: AidFile) -> L1Diff:
    """Compare old and new L1 AID files and categorize each entry.

    Comparison key: Kind + ':' + Name (e.g., 'fn:Handler.ServeHTTP').
    Fields compared: @sig, @calls, @params, @source_line.

    After initial classification, reverse-call propagation marks entries
    that @calls a MODIFIED entry as also MODIFIED (one level deep).
    """
    result = L1Diff()

    old_by_key = {_entry_key(e): e for e in old_l1.entries}
    new_by_key = {_entry_key(e): e for e in new_l1.entries}

    modified_keys: set[str] = set()

    for new_entry in new_l1.entries:
        key = _entry_key(new_entry)
        old_entry = old_by_key.get(key)
        if old_entry is not None:
            if _entries_match(old_entry, new_entry):
                result.unchanged.append(new_entry)
            else:
                result.modified.append(EntryPair(old=old_entry, new=new_entry))
                modified_keys.add(key)
        else:
            result.new.append(new_entry)

    for old_entry in old_l1.entries:
        key = _entry_key(old_entry)
        if key not in new_by_key:
            result.removed.append(old_entry)

    # Reverse-call propagation
    _propagate_callers(result, new_l1, modified_keys)

    return result


def _propagate_callers(diff: L1Diff, new_l1: AidFile, modified_keys: set[str]) -> None:
    """Promote UNCHANGED entries to MODIFIED if they call a MODIFIED entry."""
    modified_names: set[str] = set()
    for key in modified_keys:
        idx = key.find(":")
        if idx >= 0:
            modified_names.add(key[idx + 1:])

    still_unchanged: list[Entry] = []
    for entry in diff.unchanged:
        if _calls_modified(entry, modified_names):
            diff.modified.append(EntryPair(old=entry, new=entry))
        else:
            still_unchanged.append(entry)
    diff.unchanged = still_unchanged


def _calls_modified(entry: Entry, modified_names: set[str]) -> bool:
    calls = entry.fields.get("calls")
    if not calls:
        return False
    for callee in _parse_calls_list(calls.inline_value):
        if callee in modified_names:
            return True
        dot_idx = callee.rfind(".")
        if dot_idx >= 0 and callee[dot_idx + 1:] in modified_names:
            return True
    return False


def _entry_key(e: Entry) -> str:
    return f"{e.kind}:{e.name}"


def _entries_match(a: Entry, b: Entry) -> bool:
    return (
        _field_value(a, "sig") == _field_value(b, "sig")
        and _field_value(a, "calls") == _field_value(b, "calls")
        and _field_value(a, "params") == _field_value(b, "params")
        and _field_value(a, "source_line") == _field_value(b, "source_line")
    )


def _field_value(e: Entry, name: str) -> str:
    f = e.fields.get(name)
    return f.value() if f else ""
