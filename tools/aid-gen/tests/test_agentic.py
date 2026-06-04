"""Tests for Tier 4 (agentic / dataflow) extraction."""

from __future__ import annotations

from pathlib import Path

import pytest

from aid_gen.agentic import extract_agentic
from aid_gen.emitter import emit
from aid_gen.model import (
    AgentEntry,
    ConstEntry,
    GraphEntry,
    ModelEntry,
    PromptEntry,
    ToolEntry,
    TypeEntry,
)
from aid_gen.python.parser import extract_module

import ast

FIXTURES = Path(__file__).parent / "fixtures"


def _read(name: str) -> str:
    return (FIXTURES / name).read_text(encoding="utf-8")


def _classifies(out: str) -> bool:
    for line in out.splitlines():
        s = line.rstrip()
        if not (s == "" or s == "---" or s.startswith("//")
                or s.startswith("@") or line.startswith("  ")):
            return False
    return True


# --- engine detection -------------------------------------------------------


def test_detects_langgraph_engine():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    assert result.engine == "langgraph"


def test_detects_lcel_engine():
    result = extract_agentic(ast.parse(_read("lcel_sample.py")))
    assert result.engine == "lcel"


def test_non_agentic_module_returns_empty():
    src = "def add(a: int, b: int) -> int:\n    return a + b\n"
    result = extract_agentic(ast.parse(src))
    assert result.engine is None
    assert not result.graphs
    assert not result.tools


# --- tools ------------------------------------------------------------------


def test_extracts_tools():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    names = {t.name for t in result.tools}
    assert names == {"web_search", "execute_sql"}
    assert result.tool_fn_names == {"web_search", "execute_sql"}
    web = next(t for t in result.tools if t.name == "web_search")
    assert web.invoked_by == "llm"
    assert web.purpose == "Search the web and return ranked results."
    assert web.sigs == ["(query: str, k?: int) -> list"]


def test_tool_functions_not_emitted_as_fn():
    aid = extract_module(_read("langgraph_sample.py"), "langgraph/sample")
    fn_names = {e.name for e in aid.entries if type(e).__name__ == "FnEntry"}
    assert "web_search" not in fn_names
    assert "execute_sql" not in fn_names
    tool_names = {e.name for e in aid.entries if isinstance(e, ToolEntry)}
    assert tool_names == {"web_search", "execute_sql"}


# --- models -----------------------------------------------------------------


def test_extracts_model():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    assert len(result.models) == 1
    model = result.models[0]
    assert isinstance(model, ModelEntry)
    assert model.name == "planner_llm"
    assert model.provider == "anthropic"
    assert model.model_id == "claude-opus-4-8"
    assert "temperature=0.0" in model.params
    assert "max_tokens=4096" in model.params
    assert model.effects == ["Llm", "Net"]


def test_extracts_openai_model():
    result = extract_agentic(ast.parse(_read("lcel_sample.py")))
    model = result.models[0]
    assert model.provider == "openai"
    assert model.model_id == "gpt-4o"


# --- channels / reducers ----------------------------------------------------


def test_extracts_channels_and_reducers():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    assert "AgentState" in result.channels
    reducers = result.channels["AgentState"]
    assert reducers["messages"] == "add"
    assert reducers["step_count"] == "add"
    # `next` has no Annotated reducer, so it should not appear
    assert "next" not in reducers


def test_channels_applied_to_type_entry():
    aid = extract_module(_read("langgraph_sample.py"), "langgraph/sample")
    state = next(e for e in aid.entries if isinstance(e, TypeEntry) and e.name == "AgentState")
    assert state.channels is True
    messages = next(f for f in state.fields if f.name == "messages")
    assert messages.reducer == "add"
    # Annotated wrapper is unwrapped to the inner type
    assert messages.type == "list"


# --- graph ------------------------------------------------------------------


def test_extracts_graph_topology():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    assert len(result.graphs) == 1
    g = result.graphs[0]
    assert isinstance(g, GraphEntry)
    assert g.engine == "langgraph"
    assert g.state == "AgentState"
    assert g.entry == "plan"
    assert g.nodes == ["plan: planner_node", "act: tool_executor"]
    assert "act -> plan" in g.edges  # back-edge → cycle
    assert g.conditional_edges == ["plan: should_continue -> act | END"]
    assert g.interrupts == ["before act"]


def test_start_edge_becomes_entry_not_edge():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    g = result.graphs[0]
    assert g.entry == "plan"
    assert all("START" not in e for e in g.edges)


def test_checkpointer_detected():
    result = extract_agentic(ast.parse(_read("langgraph_sample.py")))
    assert result.checkpointer is not None
    assert "PostgresSaver" in result.checkpointer


# --- end-to-end emit --------------------------------------------------------


def test_full_emit_is_v03_and_parseable():
    aid = extract_module(_read("langgraph_sample.py"), "langgraph/sample",
                         file_path="langgraph_sample.py")
    out = emit(aid)
    assert "@engine langgraph" in out
    assert "@aid_version 0.3" in out
    assert "@channels" in out
    assert "reducer: add" in out
    assert "@graph graph" in out
    assert "@conditional_edges" in out
    assert "@interrupts" in out
    assert "@tool web_search" in out
    assert "@model planner_llm" in out
    # Every line must classify cleanly under the AID line model (no stray lines)
    for line in out.splitlines():
        s = line.rstrip()
        ok = (s == "" or s == "---" or s.startswith("//")
              or s.startswith("@") or line.startswith("  "))
        assert ok, f"unclassifiable line: {line!r}"


