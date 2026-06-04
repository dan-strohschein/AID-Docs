"""Mechanical (Layer 1) extraction of LangGraph / LangChain constructs.

Detects:
- `StateGraph(X)` builders → @graph (nodes, edges, conditional_edges, entry, interrupts)
- `@tool`-decorated functions → @tool entries
- `ChatAnthropic(...)` / `ChatOpenAI(...)` assignments → @model entries
- State TypedDicts with `Annotated[T, reducer]` fields → @channels + reducer constraints

Only structure that is unambiguous from the AST is emitted. Semantic fields
(cycle bounds, guardrails, determinism rationale) are left for Layer 2.
"""

from __future__ import annotations

import ast
from dataclasses import dataclass, field

from aid_gen.model import AgentEntry, GraphEntry, ModelEntry, Param, PromptEntry, ToolEntry

# Known model classes → provider
_MODEL_CLASSES = {
    "ChatAnthropic": "anthropic",
    "ChatOpenAI": "openai",
    "ChatGoogleGenerativeAI": "google",
    "ChatVertexAI": "google",
    "ChatOllama": "local",
    "ChatBedrock": "aws",
}

# Prompt-template factory classes
_PROMPT_CLASSES = {
    "ChatPromptTemplate", "PromptTemplate", "FewShotPromptTemplate",
    "HumanMessagePromptTemplate", "SystemMessagePromptTemplate",
}

# Agent-builder callables → (model_arg_index, tools_arg_index)
_AGENT_BUILDERS = {
    "create_react_agent": (0, 1),
    "create_tool_calling_agent": (0, 1),
    "create_openai_tools_agent": (0, 1),
    "create_openai_functions_agent": (0, 1),
}

# Reducer call/attr names → AID reducer vocabulary
_REDUCER_NAMES = {
    "add_messages": "append",
    "add": "add",
    "operator.add": "add",
}


@dataclass
class AgenticResult:
    """Everything the agentic pass found in one module."""
    engine: str | None = None
    checkpointer: str | None = None
    tools: list[ToolEntry] = field(default_factory=list)
    models: list[ModelEntry] = field(default_factory=list)
    graphs: list[GraphEntry] = field(default_factory=list)
    agents: list[AgentEntry] = field(default_factory=list)
    prompts: list[PromptEntry] = field(default_factory=list)
    tool_fn_names: set[str] = field(default_factory=set)
    # class_name -> {field_name -> reducer}; presence means @channels
    channels: dict[str, dict[str, str]] = field(default_factory=dict)


def extract_agentic(tree: ast.Module, file_path: str | None = None) -> AgenticResult:
    """Run all Tier 4 detectors over a parsed module, dispatching by engine."""
    result = AgenticResult()
    result.engine = _detect_engine(tree)
    if result.engine is None:
        return result  # not an agentic module — nothing to add

    if result.engine == "pipecat":
        # Imported lazily to avoid a circular import at module load.
        from aid_gen.agentic.pipecat import extract_pipecat
        extract_pipecat(tree, file_path, result)
        return result

    # langgraph / lcel — the LangChain family
    _extract_tools(tree, file_path, result)
    _extract_models(tree, file_path, result)
    _extract_prompts(tree, file_path, result)
    _extract_channels(tree, result)
    _extract_graphs(tree, file_path, result)
    _extract_lcel_graphs(tree, file_path, result)
    _extract_agents(tree, file_path, result)
    return result


# --- Engine detection -------------------------------------------------------


def _detect_engine(tree: ast.Module) -> str | None:
    """Infer the dataflow engine from imports and construction calls."""
    imported: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom) and node.module:
            imported.add(node.module.split(".")[0])
        elif isinstance(node, ast.Import):
            for alias in node.names:
                imported.add(alias.name.split(".")[0])

    if "pipecat" in imported:
        return "pipecat"
    if "langgraph" in imported:
        return "langgraph"
    # StateGraph / Pipeline used without an explicit top-level import still counts
    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            callee = _callee_name(node.func)
            if callee == "StateGraph":
                return "langgraph"
            if callee == "Pipeline" and _has_frame_processor(tree):
                return "pipecat"
    if any(m.startswith("langchain") for m in imported):
        return "lcel"
    return None


