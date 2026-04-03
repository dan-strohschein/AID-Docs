"""CLI: Generate manifest.aid from .aidocs/ directory."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from aidkit.parser import parse_file


def main() -> None:
    p = argparse.ArgumentParser(prog="aid-manifest-gen-py", description="Generate manifest.aid")
    p.add_argument("--dir", required=True, help="Path to .aidocs/ directory")
    args = p.parse_args()

    aidocs = Path(args.dir)
    if not aidocs.is_dir():
        print(f"Not a directory: {args.dir}", file=sys.stderr)
        sys.exit(1)

    aid_files = sorted(aidocs.glob("*.aid"))
    aid_files = [f for f in aid_files if f.name != "manifest.aid"]

    if not aid_files:
        print(f"No .aid files found in {args.dir}", file=sys.stderr)
        sys.exit(1)

    print("@manifest")
    print("@project TODO")
    print("@aid_version 0.2")

    for aid_path in aid_files:
        try:
            aid, _ = parse_file(str(aid_path))
        except Exception as e:
            print(f"# Warning: could not parse {aid_path.name}: {e}", file=sys.stderr)
            continue

        print()
        print("---")
        print()
        print(f"@package {aid.header.module}")
        print(f"@aid_file {aid_path.name}")
        if aid.header.purpose:
            print(f"@purpose {aid.header.purpose}")


if __name__ == "__main__":
    main()
