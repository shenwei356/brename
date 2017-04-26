# brename -- a cross-platform command-line tool for safely batch renaming files/directories via regular expression

[![Built with GoLang](https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/shenwei356/brename)](https://goreportcard.com/report/github.com/shenwei356/brename)
[![Cross-platform](https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat)](#download)
[![Latest Version](https://img.shields.io/github/release/shenwei356/brename.svg?style=flat?maxAge=86400)](https://github.com/shenwei356/brename/releases)
[![Github Releases](https://img.shields.io/github/downloads/shenwei356/brename/latest/total.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases)

`brename` is a cross-platform command-line tool for safely batch renaming files/directories via regular expression.

## Table of Contents
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Examples](#examples)
- [Contact](#contact)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

- **Cross-platform**. Supporting Windows, Mac OS X and Linux.
- **Safe**. By ***checking potential conflicts and errors***.
- **File filtering**. Supporting including and excluding files via regular expression.
    No need to run commands like `find ./ -name "*.html" -exec CMD`.
- **Recursively renaming both files and directories**.
- **Supporting dry run**.
- **Colorful output**. Screenshots:
    - Linux
        ![linux](screenshot/linux.png)
    - Windows
        ![windows](screenshot/windows.png)

## Installation

`brename` is implemented in [Go](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/brename/releases) page.

#### Method 1: Download binaries

[brename v2.1.2](https://github.com/shenwei356/brename/releases/tag/v2.1.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/brename/v2.1.2/total.svg)](https://github.com/shenwei356/brename/releases/tag/v2.1.2)


OS     |Arch      |File, (mirror为中国用户下载镜像链接)                                                                                                                                                                         |Download Count
:------|:---------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Linux  |32-bit    |[brename_linux_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_linux_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_386.tar.gz))                            |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_linux_386.tar.gz)
Linux  |**64-bit**|[**brename_linux_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_linux_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_amd64.tar.gz))                  |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_linux_amd64.tar.gz)
OS X   |32-bit    |[brename_darwin_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_darwin_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_386.tar.gz))                         |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_darwin_386.tar.gz)
OS X   |**64-bit**|[**brename_darwin_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_darwin_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_amd64.tar.gz))               |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_darwin_amd64.tar.gz)
Windows|32-bit    |[brename_windows_386.exe.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_windows_386.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_386.exe.tar.gz))          |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_386.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_windows_386.exe.tar.gz)
Windows|**64-bit**|[**brename_windows_amd64.exe.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_windows_amd64.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_amd64.exe.tar.gz))|[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_amd64.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.1.2/brename_windows_amd64.exe.tar.gz)


