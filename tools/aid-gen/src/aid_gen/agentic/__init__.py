"""Tier 4 (agentic / dataflow) extraction for AID generation.

Mechanically detects LangGraph / LangChain constructs and emits Layer 1
@graph / @tool / @model skeletons plus state-channel reducer annotations.
"""

from aid_gen.agentic.langgraph import AgenticResult, extract_agentic

__all__ = ["AgenticResult", "extract_agentic"]