def _has_frame_processor(tree: ast.Module) -> bool:
    """Heuristic: a Pipecat module defines FrameProcessor subclasses or uses Frame types."""
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            for base in node.bases:
                if _callee_name(base) in ("FrameProcessor", "Frame", "AIService"):
                    return True
        if isinstance(node, ast.Name) and node.id.endswith("Frame"):
            return True
    return False


# --- Tools ------------------------------------------------------------------


def _extract_tools(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and _has_decorator(node, "tool"):
            result.tool_fn_names.add(node.name)
            result.tools.append(_build_tool(node, file_path))


def _build_tool(node: ast.FunctionDef | ast.AsyncFunctionDef, file_path: str | None) -> ToolEntry:
    is_async = isinstance(node, ast.AsyncFunctionDef)
    sig = _simple_signature(node, is_async)
    params = _simple_params(node)
    returns = _annotation_str(node.returns) if node.returns else None
    if returns == "None":
        returns = None
    return ToolEntry(
        name=node.name,
        purpose=_docstring_first_line(node),
        sigs=[sig],
        params=params or None,
        returns=returns,
        invoked_by="llm",
        source_file=file_path,
        source_line=node.lineno,
    )


# --- Models -----------------------------------------------------------------


def _extract_models(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign) or len(node.targets) != 1:
            continue
        target = node.targets[0]
        if not isinstance(target, ast.Name):
            continue
        if not isinstance(node.value, ast.Call):
            continue
        cls = _callee_name(node.value.func)
        if cls not in _MODEL_CLASSES:
            continue
        result.models.append(_build_model(target.id, cls, node.value, file_path, node.lineno))


def _build_model(
    name: str, cls: str, call: ast.Call, file_path: str | None, lineno: int
) -> ModelEntry:
    provider = _MODEL_CLASSES[cls]
    model_id = None
    params: list[str] = []
    structured = None
    for kw in call.keywords:
        if kw.arg in ("model", "model_name") and isinstance(kw.value, ast.Constant):
            model_id = str(kw.value.value)
        elif kw.arg in ("temperature", "max_tokens", "top_p"):
            params.append(f"{kw.arg}={_literal(kw.value)}")
    return ModelEntry(
        name=name,
        purpose=f"{cls} language model",
        provider=provider,
        model_id=model_id,
        params=", ".join(params) if params else None,
        structured_output=structured,
        determinism="nondeterministic",
        effects=["Llm", "Net"],
        source_file=file_path,
        source_line=lineno,
    )


# --- Prompts ----------------------------------------------------------------


def _extract_prompts(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    for node in ast.walk(tree):
        if not (isinstance(node, ast.Assign) and len(node.targets) == 1
                and isinstance(node.targets[0], ast.Name)
                and isinstance(node.value, ast.Call)):
            continue
        cls = _class_of_call(node.value)
        if cls not in _PROMPT_CLASSES:
            continue
        result.prompts.append(PromptEntry(
            name=node.targets[0].id,
            purpose=f"{cls} prompt template",
            inputs=_prompt_inputs(node.value) or None,
            source_file=file_path,
            source_line=node.lineno,
        ))


def _prompt_inputs(call: ast.Call) -> list[Param]:
    """Collect template variables: {name} placeholders and any input_variables=[...]."""
    names: list[str] = []

    def add(n: str) -> None:
        if n and n.isidentifier() and n not in names:
            names.append(n)

    # explicit input_variables=[...]
    for kw in call.keywords:
        if kw.arg == "input_variables":
            for n in _str_list(kw.value):
                add(n)

    # {placeholder} occurrences in every string literal under the call
    import re as _re
    for sub in ast.walk(call):
        if isinstance(sub, ast.Constant) and isinstance(sub.value, str):
            text = sub.value.replace("{{", "").replace("}}", "")
            for m in _re.findall(r"\{(\w+)\}", text):
                add(m)

    return [Param(name=n, type="str") for n in names]


# --- Agents -----------------------------------------------------------------


def _extract_agents(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    for node in ast.walk(tree):
        if not (isinstance(node, ast.Assign) and len(node.targets) == 1
                and isinstance(node.targets[0], ast.Name)
                and isinstance(node.value, ast.Call)):
            continue
        call = node.value
        callee = _callee_name(call.func)
        name = node.targets[0].id
        if callee in _AGENT_BUILDERS:
            mi, ti = _AGENT_BUILDERS[callee]
            model = _callee_name(_arg(call, mi, "model", "llm"))
            tools = _name_list(_arg(call, ti, "tools"))
            result.agents.append(AgentEntry(
                name=name,
                purpose=f"{callee} agent",
                model=model,
                tools=tools,
                autonomy="autonomous",
                effects=["Llm", "Tool"],
                source_file=file_path,
                source_line=node.lineno,
            ))
        elif callee == "AgentExecutor":
            tools = _name_list(_arg(call, None, "tools"))
            result.agents.append(AgentEntry(
                name=name,
                purpose="AgentExecutor agent",
                tools=tools,
                autonomy="autonomous",
                effects=["Llm", "Tool"],
                source_file=file_path,
                source_line=node.lineno,
            ))


# --- LCEL composition graphs ------------------------------------------------


def _extract_lcel_graphs(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    known = ({m.name for m in result.models}
             | {t.name for t in result.tools}
             | {p.name for p in result.prompts})
    model_names = {m.name for m in result.models}
    tool_names = {t.name for t in result.tools}

    for node in ast.walk(tree):
        if not (isinstance(node, ast.Assign) and len(node.targets) == 1
                and isinstance(node.targets[0], ast.Name)):
            continue
        stages = _flatten_pipe(node.value)
        if len(stages) < 2:
            continue
        if not _looks_like_lcel(stages, known):
            continue  # avoid misreading int/set `|` as a chain

        graph = GraphEntry(
            name=node.targets[0].id,
            purpose="LCEL runnable composition",
            engine="lcel",
            composition="sequence",
            source_file=file_path,
            source_line=node.lineno,
        )
        used: set[str] = set()
        order: list[str] = []
        for st in stages:
            nm, fn = _lcel_stage(st)
            base, i = nm, 2
            while nm in used:
                nm = f"{base}{i}"
                i += 1
            used.add(nm)
            order.append(nm)
            eff = _lcel_effects(fn, model_names, tool_names)
            suffix = f" [{', '.join(eff)}]" if eff else ""
            graph.nodes.append(f"{nm}: {fn}{suffix}")
        for a, b in zip(order, order[1:]):
            graph.edges.append(f"{a} -> {b}")
        result.graphs.append(graph)


def _flatten_pipe(node: ast.expr) -> list[ast.expr]:
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.BitOr):
        return _flatten_pipe(node.left) + _flatten_pipe(node.right)
    return [node]


def _looks_like_lcel(stages: list[ast.expr], known: set[str]) -> bool:
    for st in stages:
        root = (_lcel_stage(st)[1] or "").split(".")[0]
        if root in known:
            return True
        if root.startswith("Runnable") or _class_of_call_expr(st) and _class_of_call_expr(st).startswith("Runnable"):
            return True
        low = root.lower()
        if any(k in low for k in ("retriev", "embed", "vector", "prompt", "llm", "model", "parser", "chain")):
            return True
    return False


def _lcel_stage(expr: ast.expr) -> tuple[str, str]:
    """Return (node_name, fn_label) for one stage of a pipe chain."""
    if isinstance(expr, ast.Name):
        return expr.id, expr.id
    if isinstance(expr, ast.Attribute):
        root = expr.value.id if isinstance(expr.value, ast.Name) else _callee_name(expr.value) or "stage"
        return root, f"{root}.{expr.attr}"
    if isinstance(expr, ast.Call):
        cls = _class_of_call_expr(expr)
        if isinstance(expr.func, ast.Attribute):
            root = (expr.func.value.id if isinstance(expr.func.value, ast.Name)
                    else _callee_name(expr.func.value) or "stage")
            return root, f"{root}.{expr.func.attr}"
        if cls == "RunnableParallel":
            return "parallel", "RunnableParallel"
        if cls == "RunnablePassthrough":
            return "passthrough", "RunnablePassthrough"
        callee = _callee_name(expr.func) or "stage"
        return callee, callee
    if isinstance(expr, ast.Dict):
        return "parallel", "RunnableParallel"
    if isinstance(expr, ast.Lambda):
        return "lambda", "lambda"
    return "stage", _safe_unparse(expr)


def _lcel_effects(fn: str, model_names: set[str], tool_names: set[str]) -> list[str]:
    root = fn.split(".")[0]
    low = root.lower()
    if root in model_names or "llm" in low or "model" in low or "chat" in low:
        return ["Llm"]
    if root in tool_names:
        return ["Tool"]
    if any(k in low for k in ("retriev", "embed", "vector")):
        return ["Embed"]
    return []


# --- State channels / reducers ----------------------------------------------


def _extract_channels(tree: ast.Module, result: AgenticResult) -> None:
    """Find TypedDict/annotated state classes whose fields use Annotated[T, reducer]."""
    for node in tree.body:
        if not isinstance(node, ast.ClassDef):
            continue
        reducers: dict[str, str] = {}
        has_annotated = False
        for item in node.body:
            if isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
                reducer = _reducer_from_annotation(item.annotation)
                if reducer is not None:
                    has_annotated = True
                    reducers[item.target.id] = reducer
        if has_annotated:
            result.channels[node.name] = reducers


def _reducer_from_annotation(ann: ast.expr) -> str | None:
    """If the annotation is Annotated[T, reducer], return the AID reducer name."""
    if not (isinstance(ann, ast.Subscript) and _callee_name(ann.value) == "Annotated"):
        return None
    args = ann.slice.elts if isinstance(ann.slice, ast.Tuple) else [ann.slice]
    if len(args) < 2:
        return "last-wins"
    meta = args[1]
    meta_name = _callee_name(meta) if not isinstance(meta, ast.Call) else _callee_name(meta.func)
    if meta_name in _REDUCER_NAMES:
        return _REDUCER_NAMES[meta_name]
    # operator.add written as an attribute
    dotted = _dotted_name(meta)
    if dotted in _REDUCER_NAMES:
        return _REDUCER_NAMES[dotted]
    return f"custom:{meta_name}" if meta_name else "last-wins"


# --- Graphs -----------------------------------------------------------------


def _extract_graphs(tree: ast.Module, file_path: str | None, result: AgenticResult) -> None:
    # Map graph-builder variable name -> (state_type, lineno)
    builders: dict[str, tuple[str | None, int]] = {}
    for node in ast.walk(tree):
        if (isinstance(node, ast.Assign) and len(node.targets) == 1
                and isinstance(node.targets[0], ast.Name)
                and isinstance(node.value, ast.Call)
                and _callee_name(node.value.func) == "StateGraph"):
            state = None
            if node.value.args:
                state = _callee_name(node.value.args[0]) or _annotation_str(node.value.args[0])
            builders[node.targets[0].id] = (state, node.lineno)

    if not builders:
        return

    graphs: dict[str, GraphEntry] = {
        var: GraphEntry(
            name=f"{var}",
            purpose=None,
            engine="langgraph",
            state=state,
            source_file=file_path,
            source_line=lineno,
        )
        for var, (state, lineno) in builders.items()
    }

    # Walk every method call on a builder variable
    for node in ast.walk(tree):
        if not (isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                and isinstance(node.func.value, ast.Name)):
            continue
        var = node.func.value.id
        if var not in graphs:
            continue
        _apply_builder_call(graphs[var], node.func.attr, node, result)

    result.graphs.extend(graphs.values())


def _apply_builder_call(
    graph: GraphEntry, method: str, call: ast.Call, result: AgenticResult
) -> None:
    if method == "add_node" and len(call.args) >= 2:
        name = _literal(call.args[0])
        fn = _callee_name(call.args[1]) or "?"
        graph.nodes.append(f"{name}: {fn}")
    elif method == "add_edge" and len(call.args) >= 2:
        src = _node_ref(call.args[0])
        dst = _node_ref(call.args[1])
        if src == "START":
            graph.entry = dst
        else:
            graph.edges.append(f"{src} -> {dst}")
    elif method == "set_entry_point" and call.args:
        graph.entry = _literal(call.args[0])
    elif method == "add_conditional_edges" and len(call.args) >= 2:
        src = _node_ref(call.args[0])
        router = _callee_name(call.args[1]) or "?"
        targets = _conditional_targets(call.args[2]) if len(call.args) >= 3 else []
        tgt = " | ".join(targets) if targets else "?"
        graph.conditional_edges.append(f"{src}: {router} -> {tgt}")
    elif method == "compile":
        _apply_compile(graph, call, result)


def _apply_compile(graph: GraphEntry, call: ast.Call, result: AgenticResult) -> None:
    interrupts: list[str] = []
    for kw in call.keywords:
        if kw.arg == "interrupt_before":
            for n in _str_list(kw.value):
                interrupts.append(f"before {n}")
        elif kw.arg == "interrupt_after":
            for n in _str_list(kw.value):
                interrupts.append(f"after {n}")
        elif kw.arg == "checkpointer" and result.checkpointer is None:
            backend = _callee_name(kw.value) or _dotted_name(kw.value) or "checkpointer"
            result.checkpointer = f"checkpointer — {backend}"
    if interrupts:
        graph.interrupts = (graph.interrupts or []) + interrupts


# --- Small AST helpers ------------------------------------------------------


def _class_of_call(call: ast.Call) -> str | None:
    """The class/factory a call constructs: ClassName(...) or ClassName.factory(...)."""
    func = call.func
    if isinstance(func, ast.Attribute) and isinstance(func.value, ast.Name):
        return func.value.id  # ClassName.from_x(...)
    if isinstance(func, ast.Name):
        return func.id  # ClassName(...)
    return None


def _class_of_call_expr(expr: ast.expr) -> str | None:
    return _class_of_call(expr) if isinstance(expr, ast.Call) else None


def _arg(call: ast.Call, index: int | None, *kw_names: str) -> ast.expr | None:
    """Positional arg by index, falling back to the named keyword args."""
    if index is not None and index < len(call.args):
        return call.args[index]
    for kw in call.keywords:
        if kw.arg in kw_names:
            return kw.value
    return None


def _name_list(node: ast.expr | None) -> list[str] | None:
    """A list/tuple of Names → their ids; a single Name → [id]; else None."""
    if node is None:
        return None
    if isinstance(node, (ast.List, ast.Tuple)):
        out = [_callee_name(e) for e in node.elts]
        out = [n for n in out if n]
        return out or None
    name = _callee_name(node)
    return [name] if name else None


def _safe_unparse(node: ast.expr) -> str:
    try:
        return ast.unparse(node)
    except Exception:
        return "?"


def _node_ref(node: ast.expr) -> str:
    """Resolve an edge endpoint: a string literal, or START/END name."""
    if isinstance(node, ast.Constant) and isinstance(node.value, str):
        return node.value
    if isinstance(node, ast.Name) and node.id in ("START", "END"):
        return node.id
    return _literal(node)


def _conditional_targets(node: ast.expr) -> list[str]:
    """Targets from a conditional-edges mapping dict or list."""
    targets: list[str] = []
    if isinstance(node, ast.Dict):
        for v in node.values:
            targets.append(_node_ref(v))
    elif isinstance(node, (ast.List, ast.Tuple)):
        for elt in node.elts:
            targets.append(_node_ref(elt))
    # de-dup, preserve order
    seen: set[str] = set()
    out: list[str] = []
    for t in targets:
        if t not in seen:
            seen.add(t)
            out.append(t)
    return out


def _str_list(node: ast.expr) -> list[str]:
    if isinstance(node, (ast.List, ast.Tuple)):
        return [e.value for e in node.elts if isinstance(e, ast.Constant) and isinstance(e.value, str)]
    if isinstance(node, ast.Constant) and isinstance(node.value, str):
        return [node.value]
    return []


def _literal(node: ast.expr) -> str:
    if isinstance(node, ast.Constant):
        return str(node.value)
    name = _callee_name(node)
    if name:
        return name
    try:
        return ast.unparse(node)
    except Exception:
        return "?"


def _callee_name(node: ast.expr) -> str | None:
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        return node.attr
    if isinstance(node, ast.Call):
        return _callee_name(node.func)
    return None


def _dotted_name(node: ast.expr) -> str | None:
    if isinstance(node, ast.Attribute) and isinstance(node.value, ast.Name):
        return f"{node.value.id}.{node.attr}"
    return None


def _annotation_str(node: ast.expr | None) -> str | None:
    if node is None:
        return None
    try:
        return ast.unparse(node)
    except Exception:
        return None


def _simple_signature(node: ast.FunctionDef | ast.AsyncFunctionDef, is_async: bool) -> str:
    parts: list[str] = []
    args = node.args
    num_defaults = len(args.defaults)
    offset = len(args.args) - num_defaults
    for i, arg in enumerate(args.args):
        ty = _annotation_str(arg.annotation) or "any"
        opt = "?" if i >= offset else ""
        parts.append(f"{arg.arg}{opt}: {ty}")
    ret = _annotation_str(node.returns) or "None"
    prefix = "async " if is_async else ""
    return f"{prefix}({', '.join(parts)}) -> {ret}"


def _simple_params(node: ast.FunctionDef | ast.AsyncFunctionDef) -> list[Param]:
    params: list[Param] = []
    args = node.args
    num_defaults = len(args.defaults)
    offset = len(args.args) - num_defaults
    for i, arg in enumerate(args.args):
        default = None
        if i >= offset:
            try:
                default = ast.unparse(args.defaults[i - offset])
            except Exception:
                default = None
        params.append(Param(name=arg.arg, type=_annotation_str(arg.annotation), default=default))
    return params


def _has_decorator(node: ast.FunctionDef | ast.AsyncFunctionDef, name: str) -> bool:
    for dec in node.decorator_list:
        if isinstance(dec, ast.Name) and dec.id == name:
            return True
        if isinstance(dec, ast.Attribute) and dec.attr == name:
            return True
        if isinstance(dec, ast.Call):
            if isinstance(dec.func, ast.Name) and dec.func.id == name:
                return True
            if isinstance(dec.func, ast.Attribute) and dec.func.attr == name:
                return True
    return False


def _docstring_first_line(node: ast.AST) -> str | None:
    body = getattr(node, "body", None)
    if (body and isinstance(body[0], ast.Expr)
            and isinstance(body[0].value, ast.Constant)
            and isinstance(body[0].value.value, str)):
        first = body[0].value.value.strip().split("\n")[0].strip()
        return first[:117] + "..." if len(first) > 120 else first
    return None
