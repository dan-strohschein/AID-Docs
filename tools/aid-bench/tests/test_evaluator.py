"""Tests for aid_bench.evaluator code execution harness."""

from __future__ import annotations

from aid_bench.evaluator import evaluate


def test_evaluate_passes_on_trivial_assertion() -> None:
    r = evaluate(
        generated_code="result = 1",
        test_code="assert result == 1",
    )
    assert r.passed is True
    assert r.error is None


def test_evaluate_fails_on_failed_assertion() -> None:
    r = evaluate(
        generated_code="result = 1",
        test_code="assert result == 99",
    )
    assert r.passed is False
    assert r.error is not None


def test_evaluate_syntax_error_in_generated_code() -> None:
    r = evaluate(
        generated_code="result = ((( invalid",
        test_code="assert True",
    )
    assert r.passed is False
    assert r.error is not None
