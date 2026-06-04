"""A small LangChain LCEL chain for extractor testing."""

from langchain_core.tools import tool
from langchain_openai import ChatOpenAI


answer_llm = ChatOpenAI(model="gpt-4o", temperature=0.0)


@tool
def vector_search(query: str, k: int = 4) -> list:
    """Retrieve the top-k most relevant context chunks."""
    return []
