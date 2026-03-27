# sqlite3 — Available Functions and Classes

sqlite3.connect(database, timeout, detect_types, isolation_level, check_same_thread, factory, cached_statements, uri)
sqlite3.Connection.cursor()
sqlite3.Connection.commit()
sqlite3.Connection.rollback()
sqlite3.Connection.close()
sqlite3.Connection.execute(sql, parameters)
sqlite3.Connection.executemany(sql, parameters)
sqlite3.Connection.executescript(sql_script)
sqlite3.Connection.total_changes
sqlite3.Connection.isolation_level
sqlite3.Cursor.execute(sql, parameters)
sqlite3.Cursor.executemany(sql, seq_of_parameters)
sqlite3.Cursor.fetchone()
sqlite3.Cursor.fetchmany(size)
sqlite3.Cursor.fetchall()
sqlite3.Cursor.description
sqlite3.Cursor.rowcount
sqlite3.Cursor.lastrowid
sqlite3.Row
