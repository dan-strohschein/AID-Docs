"""Run benchmark evaluations from pre-generated code blocks."""

import sys
sys.path.insert(0, "src")

from aid_bench.evaluator import evaluate
from aid_bench.runner import load_tasks

# All generated code from subagents, organized by library -> condition -> task_index
GENERATED = {}

# === SQLITE3 ===

GENERATED[("sqlite3", "blind")] = [
# Task 1
"""
import sqlite3
conn1 = sqlite3.connect(":memory:")
conn1.execute("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL)")
conn1.executemany("INSERT INTO users (name, email) VALUES (?, ?)", [
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Charlie', 'charlie@example.com')
])
conn1.commit()
result = conn1.execute("SELECT id, name, email FROM users").fetchall()
conn1.close()
""",
# Task 2
"""
import sqlite3
conn2 = sqlite3.connect(":memory:")
with conn2:
    conn2.execute("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)")
    conn2.executemany("INSERT INTO products (name, price) VALUES (?, ?)", [
        ('Widget', 9.99),
        ('Gadget', 24.99)
    ])
result = conn2.execute("SELECT * FROM products").fetchall()
conn2.close()
""",
# Task 3
"""
import sqlite3
conn3 = sqlite3.connect(":memory:")
conn3.execute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, role TEXT)")
conn3.executemany("INSERT INTO users (name, role) VALUES (?, ?)", [
    ('Alice', 'admin'),
    ('Bob', 'user'),
    ('Charlie', 'user'),
    ('Diana', 'admin')
])
conn3.commit()
result = conn3.execute("SELECT id, name, role FROM users WHERE role = ?", ('admin',)).fetchall()
conn3.close()
""",
# Task 4
"""
import sqlite3
conn4 = sqlite3.connect(":memory:")
conn4.execute("CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)")
conn4.executemany("INSERT INTO accounts (name, balance) VALUES (?, ?)", [
    ('Alice', 1000.0),
    ('Bob', 500.0)
])
conn4.commit()
conn4.execute("UPDATE accounts SET balance = balance - 200 WHERE name = 'Alice'")
conn4.rollback()
result = conn4.execute("SELECT id, name, balance FROM accounts").fetchall()
conn4.close()
""",
# Task 5
"""
import sqlite3
conn5 = sqlite3.connect(":memory:")
conn5.execute("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT, level TEXT)")
conn5.executemany("INSERT INTO logs (message, level) VALUES (?, ?)", [
    ('Server started', 'INFO'),
    ('Connection failed', 'ERROR'),
    ('Retrying', 'WARN'),
    ('Connected', 'INFO'),
    ('Disk full', 'ERROR')
])
conn5.commit()
result = conn5.execute("SELECT id, message, level FROM logs WHERE level = ? ORDER BY id", ('ERROR',)).fetchall()
conn5.close()
""",
]

GENERATED[("sqlite3", "human")] = [
# Task 1
"""
import sqlite3
con1 = sqlite3.connect(":memory:")
con1.execute("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL)")
con1.executemany("INSERT INTO users (name, email) VALUES (?, ?)", [
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Charlie', 'charlie@example.com')
])
con1.commit()
result = con1.execute("SELECT id, name, email FROM users").fetchall()
con1.close()
""",
# Task 2
"""
import sqlite3
con2 = sqlite3.connect(":memory:")
with con2:
    con2.execute("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)")
    con2.executemany("INSERT INTO products (name, price) VALUES (?, ?)", [
        ('Widget', 9.99),
        ('Gadget', 24.99)
    ])
result = con2.execute("SELECT * FROM products").fetchall()
con2.close()
""",
# Task 3
"""
import sqlite3
con3 = sqlite3.connect(":memory:")
con3.execute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, role TEXT)")
con3.executemany("INSERT INTO users (name, role) VALUES (?, ?)", [
    ('Alice', 'admin'),
    ('Bob', 'user'),
    ('Charlie', 'user'),
    ('Diana', 'admin')
])
con3.commit()
result = con3.execute("SELECT id, name, role FROM users WHERE role=?", ('admin',)).fetchall()
con3.close()
""",
# Task 4
"""
import sqlite3
con4 = sqlite3.connect(":memory:")
con4.execute("CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)")
con4.executemany("INSERT INTO accounts (name, balance) VALUES (?, ?)", [
    ('Alice', 1000.0),
    ('Bob', 500.0)
])
con4.commit()
con4.execute("UPDATE accounts SET balance = balance - 200 WHERE name = ?", ('Alice',))
con4.rollback()
result = con4.execute("SELECT id, name, balance FROM accounts").fetchall()
con4.close()
""",
# Task 5
"""
import sqlite3
con5 = sqlite3.connect(":memory:")
con5.execute("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT, level TEXT)")
con5.executemany("INSERT INTO logs (message, level) VALUES (?, ?)", [
    ('Server started', 'INFO'),
    ('Connection failed', 'ERROR'),
    ('Retrying', 'WARN'),
    ('Connected', 'INFO'),
    ('Disk full', 'ERROR')
])
con5.commit()
result = con5.execute("SELECT id, message, level FROM logs WHERE level=? ORDER BY id", ('ERROR',)).fetchall()
con5.close()
""",
]

