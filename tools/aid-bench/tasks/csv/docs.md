# csv — CSV File Reading and Writing

The csv module implements classes to read and write tabular data in CSV format.

## Reading

`csv.reader(csvfile, dialect='excel', **fmtparams)`

Return a reader object which will iterate over lines in the given csvfile. csvfile can be any iterable that returns strings (file objects, StringIO, lists of strings).

`csv.DictReader(f, fieldnames=None, restkey=None, restval=None, dialect='excel', *args, **kwds)`

Like reader, but maps each row to a dict. If fieldnames is omitted, the first row is used as the header.

```python
import csv, io
reader = csv.DictReader(io.StringIO("name,age\nAlice,30\nBob,25"))
for row in reader:
    print(row['name'], row['age'])
```

## Writing

`csv.writer(csvfile, dialect='excel', **fmtparams)`

Return a writer object. Call `writerow(row)` to write a single row, or `writerows(rows)` for multiple rows.

`csv.DictWriter(f, fieldnames, restval='', extrasaction='raise', dialect='excel', *args, **kwds)`

Like writer, but maps dicts to rows. You MUST call `writeheader()` first to write the header row, then use `writerow(dict)` or `writerows(list_of_dicts)`.

```python
import csv, io
output = io.StringIO()
writer = csv.DictWriter(output, fieldnames=['name', 'age'])
writer.writeheader()  # REQUIRED — writes the header row
writer.writerow({'name': 'Alice', 'age': '30'})
```

## Important: newline parameter

When opening a file for CSV writing, always use `newline=''`:

```python
with open('output.csv', 'w', newline='') as f:
    writer = csv.writer(f)
```

Without `newline=''`, the csv module's own newline handling conflicts with Python's universal newline translation, producing double blank lines on Windows.

For reading, also use `newline=''`:
```python
with open('input.csv', newline='') as f:
    reader = csv.reader(f)
```

## Format Parameters

- `delimiter` — Field separator character. Default: `,`. Use `\t` for TSV.
- `quotechar` — Character for quoting fields. Default: `"`.
- `quoting` — Controls when quotes are generated:
  - `csv.QUOTE_MINIMAL` (default) — only quote when necessary
  - `csv.QUOTE_ALL` — quote all fields
  - `csv.QUOTE_NONNUMERIC` — quote all non-numeric fields
  - `csv.QUOTE_NONE` — never quote (use escapechar for special chars)

## Using StringIO

For in-memory CSV operations, use `io.StringIO()`:
```python
import csv, io
buf = io.StringIO()
writer = csv.writer(buf)
writer.writerow(['a', 'b', 'c'])
csv_string = buf.getvalue()
```
