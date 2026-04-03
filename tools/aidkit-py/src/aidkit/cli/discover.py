"""CLI: Find .aidocs/ directory and list packages."""
from __future__ import annotations

import argparse
import sys

from aidkit.discovery import discover


def main() -> None:
    p = argparse.ArgumentParser(prog="aid-discover-py", description="Find .aidocs/ directory")
    p.add_argument("--dir", default=".", help="Start directory (default: .)")
    args = p.parse_args()

    result = discover(args.dir)

    if result is None:
        print("No .aidocs/ directory found.", file=sys.stderr)
        sys.exit(1)

    print(f"AID docs: {result.aidocs_path}")
    if result.manifest_path:
        print(f"Manifest: {result.manifest_path}")
    print(f"AID files: {len(result.aid_files)}")
    for f in sorted(result.aid_files):
        print(f"  {f}")


if __name__ == "__main__":
    main()
