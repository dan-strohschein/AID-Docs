"""A small LangChain LCEL chain for extractor testing."""

from langchain_core.tools import tool
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.runnables import RunnableParallel, RunnablePassthrough
from langchain_openai import ChatOpenAI


answer_llm = ChatOpenAI(model="gpt-4o", temperature=0.0)

rag_prompt = ChatPromptTemplate.from_messages([
    ("system", "Answer only from the provided context."),
    ("human", "Question: {question}\n\nContext: {context}"),
])


@tool
def vector_search(query: str, k: int = 4) -> list:
    """Retrieve the top-k most relevant context chunks."""
    return []


def format_docs(docs: list) -> str:
    """Render retrieved documents into a context string."""
    return ""


rag_chain = (
    RunnableParallel({"context": vector_search | format_docs, "question": RunnablePassthrough()})
    | rag_prompt
    | answer_llm
)