GENERATED[("sqlite3", "aid_l1")] = [
# Task 1
"""
import sqlite3
conn = sqlite3.connect(":memory:")
cur = conn.cursor()
cur.execute("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL)")
cur.executemany("INSERT INTO users (name, email) VALUES (?, ?)", [
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Charlie', 'charlie@example.com'),
])
conn.commit()
cur.execute("SELECT id, name, email FROM users")
result = cur.fetchall()
conn.close()
""",
# Task 2
"""
import sqlite3
conn = sqlite3.connect(":memory:")
with conn:
    conn.execute("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)")
    conn.execute("INSERT INTO products (name, price) VALUES (?, ?)", ('Widget', 9.99))
    conn.execute("INSERT INTO products (name, price) VALUES (?, ?)", ('Gadget', 24.99))
cur = conn.cursor()
cur.execute("SELECT * FROM products")
result = cur.fetchall()
conn.close()
""",
# Task 3
"""
import sqlite3
conn = sqlite3.connect(":memory:")
conn.execute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, role TEXT)")
conn.executemany("INSERT INTO users (name, role) VALUES (?, ?)", [
    ('Alice', 'admin'),
    ('Bob', 'user'),
    ('Charlie', 'user'),
    ('Diana', 'admin'),
])
conn.commit()
cur = conn.cursor()
cur.execute("SELECT id, name, role FROM users WHERE role = ?", ('admin',))
result = cur.fetchall()
conn.close()
""",
# Task 4
"""
import sqlite3
conn = sqlite3.connect(":memory:")
conn.execute("CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)")
conn.execute("INSERT INTO accounts (name, balance) VALUES (?, ?)", ('Alice', 1000.0))
conn.execute("INSERT INTO accounts (name, balance) VALUES (?, ?)", ('Bob', 500.0))
conn.commit()
conn.execute("UPDATE accounts SET balance = balance - 200 WHERE name = 'Alice'")
conn.rollback()
cur = conn.cursor()
cur.execute("SELECT id, name, balance FROM accounts")
result = cur.fetchall()
conn.close()
""",
# Task 5
"""
import sqlite3
conn = sqlite3.connect(":memory:")
conn.execute("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT, level TEXT)")
conn.executemany("INSERT INTO logs (message, level) VALUES (?, ?)", [
    ('Server started', 'INFO'),
    ('Connection failed', 'ERROR'),
    ('Retrying', 'WARN'),
    ('Connected', 'INFO'),
    ('Disk full', 'ERROR'),
])
conn.commit()
cur = conn.cursor()
cur.execute("SELECT id, message, level FROM logs WHERE level = ? ORDER BY id", ('ERROR',))
result = cur.fetchall()
conn.close()
""",
]

