"""Build prompts for each benchmark condition."""

from __future__ import annotations

from pathlib import Path


SYSTEM_PROMPT = """You are a Python developer. Generate code to accomplish the task described below.

IMPORTANT RULES:
- Output ONLY the Python code. No markdown, no explanations, no code fences.
- Your code must assign the final answer to a variable called `result`.
- Use only the Python standard library. No pip packages.
- The code will be executed as-is in a fresh Python process."""


def build_prompt(
    task_description: str,
    condition: str,
    task_dir: Path,
) -> tuple[str, str]:
    """Build system and user prompts for a given condition.

    Returns (system_prompt, user_prompt).
    """
    context = _load_context(condition, task_dir)

    user_parts: list[str] = []

    if context:
        user_parts.append("## Reference Documentation\n")
        user_parts.append(context)
        user_parts.append("\n")

    user_parts.append("## Task\n")
    user_parts.append(task_description)

    return SYSTEM_PROMPT, "\n".join(user_parts)


def _load_context(condition: str, task_dir: Path) -> str:
    """Load the documentation context for a condition."""
    if condition == "blind":
        path = task_dir / "signatures.md"
    elif condition == "human":
        path = task_dir / "docs.md"
    elif condition == "aid_l1":
        # Generate Layer 1 AID from the stdlib module
        path = task_dir / "library_l1.aid"
    elif condition == "aid_full":
        path = task_dir / "library.aid"
    else:
        raise ValueError(f"Unknown condition: {condition}")

    if path.exists():
        return path.read_text(encoding="utf-8")

    return ""
