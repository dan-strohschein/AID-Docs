"""Tests for the AID emitter."""

from aid_gen.emitter import emit
from aid_gen.model import (
    AidFile,
    ConstEntry,
    Field_,
    FnEntry,
    ModuleHeader,
    Param,
    PlatformNote,
    TraitEntry,
    TypeEntry,
    Variant,
    Workflow,
)


def test_emit_minimal_header():
    aid = AidFile(header=ModuleHeader(module="test/module"))
    output = emit(aid)
    assert "@module test/module" in output
    assert "@lang python" in output
    assert "@version 0.0.0" in output
    assert "@aid_version 0.1" in output


def test_emit_full_header():
    aid = AidFile(header=ModuleHeader(
        module="http/client",
        lang="python",
        version="2.31.0",
        stability="stable",
        purpose="HTTP client library for making web requests",
        deps=["ssl", "dns"],
        source="https://github.com/psf/requests",
        aid_version="0.1",
    ))
    output = emit(aid)
    assert "@module http/client" in output
    assert "@stability stable" in output
    assert "@purpose HTTP client library for making web requests" in output
    assert "@deps [ssl, dns]" in output
    assert "@source https://github.com/psf/requests" in output


def test_emit_fn_entry():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="get",
                purpose="Perform an HTTP GET request",
                sigs=["(url: str, timeout?: float) -> Response ! HttpError"],
                params=[
                    Param(name="url", type="str", description="Full URL"),
                    Param(name="timeout", type="float", default="30s"),
                ],
                returns="Response with status and body",
                errors=["HttpError.DnsFailure — domain not found"],
                effects=["Net"],
                related=["post", "Response"],
            ),
        ],
    )
    output = emit(aid)
    assert "@fn get" in output
    assert "@purpose Perform an HTTP GET request" in output
    assert "@sig (url: str, timeout?: float) -> Response ! HttpError" in output
    assert "@params" in output
    assert "  url: str — Full URL" in output
    assert "  timeout: float — Default 30s." in output
    assert "@returns Response with status and body" in output
    assert "@errors" in output
    assert "  HttpError.DnsFailure — domain not found" in output
    assert "@effects [Net]" in output
    assert "@related post, Response" in output


def test_emit_fn_with_subparams():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="request",
                sigs=["(url: str, opts?: RequestOpts) -> Response"],
                params=[
                    Param(
                        name="opts",
                        type="RequestOpts",
                        description="Request configuration",
                        sub_params=[
                            Param(name="timeout", type="Duration", default="30s"),
                            Param(name="redirects", type="int", default="5"),
                        ],
                    ),
                ],
            ),
        ],
    )
    output = emit(aid)
    assert "  opts: RequestOpts — Request configuration" in output
    assert "    .timeout: Duration — Default 30s." in output
    assert "    .redirects: int — Default 5." in output


def test_emit_fn_variadic():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="log",
                sigs=["(...messages: str) -> None"],
                params=[
                    Param(name="messages", type="str", is_variadic=True),
                ],
            ),
        ],
    )
    output = emit(aid)
    assert "  ...messages: str" in output


def test_emit_multiple_sigs():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="parse",
                sigs=[
                    "(input: str) -> Value ! ParseError",
                    "(input: bytes, encoding?: str) -> Value ! ParseError",
                ],
            ),
        ],
    )
    output = emit(aid)
    assert "@sig (input: str) -> Value ! ParseError" in output
    assert "@sig (input: bytes, encoding?: str) -> Value ! ParseError" in output


def test_emit_type_struct():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            TypeEntry(
                name="Response",
                kind="struct",
                purpose="HTTP response",
                fields=[
                    Field_(name="status", type="int", description="HTTP status code"),
                    Field_(name="headers", type="Headers", description="Response headers"),
                ],
                constructors="None — produced by get(), post()",
                methods=["json", "text", "close"],
                implements=["Closeable", "Debug"],
                related=["Headers", "get"],
            ),
        ],
    )
    output = emit(aid)
    assert "@type Response" in output
    assert "@kind struct" in output
    assert "@purpose HTTP response" in output
    assert "@fields" in output
    assert "  status: int — HTTP status code" in output
    assert "  headers: Headers — Response headers" in output
    assert "@constructors None — produced by get(), post()" in output
    assert "@methods json, text, close" in output
    assert "@implements [Closeable, Debug]" in output
    assert "@related Headers, get" in output


def test_emit_type_enum():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            TypeEntry(
                name="HttpError",
                kind="enum",
                purpose="Errors during HTTP operations",
                extends=["Exception"],
                variants=[
                    Variant(name="DnsFailure", payload="domain: str", description="could not resolve"),
                    Variant(name="Timeout", description="request timed out"),
                ],
                implements=["Error", "Display"],
            ),
        ],
    )
    output = emit(aid)
    assert "@type HttpError" in output
    assert "@kind enum" in output
    assert "@extends Exception" in output
    assert "@variants" in output
    assert "  | DnsFailure(domain: str) — could not resolve" in output
    assert "  | Timeout — request timed out" in output
    assert "@implements [Error, Display]" in output


