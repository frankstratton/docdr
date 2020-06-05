# DocDr

DocDr (Doc Doctor) is a simple tool for scaning a Golang codebase looking for
functions without godoc comments.  DocDr presents an interface to quickly add
comments and rewrites your source files.

# Install & Usage

> :warning: **This project is considered an alpha; it may have unintented sideeffects; always backup your work before running**

## Install

```
go get github.com/frankstratton/docdr
```

## Running

Analyze a source directory and print a coverage report for documented functions
```
docdr run <source directory>
```

Parse a package for undocumented functions and prompt for comments
```
docdr run <source directory> <package name>
```

# Features


# TODO
* Add a subcommand to select the 'best' function for comments via some
  heuristic measure; e.g. -- least covered package + longest function
* Add interactive commands:
	* n: Never ask again; add a default doc string so we always skip this function
* Fix offsets when editing the original file. Currently we don't rescan/reload Positions
  so if you add comments to one function, the next edit in the same file opens to the wrong line.
* Other editor support as necessary
* Termbox/Tcell/Other better UI for terminal only use
