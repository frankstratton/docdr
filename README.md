# DocDr

DocDr (Doc Doctor) is a simple tool for scaning a Golang codebase looking for
functions without godoc comments.  DocDr presents an interface to quickly add
comments and rewrites your source files.

# Usage
```
docdr run <source directory> <package name>
```

# Features


# TODO
* Fix non-doc comment preservation (this isn't an issue when opening the original file)
* Add commands:
	n: Never ask again; add a default doc string so we always skip this function
