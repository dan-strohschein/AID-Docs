"""Module with Protocol and ABC classes."""

from abc import ABC, abstractmethod
from typing import Protocol


class Serializable(Protocol):
    """Things that can be serialized."""

    def serialize(self) -> bytes:
        ...

    def deserialize(self, data: bytes) -> None:
        ...


class Validator(ABC):
    """Abstract base for validators."""

    @abstractmethod
    def validate(self, value: str) -> bool:
        ...

    def is_valid(self, value: str) -> bool:
        """Non-abstract convenience method."""
        return self.validate(value)
