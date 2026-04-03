"""CLI: Parse .aid files and output JSON or summary."""
from __future__ import annotations

import argparse
import json
import sys
from dataclasses import asdict

from aidkit.parser import parse_file


def main() -> None:
    p = argparse.ArgumentParser(prog="aid-parse-py", description="Parse .aid files")
    p.add_argument("file", help="Path to .aid file")
    p.add_argument("--json", action="store_true", help="Output as JSON")
    p.add_argument("--summary", action="store_true", help="Show summary only")
    args = p.parse_args()

    aid, warnings = parse_file(args.file)

    for w in warnings:
        print(f"warning: {w}", file=sys.stderr)

    if args.json:
        print(json.dumps(asdict(aid), indent=2, default=str))
    elif args.summary:
        fns = sum(1 for e in aid.entries if e.kind == "fn")
        types = sum(1 for e in aid.entries if e.kind == "type")
        traits = sum(1 for e in aid.entries if e.kind == "trait")
        consts = sum(1 for e in aid.entries if e.kind == "const")
        print(f"Module: {aid.header.module}")
        print(f"Entries: {fns} fn, {types} type, {traits} trait, {consts} const")
        print(f"Workflows: {len(aid.workflows)}")
        print(f"Annotations: {len(aid.annotations)}")
    else:
        from aidkit.emitter import emit
        print(emit(aid))


if __name__ == "__main__":
    main()