def test_emit_type_with_generics():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            TypeEntry(
                name="HashMap",
                kind="struct",
                purpose="Key-value hash map",
                generic_params="K: Hash + Eq, V",
            ),
        ],
    )
    output = emit(aid)
    assert "@generic_params K: Hash + Eq, V" in output


def test_emit_trait():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            TraitEntry(
                name="Serializable",
                purpose="Can convert to/from wire format",
                requires=[
                    "fn serialize(self) -> bytes ! SerializeError",
                    "fn deserialize(data: bytes) -> Self ! DeserializeError",
                ],
                provided=[
                    "fn to_json(self) -> str ! SerializeError",
                ],
                implementors=["str", "int", "float"],
                related=["SerializeError"],
            ),
        ],
    )
    output = emit(aid)
    assert "@trait Serializable" in output
    assert "@purpose Can convert to/from wire format" in output
    assert "@requires" in output
    assert "  fn serialize(self) -> bytes ! SerializeError" in output
    assert "@provided" in output
    assert "  fn to_json(self) -> str ! SerializeError" in output
    assert "@implementors [str, int, float]" in output
    assert "@related SerializeError" in output


def test_emit_const():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            ConstEntry(
                name="MAX_REDIRECTS",
                purpose="Maximum number of HTTP redirects",
                type="int",
                value="30",
            ),
        ],
    )
    output = emit(aid)
    assert "@const MAX_REDIRECTS" in output
    assert "@purpose Maximum number of HTTP redirects" in output
    assert "@type int" in output
    assert "@value 30" in output


def test_emit_workflow():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        workflows=[
            Workflow(
                name="basic_usage",
                purpose="Basic request lifecycle",
                steps=[
                    "1. Create: Config{} — set options",
                    "2. Execute: get(url) — send request",
                ],
                antipatterns=[
                    "Don't skip close().",
                ],
            ),
        ],
    )
    output = emit(aid)
    assert "@workflow basic_usage" in output
    assert "@purpose Basic request lifecycle" in output
    assert "@steps" in output
    assert "  1. Create: Config{} — set options" in output
    assert "  2. Execute: get(url) — send request" in output
    assert "@antipatterns" in output
    assert "  - Don't skip close()." in output


def test_emit_platform():
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="get_path",
                sigs=["() -> str"],
                platform=[
                    PlatformNote(platform="windows", note="Uses backslash separators."),
                    PlatformNote(platform="linux", note="Uses forward slash."),
                ],
            ),
        ],
    )
    output = emit(aid)
    assert "@platform" in output
    assert "  windows: Uses backslash separators." in output
    assert "  linux: Uses forward slash." in output


def test_emit_separators():
    """Entries are separated by --- lines."""
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(name="foo", sigs=["() -> None"]),
            FnEntry(name="bar", sigs=["() -> None"]),
        ],
    )
    output = emit(aid)
    assert output.count("---") == 2  # one between header and foo, one between foo and bar


def test_emit_none_fields_omitted():
    """Fields set to None should not appear in output."""
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(name="simple", sigs=["() -> None"]),
        ],
    )
    output = emit(aid)
    assert "@errors" not in output
    assert "@pre" not in output
    assert "@post" not in output
    assert "@effects" not in output
    assert "@thread_safety" not in output
    assert "@complexity" not in output
    assert "@since" not in output
    assert "@deprecated" not in output
    assert "@related" not in output
    assert "@platform" not in output
    assert "@example" not in output


def test_emit_fn_all_fields():
    """A function entry with every field populated."""
    aid = AidFile(
        header=ModuleHeader(module="test/mod"),
        entries=[
            FnEntry(
                name="Response.json",
                purpose="Parse response body as JSON",
                sigs=["[T](mut self) -> T ! ParseError"],
                params=None,
                returns="Parsed value of type T",
                errors=[
                    "ParseError.InvalidJson — body is not valid JSON",
                    "ParseError.TypeMismatch — doesn't match type T",
                ],
                pre="self.body is open",
                post="self.body is closed",
                effects=["Callback"],
                thread_safety="Not safe.",
                complexity="O(n) where n = body size",
                since="1.0.0",
                deprecated="Use parse() instead",
                related=["Response.text", "Response.bytes"],
                example='data := resp.json[MyData]()?',
            ),
        ],
    )
    output = emit(aid)
    assert "@fn Response.json" in output
    assert "@sig [T](mut self) -> T ! ParseError" in output
    assert "@pre self.body is open" in output
    assert "@post self.body is closed" in output
    assert "@effects [Callback]" in output
    assert "@thread_safety Not safe." in output
    assert "@complexity O(n) where n = body size" in output
    assert "@since 1.0.0" in output
    assert "@deprecated Use parse() instead" in output
    assert "@related Response.text, Response.bytes" in output
    assert "@example" in output
    assert "  data := resp.json[MyData]()?" in output
