"""Tests for the Python AST parser."""

import os
from pathlib import Path

from aid_gen.model import ConstEntry, FnEntry, TraitEntry, TypeEntry
from aid_gen.python.parser import extract_module

FIXTURES = Path(__file__).parent / "fixtures"


def _load(name: str) -> str:
    return (FIXTURES / name).read_text()


# --- Simple module tests ---

class TestSimpleModule:
    def setup_method(self):
        self.aid = extract_module(_load("simple_module.py"), "test/simple")

    def test_header(self):
        assert self.aid.header.module == "test/simple"
        assert self.aid.header.lang == "python"
        assert self.aid.header.purpose == "A simple HTTP utility module for testing."

    def test_public_functions_extracted(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "get" in fn_names
        assert "post" in fn_names
        assert "fetch_all" in fn_names

    def test_private_functions_skipped(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "_internal_helper" not in fn_names

    def test_function_sig(self):
        get_fn = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "get")
        assert len(get_fn.sigs) == 1
        assert get_fn.sigs[0] == "(url: str, timeout?: float) -> dict"

    def test_function_purpose(self):
        get_fn = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "get")
        assert get_fn.purpose == "Fetch a resource from the given URL."

    def test_async_function(self):
        fetch = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "fetch_all")
        assert fetch.sigs[0].startswith("async ")

    def test_function_params(self):
        post_fn = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "post")
        assert post_fn.params is not None
        names = [p.name for p in post_fn.params]
        assert "url" in names
        assert "data" in names
        assert "timeout" in names
        timeout = next(p for p in post_fn.params if p.name == "timeout")
        assert timeout.default == "30.0"

    def test_constants_extracted(self):
        consts = [e for e in self.aid.entries if isinstance(e, ConstEntry)]
        const_names = [c.name for c in consts]
        assert "MAX_RETRIES" in const_names
        assert "DEFAULT_TIMEOUT" in const_names
        assert "BASE_URL" in const_names

    def test_constant_types(self):
        consts = {e.name: e for e in self.aid.entries if isinstance(e, ConstEntry)}
        assert consts["MAX_RETRIES"].type == "int"
        assert consts["MAX_RETRIES"].value == "3"
        assert consts["DEFAULT_TIMEOUT"].type == "float"
        assert consts["BASE_URL"].type == "str"


# --- Class tests ---

class TestClasses:
    def setup_method(self):
        self.aid = extract_module(_load("classes.py"), "test/classes")

    def test_dataclass_is_struct(self):
        point = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Point")
        assert point.kind == "struct"

    def test_dataclass_fields(self):
        point = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Point")
        assert point.fields is not None
        field_names = [f.name for f in point.fields]
        assert "x" in field_names
        assert "y" in field_names
        assert "label" in field_names
        label = next(f for f in point.fields if f.name == "label")
        assert label.default == "''"

    def test_dataclass_purpose(self):
        point = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Point")
        assert point.purpose == "A 2D point with x and y coordinates."

    def test_dataclass_methods(self):
        point = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Point")
        assert point.methods is not None
        assert "distance_to" in point.methods
        assert "_private_method" not in point.methods

    def test_method_dot_notation(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "Point.distance_to" in fn_names

    def test_method_sig_has_self(self):
        dist = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "Point.distance_to")
        assert "self" in dist.sigs[0]
        # self should not be in params
        if dist.params:
            param_names = [p.name for p in dist.params]
            assert "self" not in param_names

    def test_enum(self):
        color = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Color")
        assert color.kind == "enum"
        assert color.variants is not None
        variant_names = [v.name for v in color.variants]
        assert "RED" in variant_names
        assert "GREEN" in variant_names
        assert "BLUE" in variant_names

    def test_class_with_init(self):
        conn = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Connection")
        assert conn.kind == "class"
        assert conn.constructors is not None
        assert "host: str" in conn.constructors

    def test_closeable_protocol(self):
        conn = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Connection")
        assert conn.implements is not None
        assert "Closeable" in conn.implements

    def test_debug_protocol(self):
        conn = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Connection")
        assert "Debug" in conn.implements

    def test_generic_class(self):
        box = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Box")
        assert box.generic_params is not None
        assert "T" in box.generic_params

    def test_inheritance(self):
        dog = next(e for e in self.aid.entries if isinstance(e, TypeEntry) and e.name == "Dog")
        assert dog.extends is not None
        assert "Animal" in dog.extends

    def test_private_methods_skipped(self):
        fn_names = [e.name for e in self.aid.entries if isinstance(e, FnEntry)]
        assert "Point._private_method" not in fn_names


# --- Protocol/ABC tests ---

class TestProtocols:
    def setup_method(self):
        self.aid = extract_module(_load("protocols_mod.py"), "test/protocols")

    def test_protocol_is_trait(self):
        ser = next(e for e in self.aid.entries if isinstance(e, TraitEntry) and e.name == "Serializable")
        assert isinstance(ser, TraitEntry)

    def test_protocol_requires(self):
        ser = next(e for e in self.aid.entries if isinstance(e, TraitEntry) and e.name == "Serializable")
        assert len(ser.requires) == 2
        req_strs = " ".join(ser.requires)
        assert "serialize" in req_strs
        assert "deserialize" in req_strs

    def test_abc_is_trait(self):
        val = next(e for e in self.aid.entries if isinstance(e, TraitEntry) and e.name == "Validator")
        assert isinstance(val, TraitEntry)

    def test_abc_abstract_vs_provided(self):
        val = next(e for e in self.aid.entries if isinstance(e, TraitEntry) and e.name == "Validator")
        assert len(val.requires) >= 1
        req_strs = " ".join(val.requires)
        assert "validate" in req_strs
        # is_valid is not abstract, so it should be in provided
        assert val.provided is not None
        prov_strs = " ".join(val.provided)
        assert "is_valid" in prov_strs


# --- Async module tests ---

class TestAsync:
    def setup_method(self):
        self.aid = extract_module(_load("async_module.py"), "test/async")

    def test_async_sig(self):
        fetch = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "fetch")
        assert fetch.sigs[0] == "async (url: str) -> bytes"

    def test_async_with_defaults(self):
        send = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "send")
        assert "retries?: int" in send.sigs[0]
        assert send.sigs[0].startswith("async ")

    def test_sync_not_marked_async(self):
        helper = next(e for e in self.aid.entries if isinstance(e, FnEntry) and e.name == "sync_helper")
        assert not helper.sigs[0].startswith("async ")


# --- Full pipeline test ---

class TestFullPipeline:
    """Test that extract → emit produces valid .aid output."""

    def test_emit_simple_module(self):
        from aid_gen.emitter import emit
        aid = extract_module(_load("simple_module.py"), "test/simple", version="1.0.0")
        output = emit(aid)

        # Should have all the key markers
        assert "@module test/simple" in output
        assert "@lang python" in output
        assert "@version 1.0.0" in output
        assert "@fn get" in output
        assert "@fn post" in output
        assert "@fn fetch_all" in output
        assert "@const MAX_RETRIES" in output
        assert "---" in output

    def test_emit_classes(self):
        from aid_gen.emitter import emit
        aid = extract_module(_load("classes.py"), "test/classes")
        output = emit(aid)

        assert "@type Point" in output
        assert "@kind struct" in output
        assert "@type Color" in output
        assert "@kind enum" in output
        assert "@fn Point.distance_to" in output
        assert "@implements [Closeable, Debug]" in output
