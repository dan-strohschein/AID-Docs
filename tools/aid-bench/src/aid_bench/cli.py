"""CLI entry point for aid-bench."""

from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

from aid_bench import __version__


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aid-bench",
        description="Benchmark AID documentation effectiveness for AI agents.",
    )
    sub = parser.add_subparsers(dest="command")

    run_parser = sub.add_parser("run", help="Run the benchmark")
    run_parser.add_argument(
        "--library", "-l",
        action="append",
        default=None,
        help="Library to benchmark (can be repeated). Default: all.",
    )
    run_parser.add_argument(
        "--condition", "-c",
        action="append",
        default=None,
        help="Condition to test (blind/human/aid_l1/aid_full). Default: all.",
    )
    run_parser.add_argument(
        "--output", "-o",
        default=None,
        help="Save results JSON to this file.",
    )
    run_parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Print progress information.",
    )
    run_parser.add_argument(
        "--runs", "-n",
        type=int,
        default=1,
        help="Number of runs for statistical significance. Default: 1.",
    )

    parser.add_argument(
        "--version",
        action="version",
        version=f"%(prog)s {__version__}",
    )

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.command != "run":
        parser.print_help()
        return 0

    from aid_bench.runner import run_benchmark, BenchmarkResults
    from aid_bench.report import print_report

    all_results = BenchmarkResults(
        timestamp=time.strftime("%Y-%m-%d %H:%M:%S"),
    )

    for run_num in range(args.runs):
        if args.runs > 1:
            print(f"\n--- Run {run_num + 1}/{args.runs} ---", file=sys.stderr)

        results = run_benchmark(
            libraries=args.library,
            conditions=args.condition,
            verbose=args.verbose,
        )

        for r in results.results:
            all_results.add(r)

    print_report(all_results)

    if args.output:
        out_path = Path(args.output)
        all_results.save(out_path)
        print(f"Results saved to: {out_path}", file=sys.stderr)

    return 0


if __name__ == "__main__":
    sys.exit(main())
