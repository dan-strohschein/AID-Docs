# tarfile — Available Functions and Classes

tarfile.open(name, mode, fileobj, bufsize)
tarfile.TarFile.add(name, arcname, recursive, filter)
tarfile.TarFile.addfile(tarinfo, fileobj)
tarfile.TarFile.getmember(name)
tarfile.TarFile.getmembers()
tarfile.TarFile.getnames()
tarfile.TarFile.extractall(path, members, numeric_owner)
tarfile.TarFile.extract(member, path, set_attrs, numeric_owner)
tarfile.TarFile.extractfile(member)
tarfile.TarFile.close()
tarfile.TarFile.next()
tarfile.TarInfo(name)
tarfile.TarInfo.name
tarfile.TarInfo.size
tarfile.TarInfo.type
tarfile.TarInfo.isfile()
tarfile.TarInfo.isdir()
tarfile.REGTYPE
tarfile.DIRTYPE
