"""Tests for aid_gen.cli entrypoint."""

from __future__ import annotations

from pathlib import Path

from aid_gen.cli import build_parser, main


def test_build_parser_defines_expected_flags() -> None:
    p = build_parser()
    # Exercise argparse by parsing minimal valid argv
    args = p.parse_args(["/some/path.py"])
    assert args.path == "/some/path.py"
    assert args.output == ".aidocs"
    assert args.stdout is False
    assert args.module is None
    assert args.version_tag is None
    assert args.exclude == []
    assert args.verbose is False

    args2 = p.parse_args(
        ["-o", "/out", "--stdout", "--module", "m", "--version-tag", "1.0", "--exclude", "x/*", "-v", "pkg"]
    )
    assert args2.output == "/out"
    assert args2.stdout is True
    assert args2.module == "m"
    assert args2.version_tag == "1.0"
    assert args2.exclude == ["x/*"]
    assert args2.verbose is True
    assert args2.path == "pkg"


def test_main_success_on_valid_file(tmp_path: Path) -> None:
    src = tmp_path / "mod.py"
    src.write_text(
        '"""Doc."""\ndef f() -> None:\n    pass\n',
        encoding="utf-8",
    )
    out = tmp_path / "aidocs"
    rc = main([str(src), "-o", str(out)])
    assert rc == 0
    assert (out / "mod.aid").is_file()


def test_main_returns_1_on_missing_path() -> None:
    rc = main(["/this/path/does/not/exist/zzzz.py"])
    assert rc == 1


def test_main_returns_1_on_non_python_file(tmp_path: Path) -> None:
    f = tmp_path / "nope.txt"
    f.write_text("x", encoding="utf-8")
    rc = main([str(f)])
    assert rc == 1