GENERATED[("sqlite3", "aid_full")] = [
# Task 1
"""
import sqlite3
con = sqlite3.connect(":memory:")
con.execute("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL)")
con.executemany("INSERT INTO users (name, email) VALUES (?, ?)", [
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Charlie', 'charlie@example.com'),
])
con.commit()
cursor = con.execute("SELECT id, name, email FROM users")
result = cursor.fetchall()
con.close()
""",
# Task 2
"""
import sqlite3
con = sqlite3.connect(":memory:")
with con:
    con.execute("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)")
    con.execute("INSERT INTO products (name, price) VALUES (?, ?)", ('Widget', 9.99))
    con.execute("INSERT INTO products (name, price) VALUES (?, ?)", ('Gadget', 24.99))
cursor = con.execute("SELECT * FROM products")
result = cursor.fetchall()
con.close()
""",
# Task 3
"""
import sqlite3
con = sqlite3.connect(":memory:")
con.execute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, role TEXT)")
con.executemany("INSERT INTO users (name, role) VALUES (?, ?)", [
    ('Alice', 'admin'),
    ('Bob', 'user'),
    ('Charlie', 'user'),
    ('Diana', 'admin'),
])
con.commit()
cursor = con.execute("SELECT id, name, role FROM users WHERE role = ?", ('admin',))
result = cursor.fetchall()
con.close()
""",
# Task 4
"""
import sqlite3
con = sqlite3.connect(":memory:")
con.execute("CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)")
con.executemany("INSERT INTO accounts (name, balance) VALUES (?, ?)", [
    ('Alice', 1000.0),
    ('Bob', 500.0),
])
con.commit()
con.execute("UPDATE accounts SET balance = balance - 200 WHERE name = ?", ('Alice',))
con.rollback()
cursor = con.execute("SELECT id, name, balance FROM accounts")
result = cursor.fetchall()
con.close()
""",
# Task 5
"""
import sqlite3
con = sqlite3.connect(":memory:")
con.execute("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT, level TEXT)")
con.executemany("INSERT INTO logs (message, level) VALUES (?, ?)", [
    ('Server started', 'INFO'),
    ('Connection failed', 'ERROR'),
    ('Retrying', 'WARN'),
    ('Connected', 'INFO'),
    ('Disk full', 'ERROR'),
])
con.commit()
cursor = con.execute("SELECT id, message, level FROM logs WHERE level = ? ORDER BY id", ('ERROR',))
result = cursor.fetchall()
con.close()
""",
]

# === TARFILE ===

GENERATED[("tarfile", "blind")] = [
# Task 1
"""
import tarfile, io
buf1 = io.BytesIO()
with tarfile.open(fileobj=buf1, mode='w:gz') as tf:
    for name, content in [('hello.txt', 'Hello World'), ('data.txt', '12345')]:
        data = content.encode()
        ti = tarfile.TarInfo(name=name)
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
buf1.seek(0)
with tarfile.open(fileobj=buf1, mode='r:gz') as tf:
    result = sorted(tf.getnames())
""",
# Task 2
"""
import tarfile, io
buf2 = io.BytesIO()
with tarfile.open(fileobj=buf2, mode='w') as tf:
    for name, content in [('a.txt', 'AAA'), ('b.txt', 'BBB'), ('c.txt', 'CCC')]:
        data = content.encode()
        ti = tarfile.TarInfo(name=name)
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
buf2.seek(0)
with tarfile.open(fileobj=buf2, mode='r') as tf:
    f = tf.extractfile('b.txt')
    result = f.read().decode()
""",
# Task 3
"""
import tarfile, io
buf3 = io.BytesIO()
with tarfile.open(fileobj=buf3, mode='w:gz') as tf:
    tf.add(_tmppath, arcname='public/data.txt')
buf3.seek(0)
with tarfile.open(fileobj=buf3, mode='r:gz') as tf:
    result = tf.getnames()
""",
# Task 4
"""
import tarfile, io
buf4 = io.BytesIO()
with tarfile.open(fileobj=buf4, mode='w') as tf:
    data = b'hello'
    ti = tarfile.TarInfo(name='readme.txt')
    ti.size = len(data)
    ti.type = tarfile.REGTYPE
    tf.addfile(ti, io.BytesIO(data))
    di = tarfile.TarInfo(name='subdir/')
    di.type = tarfile.DIRTYPE
    tf.addfile(di)
buf4.seek(0)
with tarfile.open(fileobj=buf4, mode='r') as tf:
    result = [m.name for m in tf.getmembers() if m.isfile()]
""",
# Task 5
"""
import tarfile, io
sizes = {}
for key, mode in [('tar', 'w'), ('gz', 'w:gz'), ('bz2', 'w:bz2')]:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode=mode) as tf:
        data = b'test data'
        ti = tarfile.TarInfo(name='test.txt')
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
    sizes[key] = buf.tell()
result = sizes
""",
]

