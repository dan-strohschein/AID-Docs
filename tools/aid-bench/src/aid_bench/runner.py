"""Orchestrate benchmark runs across libraries, tasks, and conditions."""

from __future__ import annotations

import importlib.util
import json
import re
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path

import anthropic

from aid_bench.conditions import build_prompt
from aid_bench.evaluator import EvalResult, evaluate


TASKS_DIR = Path(__file__).parent.parent.parent / "tasks"
CONDITIONS = ["blind", "human", "aid_l1", "aid_full"]
MODEL = "claude-sonnet-4-20250514"


@dataclass
class TaskResult:
    """Result for one task under one condition."""
    library: str
    task_id: str
    condition: str
    passed: bool
    error: str | None
    generated_code: str
    input_tokens: int
    output_tokens: int


@dataclass
class BenchmarkResults:
    """All results for a benchmark run."""
    results: list[TaskResult] = field(default_factory=list)
    timestamp: str = ""

    def add(self, result: TaskResult) -> None:
        self.results.append(result)

    def save(self, path: Path) -> None:
        data = {
            "timestamp": self.timestamp,
            "results": [
                {
                    "library": r.library,
                    "task_id": r.task_id,
                    "condition": r.condition,
                    "passed": r.passed,
                    "error": r.error,
                    "generated_code": r.generated_code,
                    "input_tokens": r.input_tokens,
                    "output_tokens": r.output_tokens,
                }
                for r in self.results
            ],
        }
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data, indent=2), encoding="utf-8")


def load_tasks(library: str) -> list[dict]:
    """Load task definitions for a library."""
    tasks_file = TASKS_DIR / library / "tasks.py"
    if not tasks_file.exists():
        raise FileNotFoundError(f"No tasks file: {tasks_file}")

    spec = importlib.util.spec_from_file_location(f"tasks_{library}", tasks_file)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.TASKS


def run_benchmark(
    libraries: list[str] | None = None,
    conditions: list[str] | None = None,
    verbose: bool = False,
) -> BenchmarkResults:
    """Run the full benchmark."""
    client = anthropic.Anthropic()

    if libraries is None:
        # Discover all libraries with tasks
        libraries = [
            d.name for d in TASKS_DIR.iterdir()
            if d.is_dir() and (d / "tasks.py").exists()
        ]

    conditions = conditions or CONDITIONS
    results = BenchmarkResults(
        timestamp=time.strftime("%Y-%m-%d %H:%M:%S"),
    )

    for library in sorted(libraries):
        tasks = load_tasks(library)
        task_dir = TASKS_DIR / library

        for task in tasks:
            for condition in conditions:
                if verbose:
                    print(
                        f"  [{library}] {task['id']} / {condition}...",
                        end=" ",
                        flush=True,
                        file=sys.stderr,
                    )

                result = _run_single(
                    client=client,
                    library=library,
                    task=task,
                    condition=condition,
                    task_dir=task_dir,
                    verbose=verbose,
                )
                results.add(result)

                if verbose:
                    status = "PASS" if result.passed else f"FAIL ({result.error})"
                    print(status, file=sys.stderr)

    return results


def _run_single(
    client: anthropic.Anthropic,
    library: str,
    task: dict,
    condition: str,
    task_dir: Path,
    verbose: bool,
) -> TaskResult:
    """Run a single task under a single condition."""
    system_prompt, user_prompt = build_prompt(
        task_description=task["description"],
        condition=condition,
        task_dir=task_dir,
    )

    # Call Claude API
    try:
        response = client.messages.create(
            model=MODEL,
            max_tokens=2048,
            system=system_prompt,
            messages=[{"role": "user", "content": user_prompt}],
        )

        generated_code = response.content[0].text
        input_tokens = response.usage.input_tokens
        output_tokens = response.usage.output_tokens

        # Strip markdown code fences if the model included them
        generated_code = _strip_code_fences(generated_code)

    except Exception as e:
        return TaskResult(
            library=library,
            task_id=task["id"],
            condition=condition,
            passed=False,
            error=f"API error: {e}",
            generated_code="",
            input_tokens=0,
            output_tokens=0,
        )

    # Evaluate the generated code
    eval_result = evaluate(
        generated_code=generated_code,
        test_code=task["test"],
        setup_code=task.get("setup", ""),
        teardown_code=task.get("teardown", ""),
    )

    return TaskResult(
        library=library,
        task_id=task["id"],
        condition=condition,
        passed=eval_result.passed,
        error=eval_result.error,
        generated_code=generated_code,
        input_tokens=input_tokens,
        output_tokens=output_tokens,
    )


def _strip_code_fences(text: str) -> str:
    """Remove markdown code fences if present."""
    text = text.strip()
    # Remove ```python ... ``` or ``` ... ```
    if text.startswith("```"):
        lines = text.splitlines()
        # Remove first line (```python or ```)
        lines = lines[1:]
        # Remove last line if it's ```
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        text = "\n".join(lines)
    return text
