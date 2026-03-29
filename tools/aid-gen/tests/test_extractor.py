"""Tests for aid_gen.extractor orchestration (filesystem, discovery, output)."""

from __future__ import annotations

from pathlib import Path

import pytest

from aid_gen.extractor import extract


def _minimal_module() -> str:
    return '''"""A tiny module for extraction tests."""
def public_fn(x: int) -> int:
    """Return x."""
    return x
'''


def test_extract_single_file_writes_aid(tmp_path: Path) -> None:
    src = tmp_path / "hello_world.py"
    src.write_text(_minimal_module(), encoding="utf-8")
    out = tmp_path / "out"
    extract(str(src), output_dir=str(out), stdout=False)
    # stem hello_world -> module hello/world -> filename hello-world.aid
    dest = out / "hello-world.aid"
    assert dest.is_file()
    text = dest.read_text(encoding="utf-8")
    assert "@module hello/world" in text
    assert "@fn public_fn" in text


def test_extract_stdout_contains_module(tmp_path: Path, capsys: pytest.CaptureFixture[str]) -> None:
    src = tmp_path / "solo.py"
    src.write_text(_minimal_module(), encoding="utf-8")
    extract(str(src), stdout=True, module_name="custom/mod")
    captured = capsys.readouterr()
    assert "@module custom/mod" in captured.out
    assert "@fn public_fn" in captured.out


def test_extract_path_not_found() -> None:
    with pytest.raises(FileNotFoundError, match="Path not found"):
        extract("/nonexistent/path/that/does/not/exist.py")


def test_extract_not_python_file(tmp_path: Path) -> None:
    bad = tmp_path / "readme.txt"
    bad.write_text("x", encoding="utf-8")
    with pytest.raises(ValueError, match="Not a Python file"):
        extract(str(bad))


def test_extract_directory_discovers_modules(tmp_path: Path) -> None:
    pkg = tmp_path / "mypkg"
    pkg.mkdir()
    (pkg / "__init__.py").write_text('"""Package."""\n', encoding="utf-8")
    (pkg / "util.py").write_text(_minimal_module(), encoding="utf-8")
    out = tmp_path / "aid_out"
    extract(str(pkg), output_dir=str(out), stdout=False)
    init_aid = out / "mypkg.aid"
    util_aid = out / "util.aid"
    assert init_aid.is_file(), list(out.iterdir())
    assert util_aid.is_file(), list(out.iterdir())


def test_extract_directory_skips_test_files(tmp_path: Path) -> None:
    pkg = tmp_path / "pkg"
    pkg.mkdir()
    (pkg / "normal.py").write_text(_minimal_module(), encoding="utf-8")
    (pkg / "test_foo.py").write_text(_minimal_module(), encoding="utf-8")
    (pkg / "bar_test.py").write_text(_minimal_module(), encoding="utf-8")
    out = tmp_path / "out"
    extract(str(pkg), output_dir=str(out), stdout=False)
    assert (out / "normal.aid").is_file()
    assert not (out / "test-foo.aid").exists()
    assert not (out / "bar-test.aid").exists()


def test_extract_exclude_glob(tmp_path: Path) -> None:
    pkg = tmp_path / "pkg"
    pkg.mkdir()
    (pkg / "keep.py").write_text(_minimal_module(), encoding="utf-8")
    sub = pkg / "skipme"
    sub.mkdir()
    (sub / "ignored.py").write_text(_minimal_module(), encoding="utf-8")
    out = tmp_path / "out"
    extract(str(pkg), output_dir=str(out), exclude=["skipme/*"], stdout=False)
    assert (out / "keep.aid").is_file()
    assert not (out / "ignored.aid").exists()


def test_extract_skips_nested_skip_dirs(tmp_path: Path) -> None:
    pkg = tmp_path / "root"
    pkg.mkdir()
    (pkg / "ok.py").write_text(_minimal_module(), encoding="utf-8")
    venv = pkg / ".venv" / "lib"
    venv.mkdir(parents=True)
    (venv / "site.py").write_text(_minimal_module(), encoding="utf-8")
    out = tmp_path / "out"
    extract(str(pkg), output_dir=str(out), stdout=False)
    assert (out / "ok.aid").is_file()
    assert not any(p.name.startswith("site") for p in out.glob("*.aid"))


def test_extract_empty_directory_no_crash(tmp_path: Path, capsys: pytest.CaptureFixture[str]) -> None:
    empty = tmp_path / "empty"
    empty.mkdir()
    extract(str(empty), output_dir=str(tmp_path / "out"), verbose=True, stdout=False)
    err = capsys.readouterr().err
    assert "No Python files found" in err
