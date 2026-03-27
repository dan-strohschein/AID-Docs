"""Tarfile benchmark tasks — tests archive creation, extraction, and safety."""

TASKS = [
    {
        "id": "tarfile_create_archive",
        "description": (
            "Create a tar.gz archive in memory (using io.BytesIO). Add two files to it: "
            "'hello.txt' containing 'Hello World' and 'data.txt' containing '12345'. "
            "Then read the archive back and list all member names. "
            "Assign the sorted list of member names to `result`."
        ),
        "setup": "",
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert sorted(result) == ['data.txt', 'hello.txt'], f"Expected ['data.txt', 'hello.txt'], got {sorted(result)}"
""",
    },
    {
        "id": "tarfile_extract_specific",
        "description": (
            "Create a tar archive (uncompressed) in a BytesIO buffer containing three files: "
            "'a.txt' with content 'AAA', 'b.txt' with content 'BBB', 'c.txt' with content 'CCC'. "
            "Then open the archive for reading and extract ONLY the content of 'b.txt' "
            "(without extracting to disk — read it directly from the archive). "
            "Assign the content of b.txt as a string to `result`."
        ),
        "test": """
assert isinstance(result, str), f"Expected str, got {type(result)}"
assert result == 'BBB', f"Expected 'BBB', got '{result}'"
""",
    },
    {
        "id": "tarfile_add_with_arcname",
        "description": (
            "Create a temporary file on disk at a temp path containing 'secret data'. "
            "Then create a tar.gz archive (in a BytesIO buffer) and add that file "
            "BUT with a different archive name: the file should appear in the archive as "
            "'public/data.txt' regardless of its actual path on disk. "
            "Open the archive and get the list of member names. "
            "Assign the list of member names to `result`."
        ),
        "setup": """
import tempfile
_tmpfile = tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False)
_tmpfile.write('secret data')
_tmpfile.close()
_tmppath = _tmpfile.name
""",
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 1, f"Expected 1 member, got {len(result)}"
assert result[0] == 'public/data.txt', f"Expected 'public/data.txt', got '{result[0]}'"
""",
        "teardown": "os.unlink(_tmppath)",
    },
    {
        "id": "tarfile_filter_by_type",
        "description": (
            "Create a tar archive in BytesIO containing: a regular file 'readme.txt' "
            "with content 'hello', and a directory entry 'subdir/'. "
            "To add the directory entry, create a TarInfo with type set to DIRTYPE. "
            "Then read the archive and filter to get only regular files (not directories). "
            "Assign a list of names of regular files only to `result`."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert result == ['readme.txt'], f"Expected ['readme.txt'], got {result}"
""",
    },
    {
        "id": "tarfile_compression_modes",
        "description": (
            "Create three tar archives in BytesIO buffers: one uncompressed (mode 'w'), "
            "one gzip-compressed (mode 'w:gz'), and one bzip2-compressed (mode 'w:bz2'). "
            "Each archive should contain a single file 'test.txt' with content 'test data'. "
            "Get the size in bytes of each buffer. "
            "Assign a dict to `result` with keys 'tar', 'gz', 'bz2' and values being "
            "the byte sizes of each archive. The uncompressed archive should be the largest."
        ),
        "test": """
assert isinstance(result, dict), f"Expected dict, got {type(result)}"
assert 'tar' in result and 'gz' in result and 'bz2' in result, f"Missing keys: {result.keys()}"
assert all(isinstance(v, int) for v in result.values()), f"Values should be ints: {result}"
assert result['tar'] > result['gz'], f"tar ({result['tar']}) should be larger than gz ({result['gz']})"
assert result['tar'] > result['bz2'], f"tar ({result['tar']}) should be larger than bz2 ({result['bz2']})"
""",
    },
]
