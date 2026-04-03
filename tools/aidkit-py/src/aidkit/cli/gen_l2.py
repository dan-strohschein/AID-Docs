"""CLI: L2 generation pipeline — generate, review, stale, update."""
from __future__ import annotations

import argparse
import sys

from aidkit.parser import parse_file


def main() -> None:
    p = argparse.ArgumentParser(prog="aid-gen-l2-py", description="L2 AID generation pipeline")
    sub = p.add_subparsers(dest="command", required=True)

    # generate
    gen = sub.add_parser("generate", help="Build L2 generator prompt")
    gen.add_argument("--l1", required=True, help="Path to L1 .aid file")
    gen.add_argument("--source", required=True, help="Path to source directory")
    gen.add_argument("--deps", default="", help="Comma-separated dependency .aid paths")
    gen.add_argument("--old-l1", default="", help="Previous L1 .aid (for incremental)")
    gen.add_argument("--existing-l2", default="", help="Existing L2 .aid (for incremental)")

    # review
    rev = sub.add_parser("review", help="Build L2 reviewer prompt")
    rev.add_argument("--draft", required=True, help="Path to L2 draft .aid")
    rev.add_argument("--project-root", default=".", help="Project root")

    # stale
    stl = sub.add_parser("stale", help="Check stale [src:] references")
    stl.add_argument("--aid", required=True, help="Path to .aid file")
    stl.add_argument("--project-root", default=".", help="Project root")

    # update
    upd = sub.add_parser("update", help="Build incremental update prompt")
    upd.add_argument("--aid", required=True, help="Path to .aid file")
    upd.add_argument("--project-root", default=".", help="Project root")

    args = p.parse_args()

    if args.command == "generate":
        _cmd_generate(args)
    elif args.command == "review":
        _cmd_review(args)
    elif args.command == "stale":
        _cmd_stale(args)
    elif args.command == "update":
        _cmd_update(args)


def _cmd_generate(args: argparse.Namespace) -> None:
    from aidkit.l2.generator import build_generator_prompt
    from aidkit.l2.diff import diff_l1_aids, build_incremental_generator_prompt

    l1, _ = parse_file(args.l1)

    dep_aids = []
    if args.deps:
        for dep_path in args.deps.split(","):
            dep_path = dep_path.strip()
            try:
                dep, _ = parse_file(dep_path)
                dep_aids.append(dep)
            except Exception as e:
                print(f"Warning: could not parse dep {dep_path}: {e}", file=sys.stderr)

    # Incremental mode
    if args.old_l1 and args.existing_l2:
        old_l1, _ = parse_file(args.old_l1)
        existing_l2, _ = parse_file(args.existing_l2)
        diff = diff_l1_aids(old_l1, l1)

        if not diff.new and not diff.modified and not diff.removed:
            print("No L1 changes detected. L2 is up to date.", file=sys.stderr)
            return

        print(
            f"L1 diff: {len(diff.new)} new, {len(diff.modified)} modified, "
            f"{len(diff.unchanged)} unchanged, {len(diff.removed)} removed",
            file=sys.stderr,
        )
        prompt = build_incremental_generator_prompt(l1, existing_l2, diff, args.source, dep_aids)
        print(prompt)
        return

    # Full generation
    prompt = build_generator_prompt(l1, args.source, dep_aids)
    print(prompt)


def _cmd_review(args: argparse.Namespace) -> None:
    from aidkit.l2.reviewer import build_reviewer_prompt

    draft, _ = parse_file(args.draft)
    prompt = build_reviewer_prompt(draft, args.project_root)
    print(prompt)


def _cmd_stale(args: argparse.Namespace) -> None:
    from aidkit.l2.staleness import check_staleness

    aid, _ = parse_file(args.aid)
    stale_claims = check_staleness(aid, args.project_root)

    if not stale_claims:
        print("No stale claims found. AID is up to date.")
        return

    print(f"Found {len(stale_claims)} stale claim(s):\n")
    for sc in stale_claims:
        print(f"  {sc.entry}.{sc.field}: {sc.reason}")
        print(f"    ref: {sc.ref}")
        print(f"    claim: {sc.claim_text}\n")
    sys.exit(1)


def _cmd_update(args: argparse.Namespace) -> None:
    from aidkit.l2.staleness import check_staleness
    from aidkit.l2.incremental import build_incremental_prompt

    aid, _ = parse_file(args.aid)
    stale_claims = check_staleness(aid, args.project_root)

    if not stale_claims:
        print("No stale claims found. AID is up to date.", file=sys.stderr)
        return

    prompt = build_incremental_prompt(aid, stale_claims, args.project_root)
    print(prompt)


if __name__ == "__main__":
    main()
