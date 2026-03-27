"""Generate benchmark result reports."""

from __future__ import annotations

from collections import defaultdict

from aid_bench.runner import BenchmarkResults


def print_report(results: BenchmarkResults) -> None:
    """Print a formatted benchmark report to stdout."""
    if not results.results:
        print("No results to report.")
        return

    print(f"\n{'=' * 60}")
    print(f"  AID Benchmark Results  —  {results.timestamp}")
    print(f"{'=' * 60}\n")

    # Group by library
    by_library: dict[str, list] = defaultdict(list)
    for r in results.results:
        by_library[r.library].append(r)

    all_conditions = sorted(set(r.condition for r in results.results))

    for library, lib_results in sorted(by_library.items()):
        _print_library_table(library, lib_results, all_conditions)
        print()

    _print_summary(results, all_conditions)
    _print_token_summary(results, all_conditions)


def _print_library_table(
    library: str,
    lib_results: list,
    conditions: list[str],
) -> None:
    """Print the results table for one library."""
    print(f"Library: {library}")
    print("-" * 60)

    # Find all task IDs in order
    task_ids: list[str] = []
    seen: set[str] = set()
    for r in lib_results:
        if r.task_id not in seen:
            task_ids.append(r.task_id)
            seen.add(r.task_id)

    # Build result lookup
    lookup: dict[tuple[str, str], bool] = {}
    for r in lib_results:
        lookup[(r.task_id, r.condition)] = r.passed

    # Column widths
    task_col = max(len(t) for t in task_ids) + 2
    cond_col = max(max(len(c) for c in conditions) + 2, 8)

    # Header
    header = f"{'Task':<{task_col}}"
    for c in conditions:
        header += f" | {c:^{cond_col}}"
    print(header)
    print("-" * len(header))

    # Rows
    for task_id in task_ids:
        # Strip library prefix for display
        display_name = task_id
        if display_name.startswith(f"{library}_"):
            display_name = display_name[len(library) + 1:]

        row = f"{display_name:<{task_col}}"
        for c in conditions:
            passed = lookup.get((task_id, c))
            if passed is None:
                status = "—"
            elif passed:
                status = "PASS"
            else:
                status = "FAIL"
            row += f" | {status:^{cond_col}}"
        print(row)

    # Pass rate row
    print("-" * len(header))
    rate_row = f"{'Pass rate':<{task_col}}"
    for c in conditions:
        total = sum(1 for t in task_ids if (t, c) in lookup)
        passed = sum(1 for t in task_ids if lookup.get((t, c), False))
        pct = f"{100 * passed // total}%" if total else "—"
        rate_row += f" | {pct:^{cond_col}}"
    print(rate_row)


def _print_summary(results: BenchmarkResults, conditions: list[str]) -> None:
    """Print overall pass rates."""
    print(f"{'=' * 60}")
    print("  Overall Pass Rate")
    print(f"{'=' * 60}")

    for condition in conditions:
        cond_results = [r for r in results.results if r.condition == condition]
        total = len(cond_results)
        passed = sum(1 for r in cond_results if r.passed)
        pct = 100 * passed / total if total else 0
        bar = "#" * int(pct / 5) + "." * (20 - int(pct / 5))
        print(f"  {condition:<10}  [{bar}]  {pct:.0f}%  ({passed}/{total})")
    print()


def _print_token_summary(results: BenchmarkResults, conditions: list[str]) -> None:
    """Print average token costs per condition."""
    print(f"{'=' * 60}")
    print("  Average Token Cost (per task)")
    print(f"{'=' * 60}")

    for condition in conditions:
        cond_results = [r for r in results.results if r.condition == condition]
        if not cond_results:
            continue
        avg_input = sum(r.input_tokens for r in cond_results) / len(cond_results)
        avg_output = sum(r.output_tokens for r in cond_results) / len(cond_results)
        print(f"  {condition:<10}  input: {avg_input:,.0f}  output: {avg_output:,.0f}  total: {avg_input + avg_output:,.0f}")
    print()
