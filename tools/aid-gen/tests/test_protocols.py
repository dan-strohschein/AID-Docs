"""Tests for protocol detection from dunder methods."""

import ast
from aid_gen.python.protocols import detect_protocols


def _parse_class(code: str) -> ast.ClassDef:
    tree = ast.parse(code)
    return tree.body[0]


def test_no_dunders():
    cls = _parse_class("class Foo:\n    def bar(self): ...")
    assert detect_protocols(cls) == []


def test_closeable():
    cls = _parse_class(
        "class Foo:\n"
        "    def __enter__(self): ...\n"
        "    def __exit__(self, *args): ...\n"
    )
    assert "Closeable" in detect_protocols(cls)


def test_closeable_requires_both():
    cls = _parse_class("class Foo:\n    def __enter__(self): ...")
    assert "Closeable" not in detect_protocols(cls)


def test_iterable():
    cls = _parse_class("class Foo:\n    def __iter__(self): ...")
    assert "Iterable" in detect_protocols(cls)


def test_iterator():
    cls = _parse_class("class Foo:\n    def __next__(self): ...")
    assert "Iterator" in detect_protocols(cls)


def test_hashable():
    cls = _parse_class("class Foo:\n    def __hash__(self): ...")
    assert "Hashable" in detect_protocols(cls)


def test_display():
    cls = _parse_class("class Foo:\n    def __str__(self): ...")
    assert "Display" in detect_protocols(cls)


def test_debug():
    cls = _parse_class("class Foo:\n    def __repr__(self): ...")
    assert "Debug" in detect_protocols(cls)


def test_callable():
    cls = _parse_class("class Foo:\n    def __call__(self): ...")
    assert "Callable" in detect_protocols(cls)


def test_eq():
    cls = _parse_class("class Foo:\n    def __eq__(self, other): ...")
    assert "Eq" in detect_protocols(cls)


def test_comparable_lt():
    cls = _parse_class("class Foo:\n    def __lt__(self, other): ...")
    assert "Comparable" in detect_protocols(cls)


def test_comparable_ge():
    cls = _parse_class("class Foo:\n    def __ge__(self, other): ...")
    assert "Comparable" in detect_protocols(cls)


def test_cloneable_copy():
    cls = _parse_class("class Foo:\n    def __copy__(self): ...")
    assert "Cloneable" in detect_protocols(cls)


def test_cloneable_deepcopy():
    cls = _parse_class("class Foo:\n    def __deepcopy__(self, memo): ...")
    assert "Cloneable" in detect_protocols(cls)


def test_multiple_protocols():
    cls = _parse_class(
        "class Foo:\n"
        "    def __iter__(self): ...\n"
        "    def __str__(self): ...\n"
        "    def __repr__(self): ...\n"
        "    def __eq__(self, other): ...\n"
        "    def __hash__(self): ...\n"
    )
    protocols = detect_protocols(cls)
    assert "Iterable" in protocols
    assert "Display" in protocols
    assert "Debug" in protocols
    assert "Eq" in protocols
    assert "Hashable" in protocols


def test_result_is_sorted():
    cls = _parse_class(
        "class Foo:\n"
        "    def __str__(self): ...\n"
        "    def __iter__(self): ...\n"
        "    def __repr__(self): ...\n"
    )
    protocols = detect_protocols(cls)
    assert protocols == sorted(protocols)
