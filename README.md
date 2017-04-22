# brename -- a cross-platform command-line tool for batch renaming files/directories

[![Built with GoLang](https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/shenwei356/brename)](https://goreportcard.com/report/github.com/shenwei356/brename)
[![Cross-platform](https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat)](#download)
[![Latest Version](https://img.shields.io/github/release/shenwei356/brename.svg?style=flat?maxAge=86400)](https://github.com/shenwei356/brename/releases)
[![Github Releases](https://img.shields.io/github/downloads/shenwei356/brename/latest/total.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases)


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

## Installation

`brename` is implemented in [Go](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/brename/releases) page.

#### Method 1: Download binaries

[brename v0.2.0](https://github.com/shenwei356/brename/releases/tag/v0.2.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/brename/v0.2.0/total.svg)](https://github.com/shenwei356/brename/releases/tag/v0.1.7)


OS     |Arch      |File, (mirror为中国用户下载镜像链接)                                                                                                                                                                         |Download Count
:------|:---------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Linux  |32-bit    |[brename_linux_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_linux_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_386.tar.gz))                            |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_linux_386.tar.gz)
Linux  |**64-bit**|[**brename_linux_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_linux_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_linux_amd64.tar.gz))                  |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_linux_amd64.tar.gz)
OS X   |32-bit    |[brename_darwin_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_darwin_386.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_386.tar.gz))                         |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_darwin_386.tar.gz)
OS X   |**64-bit**|[**brename_darwin_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_darwin_amd64.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_darwin_amd64.tar.gz))               |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_darwin_amd64.tar.gz)
Windows|32-bit    |[brename_windows_386.exe.tar.gz](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_windows_386.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_386.exe.tar.gz))          |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_386.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_windows_386.exe.tar.gz)
Windows|**64-bit**|[**brename_windows_amd64.exe.tar.gz**](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_windows_amd64.exe.tar.gz), ([mirror](http://app.shenwei.me/data/brename/brename_windows_amd64.exe.tar.gz))|[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_amd64.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v0.2.0/brename_windows_amd64.exe.tar.gz)


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
brename -- a cross-platform command-line tool for batch renaming files/directories

Version: 0.2.0

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Usage:
  brename [flags]

Examples:
  1. renaming all .jpeg files to .jpg in all subdirs
      brename -p "\.jpeg" -r ".jpg" -R   dir
  2. doubling "a"
      brename -p "(a)" -r "\$1\$1"  or brename -p "(a)" -r '$1$1'
  3. even renaming directory
      brename -p ":" -r "-" -R -D   pdf-dirs
  4. dry run and showing potential dangerous operation
      brename -p "abc" -d -v 2

  More examples: https://github.com/shenwei356/brename

Flags:
  -d, --dry-run              print rename operations but do not run
  -i, --ignore-case          ignore case
  -D, --including-dir        rename directories
  -p, --pattern string       search pattern (regular expression)
  -R, --recursive            rename recursively
  -r, --replacement string   replacement
  -v, --verbose int          verbose level (0 for all, 1 for warning and error, 2 for only error)
  -V, --version              print version information and check for update

```


## Examples


## Contact

[Create an issue](https://github.com/shenwei356/brename/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)
