"""Execute generated code and evaluate results."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import textwrap
from dataclasses import dataclass
from pathlib import Path


@dataclass
class EvalResult:
    """Result of evaluating a single generated code sample."""
    passed: bool
    error: str | None = None
    generated_code: str = ""
    stdout: str = ""
    stderr: str = ""


def evaluate(
    generated_code: str,
    test_code: str,
    setup_code: str = "",
    teardown_code: str = "",
    timeout: int = 30,
) -> EvalResult:
    """Execute generated code and run test assertions.

    The generated code must assign its result to a variable called `result`.
    The test code can reference `result` and use assert statements.
    """
    # Build the full script
    script_parts: list[str] = [
        "import os, sys, tempfile, json, csv, sqlite3, tarfile, io, pathlib",
        "",
    ]

    if setup_code:
        script_parts.append("# --- Setup ---")
        script_parts.append(setup_code)
        script_parts.append("")

    script_parts.append("# --- Generated code ---")
    script_parts.append(generated_code)
    script_parts.append("")

    script_parts.append("# --- Test assertions ---")
    script_parts.append("try:")
    # Indent the test code under try
    for line in test_code.strip().splitlines():
        script_parts.append(f"    {line}")
    script_parts.append('    print("BENCH_PASS")')
    script_parts.append("except AssertionError as e:")
    script_parts.append('    print(f"BENCH_FAIL: {e}")')
    script_parts.append("except Exception as e:")
    script_parts.append('    print(f"BENCH_ERROR: {type(e).__name__}: {e}")')

    if teardown_code:
        script_parts.append("")
        script_parts.append("# --- Teardown ---")
        script_parts.append(teardown_code)

    full_script = "\n".join(script_parts)

    # Write to temp file and execute
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".py", delete=False, encoding="utf-8"
    ) as f:
        f.write(full_script)
        script_path = f.name

    try:
        proc = subprocess.run(
            [sys.executable, script_path],
            capture_output=True,
            text=True,
            timeout=timeout,
            cwd=tempfile.gettempdir(),
        )

        stdout = proc.stdout.strip()
        stderr = proc.stderr.strip()

        if "BENCH_PASS" in stdout:
            return EvalResult(
                passed=True,
                generated_code=generated_code,
                stdout=stdout,
                stderr=stderr,
            )
        elif "BENCH_FAIL" in stdout:
            error_line = next(
                (l for l in stdout.splitlines() if "BENCH_FAIL" in l), ""
            )
            return EvalResult(
                passed=False,
                error=error_line.replace("BENCH_FAIL: ", ""),
                generated_code=generated_code,
                stdout=stdout,
                stderr=stderr,
            )
        elif "BENCH_ERROR" in stdout:
            error_line = next(
                (l for l in stdout.splitlines() if "BENCH_ERROR" in l), ""
            )
            return EvalResult(
                passed=False,
                error=error_line.replace("BENCH_ERROR: ", ""),
                generated_code=generated_code,
                stdout=stdout,
                stderr=stderr,
            )
        else:
            # Code crashed before reaching test assertions
            error = stderr if stderr else "Code produced no output"
            if proc.returncode != 0:
                error = f"Exit code {proc.returncode}: {stderr}"
            return EvalResult(
                passed=False,
                error=error,
                generated_code=generated_code,
                stdout=stdout,
                stderr=stderr,
            )

    except subprocess.TimeoutExpired:
        return EvalResult(
            passed=False,
            error=f"Timeout after {timeout}s",
            generated_code=generated_code,
        )
    finally:
        Path(script_path).unlink(missing_ok=True)
