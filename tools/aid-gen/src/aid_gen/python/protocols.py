"""Detect well-known AID protocols from Python dunder methods."""

from __future__ import annotations

import ast


# Mapping of dunder methods to AID protocol names.
# Some protocols require multiple dunders (e.g., Closeable needs both __enter__ and __exit__).
# Single-dunder protocols are listed individually.
_SINGLE_DUNDER_PROTOCOLS: dict[str, str] = {
    "__iter__": "Iterable",
    "__next__": "Iterator",
    "__hash__": "Hashable",
    "__str__": "Display",
    "__repr__": "Debug",
    "__call__": "Callable",
    "__eq__": "Eq",
}

# Ordering dunders — any one of these means Comparable
_ORDERING_DUNDERS = {"__lt__", "__le__", "__gt__", "__ge__"}

# Copy dunders — either means Cloneable
_COPY_DUNDERS = {"__copy__", "__deepcopy__"}

# Context manager requires both
_CONTEXT_MANAGER_DUNDERS = {"__enter__", "__exit__"}


def detect_protocols(class_node: ast.ClassDef) -> list[str]:
    """Detect well-known AID protocols from dunder methods on a class.

    Returns a sorted list of protocol names.
    """
    method_names = {
        node.name
        for node in ast.walk(class_node)
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
        and node.name.startswith("__")
        and node.name.endswith("__")
    }

    protocols: set[str] = set()

    # Single-dunder protocols
    for dunder, protocol in _SINGLE_DUNDER_PROTOCOLS.items():
        if dunder in method_names:
            protocols.add(protocol)

    # Ordering → Comparable
    if method_names & _ORDERING_DUNDERS:
        protocols.add("Comparable")

    # Copy → Cloneable
    if method_names & _COPY_DUNDERS:
        protocols.add("Cloneable")

    # Context manager → Closeable (requires both __enter__ and __exit__)
    if _CONTEXT_MANAGER_DUNDERS.issubset(method_names):
        protocols.add("Closeable")

    return sorted(protocols)