GENERATED[("tarfile", "human")] = [
# Task 1
"""
import io, tarfile
buf1 = io.BytesIO()
with tarfile.open(fileobj=buf1, mode='w:gz') as tf:
    for name, content in [('hello.txt', 'Hello World'), ('data.txt', '12345')]:
        data = content.encode()
        ti = tarfile.TarInfo(name=name)
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
buf1.seek(0)
with tarfile.open(fileobj=buf1, mode='r:gz') as tf:
    result = sorted(tf.getnames())
""",
# Task 2
"""
import io, tarfile
buf2 = io.BytesIO()
with tarfile.open(fileobj=buf2, mode='w') as tf:
    for name, content in [('a.txt', 'AAA'), ('b.txt', 'BBB'), ('c.txt', 'CCC')]:
        data = content.encode()
        ti = tarfile.TarInfo(name=name)
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
buf2.seek(0)
with tarfile.open(fileobj=buf2, mode='r') as tf:
    result = tf.extractfile('b.txt').read().decode()
""",
# Task 3
"""
import io, tarfile
buf3 = io.BytesIO()
with tarfile.open(fileobj=buf3, mode='w:gz') as tf:
    tf.add(_tmppath, arcname='public/data.txt')
buf3.seek(0)
with tarfile.open(fileobj=buf3, mode='r:gz') as tf:
    result = tf.getnames()
""",
# Task 4
"""
import io, tarfile
buf4 = io.BytesIO()
with tarfile.open(fileobj=buf4, mode='w') as tf:
    data = b'hello'
    ti = tarfile.TarInfo(name='readme.txt')
    ti.size = len(data)
    tf.addfile(ti, io.BytesIO(data))
    di = tarfile.TarInfo(name='subdir/')
    di.type = tarfile.DIRTYPE
    tf.addfile(di)
buf4.seek(0)
with tarfile.open(fileobj=buf4, mode='r') as tf:
    result = [m.name for m in tf.getmembers() if m.isfile()]
""",
# Task 5
"""
import io, tarfile
sizes = {}
for key, mode in [('tar', 'w'), ('gz', 'w:gz'), ('bz2', 'w:bz2')]:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode=mode) as tf:
        data = b'test data'
        ti = tarfile.TarInfo(name='test.txt')
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))
    sizes[key] = len(buf.getvalue())
result = sizes
""",
]

GENERATED[("tarfile", "aid_l1")] = [
# Task 1
"""
import tarfile, io
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w:gz') as tar:
    for name, content in [('hello.txt', 'Hello World'), ('data.txt', '12345')]:
        data = content.encode('utf-8')
        info = tarfile.TarInfo(name=name)
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
buf.seek(0)
with tarfile.open(fileobj=buf, mode='r:gz') as tar:
    result = sorted(tar.getnames())
""",
# Task 2
"""
import tarfile, io
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as tar:
    for name, content in [('a.txt', 'AAA'), ('b.txt', 'BBB'), ('c.txt', 'CCC')]:
        data = content.encode('utf-8')
        info = tarfile.TarInfo(name=name)
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
buf.seek(0)
with tarfile.open(fileobj=buf, mode='r') as tar:
    f = tar.extractfile('b.txt')
    result = f.read().decode('utf-8')
""",
# Task 3
"""
import tarfile, io
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w:gz') as tar:
    tar.add(_tmppath, arcname='public/data.txt')
buf.seek(0)
with tarfile.open(fileobj=buf, mode='r:gz') as tar:
    result = tar.getnames()
""",
# Task 4
"""
import tarfile, io
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as tar:
    data = b'hello'
    info = tarfile.TarInfo(name='readme.txt')
    info.size = len(data)
    tar.addfile(info, io.BytesIO(data))
    dirinfo = tarfile.TarInfo(name='subdir/')
    dirinfo.type = tarfile.DIRTYPE
    tar.addfile(dirinfo)
buf.seek(0)
with tarfile.open(fileobj=buf, mode='r') as tar:
    result = [m.name for m in tar.getmembers() if m.isfile()]
""",
# Task 5
"""
import tarfile, io
sizes = {}
for key, mode in [('tar', 'w'), ('gz', 'w:gz'), ('bz2', 'w:bz2')]:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode=mode) as tar:
        data = b'test data'
        info = tarfile.TarInfo(name='test.txt')
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
    sizes[key] = len(buf.getvalue())
result = sizes
""",
]

