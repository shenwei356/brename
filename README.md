# brename -- a cross-platform command-line tool for safely batch renaming files/directories

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
- **Safe**. By checking potential conflicts and errors.
- **Supporting dry run**.

## Installation

`brename` is implemented in [Go](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/brename/releases) page.

#### Method 1: Download binaries

[brename v2.0](https://github.com/shenwei356/brename/releases/tag/v2.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/brename/v2.0/total.svg)](https://github.com/shenwei356/brename/releases/tag/v0.1.7)


OS     |Arch      |File, (mirror为中国用户下载镜像链接)                                                                                                                                                                         |Download Count
:------|:---------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Linux  |32-bit    |[brename_linux_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.0/brename_linux_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_386.tar.gz))                            |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_linux_386.tar.gz)
Linux  |**64-bit**|[**brename_linux_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.0/brename_linux_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_amd64.tar.gz))                  |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_linux_amd64.tar.gz)
OS X   |32-bit    |[brename_darwin_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.0/brename_darwin_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_386.tar.gz))                         |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_darwin_386.tar.gz)
OS X   |**64-bit**|[**brename_darwin_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.0/brename_darwin_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_amd64.tar.gz))               |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_darwin_amd64.tar.gz)
Windows|32-bit    |[brename_windows_386.exe.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.0/brename_windows_386.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_386.exe.tar.gz))          |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_386.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_windows_386.exe.tar.gz)
Windows|**64-bit**|[**brename_windows_amd64.exe.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.0/brename_windows_amd64.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_amd64.exe.tar.gz))|[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_amd64.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.0/brename_windows_amd64.exe.tar.gz)


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
brename -- a cross-platform command-line tool for safely batch renaming files/directories

Version: 0.2.0

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Attention:
  1. Paths starting with "." is ignored
  2. Overwriting existed files is not allowed

Usage:
  brename [flags]

Examples:
  1. dry run and showing potential dangerous operations
      brename -p "abc" -d
  2. dry run and only show operations that will cause error
      brename -p "abc" -d -v 2
  3. renaming all .jpeg files to .jpg in all subdirectories
      brename -p "\.jpeg" -r ".jpg" -R   dir
  4. using capture variables, e.g., $1, $2 ...
      brename -p "(a)" -r "\$1\$1"
      or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
  5. even renaming directory
      brename -p ":" -r "-" -R -D   pdf-dirs

  More examples: https://github.com/shenwei356/brename

Flags:
  -d, --dry-run              print rename operations but do not run
  -i, --ignore-case          ignore case
  -D, --including-dir        rename directories
  -p, --pattern string       search pattern (regular expression)
  -R, --recursive            rename recursively
  -r, --replacement string   replacement. capture variables supported.  e.g. $1 represents the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character.
  -v, --verbose int          verbose level (0 for all, 1 for warning and error, 2 for only error)
  -V, --version              print version information and check for update
```


## Examples

For directory:

    $ tree
    .
    ├── abc
    │   ├── A.JPEG
    │   └── B.JPEG
    ├── a.jpeg
    └── b.jpeg

1. Renaming all `.jpeg` files to `.jpg` in all subdirectories.
 A dry run is firstly performed for safety.

        $ brename -p "\.jpeg" -r ".jpg" -R -d
        [INFO] checking: a.jpeg -> a.jpg [ok]
        [INFO] checking: b.jpeg -> b.jpg [ok]
        [INFO] 2 paths to be renamed

        $ brename -p "\.jpeg" -r ".jpg" -R
        [INFO] checking: a.jpeg -> a.jpg [ok]
        [INFO] checking: b.jpeg -> b.jpg [ok]
        [INFO] 2 paths to be renamed
        [INFO] renamed: a.jpeg -> a.jpg
        [INFO] renamed: b.jpeg -> b.jpg
        [INFO] 2 paths renamed

        $ tree
        .
        ├── abc
        │   ├── A.JPEG
        │   └── B.JPEG
        ├── a.jpg
        └── b.jpg

1. dry run and only show operations that will cause error

        # default value of -v is 0
        $ ../brename -p a -r b -R -D -d
        [INFO] checking: a.jpeg -> b.jpeg [new path existed]
        [INFO] checking: abc -> bbc [ok]
        [ERRO] 1 potential errors detected, please check

        $ brename -p a -r b -R -D -d -v 2
        [INFO] checking: a.jpeg -> b.jpeg [new path existed]
        [ERRO] 1 potential errors detected, please check

1. ignore cases

        $ brename -p "\.jpeg" -r ".jpg" -R -i
        [INFO] checking: abc/A.JPEG -> abc/A.jpg [ok]
        [INFO] checking: abc/B.JPEG -> abc/B.jpg [ok]
        [INFO] 2 paths to be renamed
        [INFO] renamed: abc/A.JPEG -> abc/A.jpg
        [INFO] renamed: abc/B.JPEG -> abc/B.jpg
        [INFO] 2 paths renamed

        $ tree
        .
        ├── abc
        │   ├── A.jpg
        │   └── B.jpg
        ├── a.jpg
        └── b.jpg

1. using capture variables, e.g., $1, $2 ...

        # or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
        $ brename -p "(a)" -r "\$1\$1"
        [INFO] checking: a.jpg -> aa.jpg [ok]
        [INFO] 1 paths to be renamed
        [INFO] renamed: a.jpg -> aa.jpg
        [INFO] 1 paths renamed

        $ tree                               
        .
        ├── aa.jpg
        ├── abc
        │   ├── A.jpg
        │   └── B.jpg
        └── b.jpg


1. even renaming directory

        $ brename -p "a" -r "A" -R -D
        [INFO] checking: aa.jpg -> AA.jpg [ok]
        [INFO] checking: abc -> Abc [ok]
        [INFO] 2 paths to be renamed
        [INFO] renamed: aa.jpg -> AA.jpg
        [INFO] renamed: abc -> Abc
        [INFO] 2 paths renamed

        $ tree                       
        .
        ├── AA.jpg
        ├── Abc
        │   ├── A.jpg
        │   └── B.jpg
        └── b.jpg


## Contact

[Create an issue](https://github.com/shenwei356/brename/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)
