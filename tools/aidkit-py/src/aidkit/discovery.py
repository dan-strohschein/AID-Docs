"""AID file discovery protocol per spec section 10.5."""
from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

from .model import AidFile
from .parser import parse_file

AIDOCS_DIR = ".aidocs"
MANIFEST_FILE = "manifest.aid"


@dataclass
class DiscoveryResult:
    """Holds the discovery outcome."""

    aidocs_path: str = ""
    manifest_path: str = ""
    manifest: AidFile | None = None
    aid_files: list[str] = field(default_factory=list)


def discover(start_dir: str | Path) -> DiscoveryResult | None:
    """Walk up from start_dir looking for .aidocs/ per the spec protocol."""
    current = Path(start_dir).resolve()

    while True:
        candidate = current / AIDOCS_DIR
        if candidate.is_dir():
            return _inspect_aidocs(candidate)

        parent = current.parent
        if parent == current:
            break
        current = parent

    return None


def _inspect_aidocs(aidocs_path: Path) -> DiscoveryResult:
    result = DiscoveryResult(aidocs_path=str(aidocs_path))

    # List .aid files
    for entry in sorted(aidocs_path.iterdir()):
        if entry.is_file() and entry.suffix == ".aid":
            result.aid_files.append(entry.name)

    # Check for manifest
    manifest_path = aidocs_path / MANIFEST_FILE
    if manifest_path.is_file():
        result.manifest_path = str(manifest_path)
        try:
            manifest, _ = parse_file(manifest_path)
            result.manifest = manifest
        except Exception:
            pass

    return result
