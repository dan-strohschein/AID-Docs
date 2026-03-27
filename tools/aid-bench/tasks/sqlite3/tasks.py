"""SQLite3 benchmark tasks — tests lifecycle, params, transactions, and edge cases."""

TASKS = [
    {
        "id": "sqlite3_basic_query",
        "description": (
            "Create an in-memory SQLite database. Create a 'users' table with columns "
            "(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL). "
            "Insert these 3 users: ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com'), "
            "('Charlie', 'charlie@example.com'). Query all users and assign the list of "
            "all (id, name, email) tuples to `result`."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 3, f"Expected 3 rows, got {len(result)}"
names = [row[1] for row in result]
assert 'Alice' in names, "Alice not found"
assert 'Bob' in names, "Bob not found"
assert 'Charlie' in names, "Charlie not found"
""",
    },
    {
        "id": "sqlite3_context_manager",
        "description": (
            "Create an in-memory SQLite database. Using the connection as a context manager "
            "(with statement), create a 'products' table (id INTEGER PRIMARY KEY, name TEXT, "
            "price REAL) and insert 2 products: ('Widget', 9.99) and ('Gadget', 24.99). "
            "After the with block, query the products and assign the list of rows to `result`. "
            "Make sure the connection is properly closed after you're done."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 2, f"Expected 2 rows, got {len(result)}"
prices = sorted([row[2] for row in result])
assert abs(prices[0] - 9.99) < 0.01, f"Expected 9.99, got {prices[0]}"
assert abs(prices[1] - 24.99) < 0.01, f"Expected 24.99, got {prices[1]}"
""",
    },
    {
        "id": "sqlite3_parameterized_query",
        "description": (
            "Create an in-memory SQLite database with a 'users' table "
            "(id INTEGER PRIMARY KEY, name TEXT, role TEXT). Insert these users: "
            "('Alice', 'admin'), ('Bob', 'user'), ('Charlie', 'user'), ('Diana', 'admin'). "
            "Then write a query that selects only users with role='admin' using a "
            "PARAMETERIZED query (NOT string formatting — use ? placeholders to prevent "
            "SQL injection). Assign the list of admin (id, name, role) tuples to `result`."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 2, f"Expected 2 admins, got {len(result)}"
names = sorted([row[1] for row in result])
assert names == ['Alice', 'Diana'], f"Expected Alice and Diana, got {names}"
# Verify all returned rows have role='admin'
for row in result:
    assert row[2] == 'admin', f"Non-admin row returned: {row}"
""",
    },
    {
        "id": "sqlite3_transaction_rollback",
        "description": (
            "Create an in-memory SQLite database with a 'accounts' table "
            "(id INTEGER PRIMARY KEY, name TEXT, balance REAL). Insert: "
            "('Alice', 1000.0), ('Bob', 500.0). "
            "Now simulate a transfer: deduct 200 from Alice and add 200 to Bob, "
            "but wrap it in a transaction. After the first UPDATE (deducting from Alice), "
            "intentionally cause the transaction to rollback (call connection.rollback()). "
            "Then query all accounts and assign the list to `result`. "
            "The balances should be UNCHANGED because the transaction was rolled back."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 2, f"Expected 2 accounts, got {len(result)}"
balances = {row[1]: row[2] for row in result}
assert abs(balances['Alice'] - 1000.0) < 0.01, f"Alice balance should be 1000, got {balances['Alice']}"
assert abs(balances['Bob'] - 500.0) < 0.01, f"Bob balance should be 500, got {balances['Bob']}"
""",
    },
    {
        "id": "sqlite3_executemany_returning",
        "description": (
            "Create an in-memory SQLite database with a 'logs' table "
            "(id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT, level TEXT). "
            "Use executemany() to insert these log entries in a single call: "
            "[('Server started', 'INFO'), ('Connection failed', 'ERROR'), "
            "('Retrying', 'WARN'), ('Connected', 'INFO'), ('Disk full', 'ERROR')]. "
            "Then query for only ERROR-level logs, ordered by id. "
            "Assign the list of (id, message, level) tuples to `result`."
        ),
        "test": """
assert isinstance(result, list), f"Expected list, got {type(result)}"
assert len(result) == 2, f"Expected 2 ERROR logs, got {len(result)}"
assert result[0][1] == 'Connection failed', f"First error should be 'Connection failed', got {result[0][1]}"
assert result[1][1] == 'Disk full', f"Second error should be 'Disk full', got {result[1][1]}"
for row in result:
    assert row[2] == 'ERROR', f"Non-ERROR row: {row}"
""",
    },
]
