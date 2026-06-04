"""A small LangGraph ReAct agent for extractor testing."""

from typing import Annotated, TypedDict
import operator

from langgraph.graph import StateGraph, START, END
from langgraph.checkpoint.postgres import PostgresSaver
from langchain_core.tools import tool
from langchain_anthropic import ChatAnthropic


class AgentState(TypedDict):
    """Shared state threaded through the agent loop."""
    messages: Annotated[list, operator.add]
    step_count: Annotated[int, operator.add]
    next: str


planner_llm = ChatAnthropic(model="claude-opus-4-8", temperature=0.0, max_tokens=4096)


@tool
def web_search(query: str, k: int = 5) -> list:
    """Search the web and return ranked results."""
    return []


@tool
def execute_sql(query: str) -> list:
    """Run a read-only SQL query against the warehouse."""
    return []


def planner_node(state: AgentState) -> AgentState:
    """Call the planner LLM and record its decision."""
    return state


def tool_executor(state: AgentState) -> AgentState:
    """Execute the requested tool calls."""
    return state


def should_continue(state: AgentState) -> str:
    """Route to act or finish."""
    return "act"


def build_agent():
    """Construct and compile the agent graph."""
    graph = StateGraph(AgentState)
    graph.add_node("plan", planner_node)
    graph.add_node("act", tool_executor)
    graph.add_edge(START, "plan")
    graph.add_conditional_edges("plan", should_continue, {"act": "act", "END": END})
    graph.add_edge("act", "plan")
    return graph.compile(checkpointer=PostgresSaver(), interrupt_before=["act"])
