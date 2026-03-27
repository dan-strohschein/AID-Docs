"""Module exercising Python edge cases for AID extraction."""

from typing import NamedTuple, NewType, TypeVar

__all__ = ["PublicFunc", "ExportedClass", "UserId", "Coordinate", "MAX_SIZE"]

T = TypeVar("T")
K = TypeVar("K", bound=str)
V = TypeVar("V", int, float)

# NewType
UserId = NewType("UserId", int)

# TypeAlias (Python 3.10+ style)
Headers = dict[str, str]

MAX_SIZE = 1024


class Coordinate(NamedTuple):
    """A 2D coordinate."""
    x: float
    y: float
    label: str = ""


class ExportedClass:
    """This class is in __all__."""

    def public_method(self) -> str:
        """A public method."""
        ...


class _PrivateClass:
    """This should be skipped (underscore prefix)."""

    def method(self) -> None:
        ...


class NotExportedClass:
    """This should be skipped (not in __all__)."""

    def method(self) -> None:
        ...


def PublicFunc(x: int) -> str:
    """This is in __all__."""
    ...


def not_exported(x: int) -> str:
    """Not in __all__, should be skipped."""
    ...


def _private_func() -> None:
    """Private, should be skipped."""
    ...
