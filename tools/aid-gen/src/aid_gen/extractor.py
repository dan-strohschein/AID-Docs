"""Main orchestrator for AID generation."""

from __future__ import annotations

import fnmatch
import os
import sys
from pathlib import Path

from aid_gen.emitter import emit
from aid_gen.python.parser import extract_module


# Files to always skip
_SKIP_FILES = {
    "setup.py", "conftest.py", "noxfile.py", "fabfile.py",
}

# Directory names to always skip
_SKIP_DIRS = {
    "__pycache__", ".git", ".venv", "venv", "env",
    "node_modules", ".tox", ".nox", ".mypy_cache",
    ".pytest_cache", "dist", "build", "egg-info",
}


def extract(
    path: str,
    output_dir: str = ".aidocs",
    stdout: bool = False,
    module_name: str | None = None,
    version: str | None = None,
    exclude: list[str] | None = None,
    verbose: bool = False,
) -> None:
    """Extract AID files from Python source code."""
    target = Path(path)

    if not target.exists():
        raise FileNotFoundError(f"Path not found: {path}")

    version = version or "0.0.0"
    exclude = exclude or []

    if target.is_file():
        if not target.suffix == ".py":
            raise ValueError(f"Not a Python file: {path}")
        _extract_file(target, module_name, version, stdout, output_dir, verbose)
    elif target.is_dir():
        _extract_directory(target, module_name, version, stdout, output_dir, exclude, verbose)
    else:
        raise ValueError(f"Not a file or directory: {path}")


def _extract_file(
    file_path: Path,
    module_name: str | None,
    version: str,
    stdout: bool,
    output_dir: str,
    verbose: bool,
) -> None:
    """Extract a single Python file."""
    if not module_name:
        module_name = _file_to_module_name(file_path)

    if verbose:
        print(f"Extracting: {file_path} → {module_name}", file=sys.stderr)

    source = file_path.read_text(encoding="utf-8")

    try:
        aid_file = extract_module(source, module_name, version)
    except SyntaxError as e:
        print(f"Warning: Could not parse {file_path}: {e}", file=sys.stderr)
        return

    output = emit(aid_file)

    if stdout:
        print(output)
    else:
        out_path = Path(output_dir)
        out_path.mkdir(parents=True, exist_ok=True)
        aid_filename = module_name.replace("/", "-").replace(".", "-") + ".aid"
        dest = out_path / aid_filename
        dest.write_text(output, encoding="utf-8")
        if verbose:
            print(f"  → {dest}", file=sys.stderr)


def _extract_directory(
    dir_path: Path,
    base_module_name: str | None,
    version: str,
    stdout: bool,
    output_dir: str,
    exclude: list[str],
    verbose: bool,
) -> None:
    """Extract all Python files in a directory."""
    py_files = _discover_python_files(dir_path, exclude)

    if not py_files:
        if verbose:
            print(f"No Python files found in {dir_path}", file=sys.stderr)
        return

    for file_path in sorted(py_files):
        module_name = base_module_name or _path_to_module_name(file_path, dir_path)
        # For __init__.py, use the package name
        if file_path.name == "__init__.py":
            module_name = _path_to_module_name(file_path.parent, dir_path.parent)

        _extract_file(file_path, module_name, version, stdout, output_dir, verbose)


def _discover_python_files(dir_path: Path, exclude: list[str]) -> list[Path]:
    """Find all .py files in a directory, respecting exclusions."""
    py_files: list[Path] = []

    for root, dirs, files in os.walk(dir_path):
        root_path = Path(root)

        # Prune skipped directories
        dirs[:] = [d for d in dirs if d not in _SKIP_DIRS]

        for name in files:
            if not name.endswith(".py"):
                continue
            if name in _SKIP_FILES:
                continue
            if name.startswith("test_") or name.endswith("_test.py"):
                continue

            file_path = root_path / name

            # Check exclude patterns
            rel_path = str(file_path.relative_to(dir_path))
            if any(fnmatch.fnmatch(rel_path, pat) for pat in exclude):
                continue

            py_files.append(file_path)

    return py_files


def _file_to_module_name(file_path: Path) -> str:
    """Convert a file path to a module name."""
    stem = file_path.stem
    return stem.replace("_", "/")


def _path_to_module_name(file_path: Path, base_dir: Path) -> str:
    """Convert a file path relative to a base dir to a module name."""
    try:
        rel = file_path.relative_to(base_dir)
    except ValueError:
        return file_path.stem

    parts = list(rel.parts)

    # Remove .py extension from last part
    if parts and parts[-1].endswith(".py"):
        parts[-1] = parts[-1][:-3]

    # Join with /
    return "/".join(parts)
