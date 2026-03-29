"""Tests for aid_bench.conditions prompt building."""

from __future__ import annotations

from pathlib import Path

import pytest

from aid_bench.conditions import SYSTEM_PROMPT, build_prompt


def test_build_prompt_unknown_condition(tmp_path: Path) -> None:
    with pytest.raises(ValueError, match="Unknown condition"):
        build_prompt("do something", "not_a_real_condition", tmp_path)


def test_build_prompt_blind_loads_signatures(tmp_path: Path) -> None:
    task_dir = tmp_path / "task"
    task_dir.mkdir()
    sig = "## API\nvoid foo();"
    (task_dir / "signatures.md").write_text(sig, encoding="utf-8")
    sys_p, user_p = build_prompt("Return 42.", "blind", task_dir)
    assert sys_p == SYSTEM_PROMPT
    assert "Return 42." in user_p
    assert "## API" in user_p
    assert "void foo()" in user_p


def test_build_prompt_human_loads_docs(tmp_path: Path) -> None:
    task_dir = tmp_path / "task"
    task_dir.mkdir()
    (task_dir / "docs.md").write_text("Long-form docs here.", encoding="utf-8")
    _, user_p = build_prompt("Task text.", "human", task_dir)
    assert "Long-form docs here." in user_p
    assert "Task text." in user_p


def test_build_prompt_aid_l1_loads_file(tmp_path: Path) -> None:
    task_dir = tmp_path / "task"
    task_dir.mkdir()
    (task_dir / "library_l1.aid").write_text("@module x\n@lang python\n", encoding="utf-8")
    _, user_p = build_prompt("Do task.", "aid_l1", task_dir)
    assert "@module x" in user_p


def test_build_prompt_aid_full_loads_file(tmp_path: Path) -> None:
    task_dir = tmp_path / "task"
    task_dir.mkdir()
    (task_dir / "library.aid").write_text("@module full\n", encoding="utf-8")
    _, user_p = build_prompt("Do task.", "aid_full", task_dir)
    assert "@module full" in user_p


def test_build_prompt_missing_context_files_still_has_task(tmp_path: Path) -> None:
    task_dir = tmp_path / "empty"
    task_dir.mkdir()
    _, user_p = build_prompt("Only task.", "blind", task_dir)
    assert "Only task." in user_p
    assert "## Reference Documentation" not in user_p
