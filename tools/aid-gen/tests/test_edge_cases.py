"""Tests for Python edge cases."""

from pathlib import Path

from aid_gen.model import ConstEntry, FnEntry, TypeEntry
from aid_gen.python.parser import extract_module

FIXTURES = Path(__file__).parent / "fixtures"


def _load(name: str) -> str:
    return (FIXTURES / name).read_text()


class TestAllExports:
    """Test __all__ filtering."""

    def setup_method(self):
        self.aid = extract_module(_load("edge_cases.py"), "test/edge")

    def test_exported_function_included(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "PublicFunc" in fn_names

    def test_non_exported_function_excluded(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "not_exported" not in fn_names

    def test_private_function_excluded(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "_private_func" not in fn_names

    def test_exported_class_included(self):
        type_names = [e.name for e in self.aid.entries if isinstance(e, TypeEntry)]
        assert "ExportedClass" in type_names

    def test_non_exported_class_excluded(self):
        type_names = [e.name for e in self.aid.entries if isinstance(e, TypeEntry)]
        assert "NotExportedClass" not in type_names

    def test_private_class_excluded(self):
        type_names = [e.name for e in self.aid.entries if isinstance(e, TypeEntry)]
        assert "_PrivateClass" not in type_names

    def test_exported_constant_included(self):
        const_names = [e.name for e in self.aid.entries if isinstance(e, ConstEntry)]
        assert "MAX_SIZE" in const_names


class TestNewType:
    def setup_method(self):
        self.aid = extract_module(_load("edge_cases.py"), "test/edge")

    def test_newtype_extracted(self):
        user_id = next(
            (e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "UserId"),
            None,
        )
        assert user_id is not None

    def test_newtype_kind(self):
        user_id = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "UserId")
        assert user_id.kind == "newtype"

    def test_newtype_has_inner_field(self):
        user_id = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "UserId")
        assert user_id.fields is not None
        assert any(f.type == "int" for f in user_id.fields)


class TestNamedTuple:
    def setup_method(self):
        self.aid = extract_module(_load("edge_cases.py"), "test/edge")

    def test_namedtuple_is_struct(self):
        coord = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Coordinate")
        assert coord.kind == "struct"

    def test_namedtuple_fields(self):
        coord = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Coordinate")
        assert coord.fields is not None
        field_names = [f.name for f in coord.fields]
        assert "x" in field_names
        assert "y" in field_names
        assert "label" in field_names

    def test_namedtuple_purpose(self):
        coord = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Coordinate")
        assert coord.purpose == "A 2D coordinate."

    def test_namedtuple_has_constructor(self):
        coord = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Coordinate")
        assert coord.constructors is not None
        assert "x: float" in coord.constructors


class TestTypeAlias:
    def test_bare_alias(self):
        source = 'Headers = dict[str, str]\n'
        aid = extract_module(source, "test/alias")
        headers = next(
            (e for e in aid.entries if isinstance(e, TypeEntry) and e.name == "Headers"),
            None,
        )
        assert headers is not None
        assert headers.kind == "alias"

    def test_typed_alias(self):
        source = 'from typing import TypeAlias\nHeaders: TypeAlias = dict[str, str]\n'
        aid = extract_module(source, "test/alias")
        headers = next(
            (e for e in aid.entries if isinstance(e, TypeEntry) and e.name == "Headers"),
            None,
        )
        assert headers is not None
        assert headers.kind == "alias"


class TestTypeVarNotExtracted:
    """TypeVar assignments should not produce entries."""

    def test_typevar_skipped(self):
        aid = extract_module(_load("edge_cases.py"), "test/edge")
        entry_names = [
            e.name for e in aid.entries
            if isinstance(e, (TypeEntry, ConstEntry))
        ]
        # T, K, V are TypeVars — they should not appear as types or constants
        assert "T" not in entry_names
        assert "K" not in entry_names
        assert "V" not in entry_names


class TestNoAll:
    """When __all__ is not defined, all public names are exported."""

    def test_all_public_exported(self):
        source = (
            'def public_fn() -> None: ...\n'
            'def another() -> None: ...\n'
            'def _private() -> None: ...\n'
        )
        aid = extract_module(source, "test/noall")
        fn_names = [e.name for e in aid.entries if isinstance(e, FnEntry)]
        assert "public_fn" in fn_names
        assert "another" in fn_names
        assert "_private" not in fn_names
