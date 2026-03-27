# tarfile — Read and write tar archive files

The tarfile module makes it possible to read and write tar archives, including those using gzip, bz2 and lzma compression.

## Opening Archives

`tarfile.open(name=None, mode='r', fileobj=None, bufsize=10240)`

Open a tar archive. Returns a TarFile object.

Modes for reading:
- `'r'` or `'r:*'` — open for reading with transparent compression detection
- `'r:'` — open for reading, uncompressed
- `'r:gz'` — open for reading, gzip compressed
- `'r:bz2'` — open for reading, bzip2 compressed
- `'r:xz'` — open for reading, lzma compressed

Modes for writing:
- `'w'` or `'w:'` — open for writing, uncompressed
- `'w:gz'` — open for writing, gzip compressed
- `'w:bz2'` — open for writing, bzip2 compressed
- `'w:xz'` — open for writing, lzma compressed

Either `name` (filename) or `fileobj` (file-like object like BytesIO) must be provided. If both, `fileobj` is used and `name` is set on the archive.

TarFile objects can be used as context managers (with statement).

## Adding Files

- `TarFile.add(name, arcname=None, recursive=True, *, filter=None)` — Add the file `name` to the archive. `arcname` overrides the name inside the archive. If `recursive` is True and the name is a directory, it is added recursively.

- `TarFile.addfile(tarinfo, fileobj=None)` — Add a TarInfo object to the archive. If `fileobj` is given, `tarinfo.size` bytes are read from it. Use this for adding files from memory (BytesIO).

## Reading/Extracting

- `TarFile.getmembers()` — Return a list of TarInfo objects for all members.
- `TarFile.getnames()` — Return a list of member names.
- `TarFile.getmember(name)` — Return a TarInfo object for member `name`. Raises KeyError if not found.
- `TarFile.extractfile(member)` — Extract a member as a file-like object (read-only). Returns None for directories and other non-file entries.
- `TarFile.extractall(path=".", members=None)` — Extract all members to `path`.
- `TarFile.extract(member, path=".")` — Extract a single member to `path`.

**Security warning:** Never extract archives from untrusted sources without inspection. A malicious archive could use absolute paths or `../` to overwrite files outside the extraction directory.

## TarInfo Objects

`TarInfo(name="")` — Create a TarInfo object.

Key attributes:
- `name` — Name of the archive member
- `size` — File size in bytes
- `type` — Member type (REGTYPE for regular file, DIRTYPE for directory, etc.)

Key methods:
- `isfile()` / `isreg()` — Is this a regular file?
- `isdir()` — Is this a directory?
- `issym()` — Is this a symbolic link?

## Constants

- `tarfile.REGTYPE` — Regular file type
- `tarfile.DIRTYPE` — Directory type