GENERATED[("tarfile", "aid_full")] = [
# Task 1
"""
import tarfile, io
buf = io.BytesIO()
tar = tarfile.open(fileobj=buf, mode='w:gz')
for fname, content in [('hello.txt', 'Hello World'), ('data.txt', '12345')]:
    data = content.encode('utf-8')
    info = tarfile.TarInfo(name=fname)
    info.size = len(data)
    tar.addfile(info, io.BytesIO(data))
tar.close()
buf.seek(0)
tar_r = tarfile.open(fileobj=buf, mode='r:gz')
result = sorted(tar_r.getnames())
tar_r.close()
""",
# Task 2
"""
import tarfile, io
buf = io.BytesIO()
tar = tarfile.open(fileobj=buf, mode='w')
for fname, content in [('a.txt', 'AAA'), ('b.txt', 'BBB'), ('c.txt', 'CCC')]:
    data = content.encode('utf-8')
    info = tarfile.TarInfo(name=fname)
    info.size = len(data)
    tar.addfile(info, io.BytesIO(data))
tar.close()
buf.seek(0)
tar_r = tarfile.open(fileobj=buf, mode='r')
f = tar_r.extractfile('b.txt')
result = f.read().decode('utf-8')
tar_r.close()
""",
# Task 3
"""
import tarfile, io
buf = io.BytesIO()
tar = tarfile.open(fileobj=buf, mode='w:gz')
tar.add(_tmppath, arcname='public/data.txt')
tar.close()
buf.seek(0)
tar_r = tarfile.open(fileobj=buf, mode='r:gz')
result = tar_r.getnames()
tar_r.close()
""",
# Task 4
"""
import tarfile, io
buf = io.BytesIO()
tar = tarfile.open(fileobj=buf, mode='w')
file_data = b'hello'
info_file = tarfile.TarInfo(name='readme.txt')
info_file.size = len(file_data)
info_file.type = tarfile.REGTYPE
tar.addfile(info_file, io.BytesIO(file_data))
info_dir = tarfile.TarInfo(name='subdir/')
info_dir.type = tarfile.DIRTYPE
tar.addfile(info_dir)
tar.close()
buf.seek(0)
tar_r = tarfile.open(fileobj=buf, mode='r')
result = [m.name for m in tar_r.getmembers() if m.isfile()]
tar_r.close()
""",
# Task 5
"""
import tarfile, io
sizes = {}
for key, mode in [('tar', 'w'), ('gz', 'w:gz'), ('bz2', 'w:bz2')]:
    buf = io.BytesIO()
    tar = tarfile.open(fileobj=buf, mode=mode)
    data = b'test data'
    info = tarfile.TarInfo(name='test.txt')
    info.size = len(data)
    tar.addfile(info, io.BytesIO(data))
    tar.close()
    sizes[key] = len(buf.getvalue())
result = sizes
""",
]

# === CSV ===

