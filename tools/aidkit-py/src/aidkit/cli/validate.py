"""CLI: Validate .aid files against spec rules."""
from __future__ import annotations

import argparse
import sys

from aidkit.parser import parse_file
from aidkit.validator import validate


def main() -> None:
    p = argparse.ArgumentParser(prog="aid-validate-py", description="Validate .aid files")
    p.add_argument("files", nargs="+", help="Path(s) to .aid files")
    args = p.parse_args()

    total_errors = 0
    for path in args.files:
        aid, warnings = parse_file(path)
        issues = validate(aid)

        for w in warnings:
            print(f"{path}: parse warning: {w}", file=sys.stderr)

        errors = [i for i in issues if i.severity == "error"]
        warns = [i for i in issues if i.severity == "warning"]

        for issue in issues:
            prefix = "ERROR" if issue.severity == "error" else "WARN"
            entry = f" [{issue.entry}]" if issue.entry else ""
            print(f"{path}{entry}: {prefix}: {issue.message} ({issue.rule})")

        total_errors += len(errors)

    if total_errors > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
