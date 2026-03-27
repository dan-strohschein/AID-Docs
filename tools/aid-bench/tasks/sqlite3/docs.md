# sqlite3 — DB-API 2.0 interface for SQLite databases

SQLite is a C library that provides a lightweight disk-based database that doesn't require a separate server process. The sqlite3 module provides a SQL interface compliant with the DB-API 2.0 specification.

## Connection

`sqlite3.connect(database, timeout=5.0, detect_types=0, isolation_level='deferred', check_same_thread=True, factory=Connection, cached_statements=128, uri=False)`

Opens a connection to the SQLite database file `database`. Use `:memory:` to create a database in RAM. Returns a Connection object.

The connection can be used as a context manager that automatically commits or rolls back transactions:
```python
with sqlite3.connect(":memory:") as con:
    con.execute("CREATE TABLE ...")
    con.execute("INSERT ...")
    # auto-committed at end of with block
```

Note: The context manager does NOT close the connection. You must call `con.close()` separately or use a second `with` block / try-finally.

## Connection Methods

- `Connection.cursor()` — Create a new Cursor object.
- `Connection.commit()` — Commit the current transaction.
- `Connection.rollback()` — Roll back any changes since the last commit.
- `Connection.close()` — Close the database connection. Does NOT automatically commit.
- `Connection.execute(sql, parameters=())` — Shortcut that creates a cursor, calls execute(), and returns the cursor.
- `Connection.executemany(sql, seq_of_parameters)` — Shortcut that creates a cursor, calls executemany(), and returns the cursor.
- `Connection.executescript(sql_script)` — Execute multiple SQL statements at once. Issues a COMMIT first.

## Cursor Methods

- `Cursor.execute(sql, parameters=())` — Execute a single SQL statement. Use `?` as placeholder for parameters: `cursor.execute("SELECT * FROM t WHERE id=?", (id,))`. Never use string formatting for SQL parameters.
- `Cursor.executemany(sql, seq_of_parameters)` — Execute a SQL command against all parameter sequences in `seq_of_parameters`.
- `Cursor.fetchone()` — Fetch the next row, returning None if no more rows.
- `Cursor.fetchmany(size=cursor.arraysize)` — Fetch `size` rows.
- `Cursor.fetchall()` — Fetch all remaining rows as a list.

## Cursor Attributes

- `Cursor.description` — Column names and types of the last query.
- `Cursor.rowcount` — Number of rows modified by the last statement.
- `Cursor.lastrowid` — Row ID of the last inserted row.

## SQL Injection Prevention

Always use parameterized queries with `?` placeholders:
```python
# SAFE
cursor.execute("SELECT * FROM users WHERE name=?", (name,))

# UNSAFE — SQL injection risk
cursor.execute(f"SELECT * FROM users WHERE name='{name}'")
```

## Row Objects

By default, rows are tuples. Use `sqlite3.Row` as `row_factory` on the connection for dict-like access:
```python
con.row_factory = sqlite3.Row
cursor = con.execute("SELECT ...")
row = cursor.fetchone()
print(row["column_name"])
```

## Transactions

By default, sqlite3 opens transactions implicitly before DML statements (INSERT/UPDATE/DELETE). Call `connection.commit()` to persist changes, or `connection.rollback()` to discard them.
