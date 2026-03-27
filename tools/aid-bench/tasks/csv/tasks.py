"""CSV benchmark tasks — tests reading, writing, DictReader, and edge cases."""

TASKS = [
    {
        "id": "csv_read_basic",
        "description": (
            "Given a CSV string with headers 'name,age,city' and rows: "
            "'Alice,30,NYC', 'Bob,25,LA', 'Charlie,35,Chicago', "
            "parse it using the csv module and assign a list of dicts "
            "(one per row, with keys from the header) to `result`."
        ),
        "setup": """
_csv_data = "name,age,city\\nAlice,30,NYC\\nBob,25,LA\\nCharlie,35,Chicago"
""",
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 3, f"Expected 3 rows, got {len(result)}"
assert result[0]['name'] == 'Alice', f"Expected Alice, got {result[0].get('name')}"
assert result[1]['age'] == '25', f"Expected '25', got {result[1].get('age')}"
assert result[2]['city'] == 'Chicago', f"Expected Chicago, got {result[2].get('city')}"
""",
    },
    {
        "id": "csv_write_file",
        "description": (
            "Write a CSV file to a temporary path. The file should have headers "
            "'product,price,quantity' and these rows: "
            "('Widget', '9.99', '100'), ('Gadget', '24.99', '50'), ('Doohickey', '4.99', '200'). "
            "Then read the file back and assign the entire file content as a string to `result`. "
            "IMPORTANT: When opening files for CSV writing in Python, you must use newline='' "
            "to prevent double newlines on Windows."
        ),
        "setup": """
import tempfile
_tmpfile = tempfile.NamedTemporaryFile(mode='w', suffix='.csv', delete=False, newline='')
_tmppath = _tmpfile.name
_tmpfile.close()
""",
        "test": """
assert isinstance(result, str), f"Expected str, got {type(result)}"
assert 'product,price,quantity' in result, f"Missing header in: {result[:50]}"
assert 'Widget,9.99,100' in result, f"Missing Widget row"
assert 'Gadget,24.99,50' in result, f"Missing Gadget row"
assert 'Doohickey,4.99,200' in result, f"Missing Doohickey row"
lines = [l for l in result.strip().split('\\n') if l.strip()]
assert len(lines) == 4, f"Expected 4 lines (header + 3 rows), got {len(lines)}"
""",
        "teardown": "os.unlink(_tmppath)",
    },
    {
        "id": "csv_dictwriter",
        "description": (
            "Use csv.DictWriter to write data to a StringIO buffer. "
            "The fieldnames are ['id', 'name', 'score']. Write these rows: "
            "{'id': '1', 'name': 'Alice', 'score': '95'}, "
            "{'id': '2', 'name': 'Bob', 'score': '87'}, "
            "{'id': '3', 'name': 'Charlie', 'score': '92'}. "
            "Make sure to include the header row. "
            "Assign the resulting CSV string to `result`."
        ),
        "test": """
assert isinstance(result, str), f"Expected str, got {type(result)}"
lines = [l for l in result.strip().split('\\n') if l.strip()]
assert len(lines) == 4, f"Expected 4 lines (header + 3), got {len(lines)}: {lines}"
assert 'id,name,score' in lines[0] or 'id' in lines[0], f"Header missing or wrong: {lines[0]}"
assert 'Alice' in result, f"Alice not found in output"
assert '87' in result, f"Bob's score 87 not found"
""",
    },
    {
        "id": "csv_quoting_special_chars",
        "description": (
            "Create a CSV string using csv.writer with a StringIO buffer. "
            "Write a header row ['name', 'description', 'price'] and these data rows: "
            "('Widget', 'A small, useful device', '9.99'), "
            "('Gadget', 'Contains \"quotes\" inside', '24.99'), "
            "('Thing', 'Has a\\nnewline', '4.99'). "
            "Use csv.QUOTE_ALL quoting mode so all fields are quoted. "
            "Assign the resulting string to `result`."
        ),
        "test": """
assert isinstance(result, str), f"Expected str, got {type(result)}"
# With QUOTE_ALL, every field should be quoted
assert '"name"' in result or '"Widget"' in result, f"Fields not quoted: {result[:100]}"
# The embedded quotes should be escaped (doubled)
assert '""quotes""' in result or '\\"quotes\\"' in result, f"Embedded quotes not escaped: {result}"
""",
    },
    {
        "id": "csv_custom_delimiter",
        "description": (
            "Parse a TSV (tab-separated) string using the csv module with a tab delimiter. "
            "The data is: 'name\\tage\\tcity\\nAlice\\t30\\tNew York\\nBob\\t25\\tLos Angeles'. "
            "Parse it into a list of dicts using DictReader with delimiter='\\t'. "
            "Assign the list of dicts to `result`."
        ),
        "setup": """
_tsv_data = "name\\tage\\tcity\\nAlice\\t30\\tNew York\\nBob\\t25\\tLos Angeles"
""",
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 2, f"Expected 2 rows, got {len(result)}"
assert result[0]['name'] == 'Alice', f"Expected Alice, got {result[0].get('name')}"
assert result[0]['city'] == 'New York', f"Expected 'New York', got {result[0].get('city')}"
assert result[1]['city'] == 'Los Angeles', f"Expected 'Los Angeles', got {result[1].get('city')}"
""",
    },
]