GENERATED[("csv", "blind")] = [
# Task 1
"""
import csv, io
reader = csv.DictReader(io.StringIO(_csv_data))
result = [dict(row) for row in reader]
""",
# Task 2
"""
import csv
with open(_tmppath, "w", newline="") as f:
    writer = csv.writer(f)
    writer.writerow(["product", "price", "quantity"])
    writer.writerows([("Widget", "9.99", "100"), ("Gadget", "24.99", "50"), ("Doohickey", "4.99", "200")])
with open(_tmppath, "r") as f:
    result = f.read()
""",
# Task 3
"""
import csv, io
buf = io.StringIO()
writer = csv.DictWriter(buf, fieldnames=["id", "name", "score"])
writer.writeheader()
writer.writerows([
    {"id": "1", "name": "Alice", "score": "95"},
    {"id": "2", "name": "Bob", "score": "87"},
    {"id": "3", "name": "Charlie", "score": "92"},
])
result = buf.getvalue()
""",
# Task 4
"""
import csv, io
buf = io.StringIO()
writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
writer.writerow(["name", "description", "price"])
writer.writerows([
    ("Widget", "A small, useful device", "9.99"),
    ("Gadget", 'Contains "quotes" inside', "24.99"),
    ("Thing", "Has a\\nnewline", "4.99"),
])
result = buf.getvalue()
""",
# Task 5
"""
import csv, io
reader = csv.DictReader(io.StringIO(_tsv_data), delimiter="\\t")
result = [dict(row) for row in reader]
""",
]

GENERATED[("csv", "human")] = [
# Task 1
"""
import csv, io
reader = csv.DictReader(io.StringIO(_csv_data))
result = [dict(row) for row in reader]
""",
# Task 2
"""
import csv
with open(_tmppath, 'w', newline='') as f:
    writer = csv.writer(f)
    writer.writerow(['product', 'price', 'quantity'])
    writer.writerows([
        ('Widget', '9.99', '100'),
        ('Gadget', '24.99', '50'),
        ('Doohickey', '4.99', '200'),
    ])
with open(_tmppath, 'r') as f:
    result = f.read()
""",
# Task 3
"""
import csv, io
buf = io.StringIO()
writer = csv.DictWriter(buf, fieldnames=['id', 'name', 'score'])
writer.writeheader()
writer.writerows([
    {'id': '1', 'name': 'Alice', 'score': '95'},
    {'id': '2', 'name': 'Bob', 'score': '87'},
    {'id': '3', 'name': 'Charlie', 'score': '92'},
])
result = buf.getvalue()
""",
# Task 4
"""
import csv, io
buf = io.StringIO()
writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
writer.writerow(['name', 'description', 'price'])
writer.writerows([
    ('Widget', 'A small, useful device', '9.99'),
    ('Gadget', 'Contains "quotes" inside', '24.99'),
    ('Thing', 'Has a\\nnewline', '4.99'),
])
result = buf.getvalue()
""",
# Task 5
"""
import csv, io
reader = csv.DictReader(io.StringIO(_tsv_data), delimiter='\\t')
result = [dict(row) for row in reader]
""",
]

GENERATED[("csv", "aid_l1")] = [
# Task 1
"""
import csv, io
source = io.StringIO(_csv_data)
reader = csv.DictReader(source)
result = [dict(row) for row in reader]
""",
# Task 2
"""
import csv
with open(_tmppath, 'w', newline='') as f:
    writer = csv.writer(f)
    writer.writerow(['product', 'price', 'quantity'])
    writer.writerows([
        ('Widget', '9.99', '100'),
        ('Gadget', '24.99', '50'),
        ('Doohickey', '4.99', '200'),
    ])
with open(_tmppath, 'r') as f:
    result = f.read()
""",
# Task 3
"""
import csv, io
buf = io.StringIO()
writer = csv.DictWriter(buf, fieldnames=['id', 'name', 'score'])
writer.writeheader()
writer.writerows([
    {'id': '1', 'name': 'Alice', 'score': '95'},
    {'id': '2', 'name': 'Bob', 'score': '87'},
    {'id': '3', 'name': 'Charlie', 'score': '92'},
])
result = buf.getvalue()
""",
# Task 4
"""
import csv, io
buf = io.StringIO()
writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
writer.writerow(['name', 'description', 'price'])
writer.writerows([
    ('Widget', 'A small, useful device', '9.99'),
    ('Gadget', 'Contains "quotes" inside', '24.99'),
    ('Thing', 'Has a\\nnewline', '4.99'),
])
result = buf.getvalue()
""",
# Task 5
"""
import csv, io
reader = csv.DictReader(io.StringIO(_tsv_data), delimiter='\\t')
result = [dict(row) for row in reader]
""",
]

