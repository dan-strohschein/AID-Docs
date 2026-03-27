"""Module with various class types for testing."""

from dataclasses import dataclass
from enum import Enum
from typing import Generic, Optional, TypeVar

T = TypeVar("T")


@dataclass
class Point:
    """A 2D point with x and y coordinates."""
    x: float
    y: float
    label: str = ""

    def distance_to(self, other: "Point") -> float:
        """Calculate distance to another point."""
        ...

    def _private_method(self) -> None:
        ...


class Color(Enum):
    """Available colors."""
    RED = "red"
    GREEN = "green"
    BLUE = "blue"


class Connection:
    """A database connection that must be closed."""

    def __init__(self, host: str, port: int = 5432):
        ...

    def query(self, sql: str) -> list[dict]:
        """Execute a SQL query."""
        ...

    def close(self) -> None:
        """Close the connection."""
        ...

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self.close()

    def __repr__(self) -> str:
        ...


class Box(Generic[T]):
    """A generic container for a single value."""

    value: T

    def get(self) -> T:
        """Get the contained value."""
        ...

    def set(self, value: T) -> None:
        """Set the contained value."""
        ...


class Animal:
    """Base class for animals."""

    name: str

    def speak(self) -> str:
        """Make a sound."""
        ...


class Dog(Animal):
    """A dog that extends Animal."""

    breed: str

    def fetch(self, item: str) -> str:
        """Fetch an item."""
        ...
