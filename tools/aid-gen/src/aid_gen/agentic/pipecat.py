"""Mechanical (Layer 1) extraction of Pipecat pipelines.

Detects:
- `Pipeline([p1, p2, ...])` / `ParallelPipeline([...])` → @graph @engine pipecat
  with sequential downstream edges (or @composition parallel)
- `Frame` subclasses → the @frames vocabulary attached to the pipeline graph

Frame direction and per-edge frame typing require dataflow analysis of
push_frame() calls and are left for Layer 2.
"""

from __future__ import annotations

import ast

from aid_gen.agentic.langgraph import AgenticResult, _callee_name, _docstring_first_line
from aid_gen.model import GraphEntry

_PIPELINE_CLASSES = {"Pipeline", "ParallelPipeline"}
_FRAME_BASES = {"Frame", "DataFrame", "ControlFrame", "SystemFrame"}


def extract_pipecat(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    frames = _collect_frames(tree)
    for node in ast.walk(tree):
        if not (isinstance(node, ast.Assign) and len(node.targets) == 1
                and isinstance(node.targets[0], ast.Name)
                and isinstance(node.value, ast.Call)):
            continue
        cls = _callee_name(node.value.func)
        if cls not in _PIPELINE_CLASSES:
            continue
        result.graphs.append(
            _build_pipeline(node.targets[0].id, cls, node.value, frames, file_path, node.lineno)
        )


def _build_pipeline(
    name: str, cls: str, call: ast.Call, frames: list[str],
    file_path: str | None, lineno: int,
) -> GraphEntry:
    graph = GraphEntry(
        name=name,
        purpose="Pipecat pipeline",
        engine="pipecat",
        frames=frames or None,
        source_file=file_path,
        source_line=lineno,
    )
    elems: list[ast.expr] = []
    if call.args and isinstance(call.args[0], (ast.List, ast.Tuple)):
        elems = list(call.args[0].elts)

    used: set[str] = set()
    order: list[str] = []
    for el in elems:
        nm, fn = _pipe_node(el)
        base, i = nm, 2
        while nm in used:
            nm = f"{base}{i}"
            i += 1
        used.add(nm)
        order.append(nm)
        eff = _service_effects(fn)
        suffix = f" [{', '.join(eff)}]" if eff else ""
        graph.nodes.append(f"{nm}: {fn}{suffix}")

    if cls == "ParallelPipeline":
        graph.composition = "parallel"
    else:
        for a, b in zip(order, order[1:]):
            graph.edges.append(f"{a} -> {b} [downstream]")
    return graph


def _pipe_node(expr: ast.expr) -> tuple[str, str]:
    """Name a pipeline element: a processor variable or an inline service constructor."""
    if isinstance(expr, ast.Name):
        return expr.id, expr.id
    if isinstance(expr, ast.Call):
        cls = _callee_name(expr.func)
        if cls:
            return cls, cls
    if isinstance(expr, ast.Attribute):
        return expr.attr, expr.attr
    return "stage", "stage"


def _service_effects(fn: str) -> list[str]:
    low = fn.lower()
    if "llm" in low:
        return ["Llm", "Net", "Stream"]
    if "stt" in low or "transcri" in low:
        return ["Llm", "Net", "Stream"]
    if "tts" in low or "speech" in low or "speak" in low:
        return ["Net", "Stream"]
    if "transport" in low or "input" in low or "output" in low or "webrtc" in low or "daily" in low:
        return ["Net", "Stream"]
    return []


def _collect_frames(tree: ast.Module) -> list[str]:
    frames: list[str] = []
    for node in tree.body:
        if not isinstance(node, ast.ClassDef):
            continue
        base_names = {_callee_name(b) for b in node.bases}
        is_frame = bool(base_names & _FRAME_BASES) or (
            node.name.endswith("Frame") and node.name not in _FRAME_BASES
        )
        if is_frame and node.name not in _FRAME_BASES:
            doc = _docstring_first_line(node)
            if doc:
                sep = "" if doc.endswith(".") else "."
                frames.append(f"{node.name} — {doc}{sep}")
            else:
                frames.append(node.name)
    return frames