GENERATED[("csv", "aid_full")] = [
# Task 1
"""
import csv, io
source = io.StringIO(_csv_data)
reader = csv.DictReader(source, delimiter=',')
result = list(reader)
""",
# Task 2
"""
import csv
with open(_tmppath, 'w', newline='') as f:
    writer = csv.DictWriter(f, fieldnames=['product', 'price', 'quantity'])
    writer.writeheader()
    writer.writerows([
        {'product': 'Widget', 'price': '9.99', 'quantity': '100'},
        {'product': 'Gadget', 'price': '24.99', 'quantity': '50'},
        {'product': 'Doohickey', 'price': '4.99', 'quantity': '200'},
    ])
with open(_tmppath, 'r', newline='') as f:
    result = f.read()
""",
# Task 3
"""
import csv, io
buf = io.StringIO()
writer = csv.DictWriter(buf, fieldnames=['id', 'name', 'score'])
writer.writeheader()
writer.writerows([
    {'id': '1', 'name': 'Alice', 'score': '95'},
    {'id': '2', 'name': 'Bob', 'score': '87'},
    {'id': '3', 'name': 'Charlie', 'score': '92'},
])
result = buf.getvalue()
""",
# Task 4
"""
import csv, io
buf = io.StringIO()
writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
writer.writerow(['name', 'description', 'price'])
writer.writerows([
    ('Widget', 'A small, useful device', '9.99'),
    ('Gadget', 'Contains "quotes" inside', '24.99'),
    ('Thing', 'Has a\\nnewline', '4.99'),
])
result = buf.getvalue()
""",
# Task 5
"""
import csv, io
source = io.StringIO(_tsv_data)
reader = csv.DictReader(source, delimiter='\\t')
result = list(reader)
""",
]


def main():
    conditions = ["blind", "human", "aid_l1", "aid_full"]
    libraries = ["sqlite3", "tarfile", "csv"]

    # Results: library -> task_id -> condition -> pass/fail
    results = {}

    for lib in libraries:
        tasks = load_tasks(lib)
        results[lib] = {}

        for task_idx, task in enumerate(tasks):
            task_id = task["id"]
            results[lib][task_id] = {}

            for condition in conditions:
                code_list = GENERATED.get((lib, condition), [])
                if task_idx >= len(code_list):
                    results[lib][task_id][condition] = ("SKIP", "No code")
                    continue

                code = code_list[task_idx]
                r = evaluate(
                    generated_code=code,
                    test_code=task["test"],
                    setup_code=task.get("setup", ""),
                    teardown_code=task.get("teardown", ""),
                )
                status = "PASS" if r.passed else "FAIL"
                results[lib][task_id][condition] = (status, r.error)

    # Print report
    print()
    print("=" * 70)
    print("  AID Benchmark Results")
    print("=" * 70)

    for lib in libraries:
        print(f"\nLibrary: {lib}")
        print("-" * 70)

        task_ids = list(results[lib].keys())
        # Header
        header = f"{'Task':<30}"
        for c in conditions:
            header += f" | {c:^10}"
        print(header)
        print("-" * len(header))

        pass_counts = {c: 0 for c in conditions}
        total = len(task_ids)

        for task_id in task_ids:
            display = task_id.replace(f"{lib}_", "")
            row = f"{display:<30}"
            for c in conditions:
                status, error = results[lib][task_id][c]
                row += f" | {status:^10}"
                if status == "PASS":
                    pass_counts[c] += 1
            print(row)

        print("-" * len(header))
        rate_row = f"{'Pass rate':<30}"
        for c in conditions:
            pct = f"{100 * pass_counts[c] // total}%"
            rate_row += f" | {pct:^10}"
        print(rate_row)

    # Overall summary
    print("\n" + "=" * 70)
    print("  Overall Summary")
    print("=" * 70)

    for c in conditions:
        total_pass = 0
        total_tasks = 0
        for lib in libraries:
            for task_id in results[lib]:
                status, _ = results[lib][task_id][c]
                total_tasks += 1
                if status == "PASS":
                    total_pass += 1
        pct = 100 * total_pass / total_tasks if total_tasks else 0
        bar = "#" * int(pct / 5) + "." * (20 - int(pct / 5))
        print(f"  {c:<10}  [{bar}]  {pct:.0f}%  ({total_pass}/{total_tasks})")


if __name__ == "__main__":
    main()