def test_non_agentic_emit_stays_v02():
    src = "def add(a: int, b: int) -> int:\n    '''Add two ints.'''\n    return a + b\n"
    aid = extract_module(src, "math/util")
    out = emit(aid)
    assert "@aid_version 0.2" in out
    assert "@engine" not in out
    assert "@graph" not in out


# --- LCEL composition -------------------------------------------------------


def test_lcel_engine_and_prompt():
    result = extract_agentic(ast.parse(_read("lcel_sample.py")))
    assert result.engine == "lcel"
    assert len(result.prompts) == 1
    prompt = result.prompts[0]
    assert isinstance(prompt, PromptEntry)
    assert prompt.name == "rag_prompt"
    input_names = [p.name for p in prompt.inputs]
    assert "question" in input_names and "context" in input_names


def test_lcel_chain_becomes_graph():
    result = extract_agentic(ast.parse(_read("lcel_sample.py")))
    chains = [g for g in result.graphs if g.name == "rag_chain"]
    assert len(chains) == 1
    g = chains[0]
    assert g.engine == "lcel"
    assert g.composition == "sequence"
    # Pipe order: RunnableParallel -> rag_prompt -> answer_llm
    assert g.nodes[0].startswith("parallel:")
    assert any("rag_prompt" in n for n in g.nodes)
    assert any("answer_llm" in n and "[Llm]" in n for n in g.nodes)
    assert g.edges == ["parallel -> rag_prompt", "rag_prompt -> answer_llm"]


def test_lcel_chain_not_double_emitted_as_type():
    aid = extract_module(_read("lcel_sample.py"), "lcel/sample")
    aliases = [e for e in aid.entries
               if isinstance(e, (TypeEntry, ConstEntry)) and e.name == "rag_chain"]
    assert aliases == []
    graphs = [e for e in aid.entries if isinstance(e, GraphEntry) and e.name == "rag_chain"]
    assert len(graphs) == 1


def test_lcel_emit_parseable():
    aid = extract_module(_read("lcel_sample.py"), "lcel/sample", file_path="lcel_sample.py")
    out = emit(aid)
    assert "@engine lcel" in out
    assert "@prompt rag_prompt" in out
    assert "@composition sequence" in out
    assert _classifies(out)


def test_pipe_chain_not_misread_when_not_lcel():
    # A bitwise-or on ints inside an agentic module must not become a graph.
    src = (
        "from langchain_core.tools import tool\n"
        "FLAGS = 1 | 2 | 4\n"
        "@tool\n"
        "def t(x: int) -> int:\n"
        "    '''t'''\n"
        "    return x\n"
    )
    result = extract_agentic(ast.parse(src))
    assert all(g.name != "FLAGS" for g in result.graphs)


# --- agents -----------------------------------------------------------------


def test_create_react_agent_extracted():
    src = (
        "from langgraph.prebuilt import create_react_agent\n"
        "from langchain_anthropic import ChatAnthropic\n"
        "llm = ChatAnthropic(model='claude-opus-4-8')\n"
        "researcher = create_react_agent(llm, [web_search, fetch_url])\n"
    )
    result = extract_agentic(ast.parse(src))
    agents = result.agents
    assert len(agents) == 1
    a = agents[0]
    assert isinstance(a, AgentEntry)
    assert a.name == "researcher"
    assert a.model == "llm"
    assert a.tools == ["web_search", "fetch_url"]
    assert a.autonomy == "autonomous"


def test_agent_executor_extracted():
    src = (
        "from langchain.agents import AgentExecutor\n"
        "exec_agent = AgentExecutor(agent=a, tools=[search])\n"
    )
    result = extract_agentic(ast.parse(src))
    assert result.agents[0].name == "exec_agent"
    assert result.agents[0].tools == ["search"]


# --- pipecat ----------------------------------------------------------------


def test_pipecat_engine_detected():
    result = extract_agentic(ast.parse(_read("pipecat_sample.py")))
    assert result.engine == "pipecat"


def test_pipecat_pipeline_graph():
    result = extract_agentic(ast.parse(_read("pipecat_sample.py")))
    assert len(result.graphs) == 1
    g = result.graphs[0]
    assert g.engine == "pipecat"
    assert g.name == "pipeline"
    node_names = [n.split(":")[0] for n in g.nodes]
    assert node_names == ["transport_in", "stt", "llm", "tts"]
    assert g.edges == [
        "transport_in -> stt [downstream]",
        "stt -> llm [downstream]",
        "llm -> tts [downstream]",
    ]
    # streaming service effects inferred
    assert any("[Llm, Net, Stream]" in n for n in g.nodes)


def test_pipecat_frames_collected():
    result = extract_agentic(ast.parse(_read("pipecat_sample.py")))
    frames = result.graphs[0].frames
    joined = " ".join(frames)
    assert "TranscriptionFrame" in joined
    assert "TextFrame" in joined
    # No double period
    assert ".." not in joined
    # FrameProcessor subclasses are not frames
    assert "DeepgramSTT" not in joined


def test_pipecat_emit_parseable():
    aid = extract_module(_read("pipecat_sample.py"), "pipecat/sample",
                         file_path="pipecat_sample.py")
    out = emit(aid)
    assert "@engine pipecat" in out
    assert "@frames" in out
    assert "@graph pipeline" in out
    assert "[downstream]" in out
    assert _classifies(out)
    # Frame subclasses still emitted as types
    assert "@type TranscriptionFrame" in out