Just [download](https://github.com/shenwei356/brename/releases) compressed
executable file of your operating system,
and decompress it with `tar -zxvf *.tar.gz` command or other tools.
And then:

1. **For Linux-like systems**
    1. If you have root privilege simply copy it to `/usr/local/bin`:

            sudo cp brename /usr/local/bin/

    1. Or add the current directory of the executable file to environment variable
    `PATH`:

            echo export PATH=\$PATH:\"$(pwd)\" >> ~/.bashrc
            source ~/.bashrc


1. **For windows**, just copy `brename.exe` to `C:\WINDOWS\system32`.

#### Method 2: For Go developer

    go get -u github.com/shenwei356/brename/


## Usage

```
brename -- a cross-platform command-line tool for safely batch renaming files/directories via regular expression

Version: 2.1.2

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Attention:
  1. Paths starting with "." is ignored.
  2. Overwriting existed files is not allowed.
  3. Flag -f/--include-filters and -F/--exclude-filters support multiple values,
     e.g., -f ".html" -f ".htm".
     But ATTENTION: comma in filter is treated as separater of multiple filters.

Usage:
  brename [flags]

Examples:
  1. dry run and showing potential dangerous operations
      brename -p "abc" -d
  2. dry run and only show operations that will cause error
      brename -p "abc" -d -v 2
  3. only renaming specific paths via include filters
      brename -p ":" -r "-" -f ".htm$" -f ".html$"
  4. renaming all .jpeg files to .jpg in all subdirectories
      brename -p "\.jpeg" -r ".jpg" -R   dir
  5. using capture variables, e.g., $1, $2 ...
      brename -p "(a)" -r "\$1\$1"
      or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
  6. renaming directory too
      brename -p ":" -r "-" -R -D   pdf-dirs

  More examples: https://github.com/shenwei356/brename

Flags:
  -d, --dry-run                       print rename operations but do not run
  -F, --exclude-filters stringSlice   exclude file filter(s) (regular expression, case ignored). multiple values supported, e.g., -F ".html" -F ".htm", but ATTENTION: comma in filter is treated as separater of multiple filters
  -i, --ignore-case                   ignore case
  -f, --include-filters stringSlice   include file filter(s) (regular expression, case ignored). multiple values supported, e.g., -f ".html" -f ".htm", but ATTENTION: comma in filter is treated as separater of multiple filters (default [.])
  -D, --including-dir                 rename directories
  -p, --pattern string                search pattern (regular expression)
  -R, --recursive                     rename recursively
  -r, --replacement string            replacement. capture variables supported.  e.g. $1 represents the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character.
  -v, --verbose int                   verbose level (0 for all, 1 for warning and error, 2 for only error)
  -V, --version                       print version information and check for update

```


## Examples

Take a directory for example:

    $ tree
    .
    ├── abc
    │   ├── A.JPEG
    │   ├── B.HTM
    │   └── B.JPEG
    ├── a.jpeg
    ├── b.html
    └── b.jpeg


1. Recursively renaming all `.jpeg` files to `.jpg` in all subdirectories (`-R/--recursive`).
 A dry run is firstly performed for safety checking (`-d/--dry-run`).

        $ brename -p "\.jpeg" -r ".jpg" -R -d
        [INFO] checking: [ ok ] 'a.jpeg' -> 'a.jpg'
        [INFO] checking: [ ok ] 'b.jpeg' -> 'b.jpg'
        [INFO] 2 path(s) to be renamed

        $ brename -p "\.jpeg" -r ".jpg" -R
        [INFO] checking: [ ok ] 'a.jpeg' -> 'a.jpg'
        [INFO] checking: [ ok ] 'b.jpeg' -> 'b.jpg'
        [INFO] 2 path(s) to be renamed
        [INFO] renamed: 'a.jpeg' -> 'a.jpg'
        [INFO] renamed: 'b.jpeg' -> 'b.jpg'
        [INFO] 2 path(s) renamed

        $ tree
        .
        ├── abc
        │   ├── A.JPEG
        │   ├── B.HTM
        │   └── B.JPEG
        ├── a.jpg
        ├── b.html
        └── b.jpg


1. Dry run and only showing operations that will cause error (`-v/--verbose`)

        # default value of -v is 0
        $ brename -p a -r b -R -D -d
        [ERRO] checking: [ new path existed ] 'a.jpg' -> 'b.jpg'
        [INFO] checking: [ ok ] 'abc' -> 'bbc'
        [ERRO] 1 potential error(s) detected, please check

        $ brename -p a -r b -R -D -d -v 2
        [ERRO] checking: [ new path existed ] 'a.jpg' -> 'b.jpg'
        [ERRO] 1 potential error(s) detected, please check

1. Ignoring cases (`-i/--ignore-case`)

        $ brename -p "\.jpeg" -r ".jpg" -R -i
        [INFO] checking: [ ok ] 'abc/A.JPEG' -> 'abc/A.jpg'
        [INFO] checking: [ ok ] 'abc/B.JPEG' -> 'abc/B.jpg'
        [INFO] 2 path(s) to be renamed
        [INFO] renamed: 'abc/A.JPEG' -> 'abc/A.jpg'
        [INFO] renamed: 'abc/B.JPEG' -> 'abc/B.jpg'
        [INFO] 2 path(s) renamed

        $ tree
        .
        ├── abc
        │   ├── A.jpg
        │   ├── B.HTM
        │   └── B.jpg
        ├── a.jpg
        ├── b.html
        └── b.jpg

1. Using capture variables, e.g., $1, $2 ...

        # or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
        $ brename -p "(a)" -r "\$1\$1" -i
        [INFO] checking: [ ok ] 'a.jpg' -> 'aa.jpg'
        [INFO] 1 path(s) to be renamed
        [INFO] renamed: 'a.jpg' -> 'aa.jpg'
        [INFO] 1 path(s) renamed

        $ tree
        .
        ├── aa.jpg
        ├── abc
        │   ├── A.jpg
        │   ├── B.HTM
        │   └── B.jpg
        ├── b.html
        └── b.jpg


1. Renaming directory too (`-D/--including-dir`)

        $ brename -p "a" -r "A" -R -D
        [INFO] checking: [ ok ] 'aa.jpg' -> 'AA.jpg'
        [INFO] checking: [ ok ] 'abc' -> 'Abc'
        [INFO] 2 path(s) to be renamed
        [INFO] renamed: 'aa.jpg' -> 'AA.jpg'
        [INFO] renamed: 'abc' -> 'Abc'
        [INFO] 2 path(s) renamed

        $ tree
        .
        ├── AA.jpg
        ├── Abc
        │   ├── A.jpg
        │   ├── B.HTM
        │   └── B.jpg
        ├── b.html
        └── b.jpg

1. Only renaming specific files via include filters (regular expression) (`-f/--include-filters`)

        $ brename -p "^" -r "hello " -f ".htm$" -f ".html$" -R
        [INFO] checking: [ ok ] 'Abc/B.HTM' -> 'Abc/hello B.HTM'
        [INFO] checking: [ ok ] 'b.html' -> 'hello b.html'
        [INFO] 2 path(s) to be renamed
        [INFO] renamed: 'Abc/B.HTM' -> 'Abc/hello B.HTM'
        [INFO] renamed: 'b.html' -> 'hello b.html'
        [INFO] 2 path(s) renamed

        $ tree
        .
        ├── AA.jpg
        ├── Abc
        │   ├── A.jpg
        │   ├── B.jpg
        │   └── hello\ B.HTM
        ├── b.jpg
        └── hello\ b.html

1. Excluding files via exclude filters (regular expression) (`-F/--exclude-filters`)

        $ brename -p b -r c -d
        [INFO] checking: [ ok ] 'b.jpg' -> 'c.jpg'
        [INFO] checking: [ ok ] 'hello b.html' -> 'hello c.html'
        [INFO] 2 path(s) to be renamed

        $ brename -p b -r c -d -F '.html$'
        [INFO] checking: [ ok ] 'b.jpg' -> 'c.jpg'
        [INFO] 2 path(s) to be renamed

## Contact

[Create an issue](https://github.com/shenwei356/brename/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)
