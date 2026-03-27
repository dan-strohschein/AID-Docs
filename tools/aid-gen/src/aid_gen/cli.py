"""CLI entry point for aid-gen."""

import argparse
import sys

from aid_gen import __version__


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aid-gen",
        description="Generate AID (Agent Interface Document) files from Python source code.",
    )
    parser.add_argument(
        "path",
        help="Python file or directory to extract from",
    )
    parser.add_argument(
        "--output", "-o",
        default=".aidocs",
        help="Output directory for .aid files (default: .aidocs/)",
    )
    parser.add_argument(
        "--stdout",
        action="store_true",
        help="Print .aid output to stdout instead of writing files",
    )
    parser.add_argument(
        "--module",
        help="Override the auto-detected module name",
    )
    parser.add_argument(
        "--version-tag",
        help="Set the library version in the AID header",
    )
    parser.add_argument(
        "--exclude",
        action="append",
        default=[],
        help="Glob pattern for files to skip (can be repeated)",
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Print progress information",
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

    # Import here to keep --help fast
    from aid_gen.extractor import extract

    try:
        extract(
            path=args.path,
            output_dir=args.output,
            stdout=args.stdout,
            module_name=args.module,
            version=args.version_tag,
            exclude=args.exclude,
            verbose=args.verbose,
        )
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
