# brename -- a practical cross-platform command-line tool for safely batch renaming files/directories via regular expression

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
- [Real-world examples](#real-world-examples)
- [Contact](#contact)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

- **Cross-platform**. Supporting Windows, Mac OS X and Linux.
- **Safe**. By ***checking potential conflicts and errors***.
- **File filtering**. Supporting including and excluding files via regular expression.
    No need to run commands like `find ./ -name "*.html" -exec CMD`.
- **Renaming submatch with corresponding value via key-value file**.
- **Renaming via ascending integer**.
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

[brename v2.5.1](https://github.com/shenwei356/brename/releases/tag/v2.5.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/brename/v2.5.1/total.svg)](https://github.com/shenwei356/brename/releases/tag/v2.5.1)

***Tip: run `brename -V` to check update !!!***

OS     |Arch      |File, 中国镜像                                                                                                                                                                              |Download Count
:------|:---------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Linux  |32-bit    |[brename_linux_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_linux_386.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_linux_386.tar.gz)                            |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_linux_386.tar.gz)
Linux  |**64-bit**|[**brename_linux_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_linux_amd64.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_linux_amd64.tar.gz)                  |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_linux_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_linux_amd64.tar.gz)
OS X   |32-bit    |[brename_darwin_386.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_darwin_386.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_darwin_386.tar.gz)                         |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_darwin_386.tar.gz)
OS X   |**64-bit**|[**brename_darwin_amd64.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_darwin_amd64.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_darwin_amd64.tar.gz)               |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_darwin_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_darwin_amd64.tar.gz)
Windows|32-bit    |[brename_windows_386.exe.tar.gz](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_windows_386.exe.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_windows_386.exe.tar.gz)          |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_386.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_windows_386.exe.tar.gz)
Windows|**64-bit**|[**brename_windows_amd64.exe.tar.gz**](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_windows_amd64.exe.tar.gz),<br/> [中国镜像](http://app.shenwei.me/data/brename/brename_windows_amd64.exe.tar.gz)|[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/brename/latest/brename_windows_amd64.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/brename/releases/download/v2.5.1/brename_windows_amd64.exe.tar.gz)


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

#### Method 3: For ArchLinux AUR users

    yaourt -S brename

## Usage

```
brename -- a practical cross-platform command-line tool for safely batch renaming files/directories via regular expression

Version: 2.5.1

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Attention:
  1. Paths starting with "." is ignored.
  2. Overwriting existed files is not allowed.
  3. Flag -f/--include-filters and -F/--exclude-filters support multiple values,
     e.g., -f ".html" -f ".htm".
     But ATTENTION: comma in filter is treated as separater of multiple filters.

Special replacement symbols:

  {nr}    Ascending integer
  {kv}    Corresponding value of the key (captured variable $n) by key-value file,
          n can be specified by flag -I/--key-capt-idx (default: 1)

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
  7. using key-value file
      brename -p "(.+)" -r "{kv}" -k kv.tsv
  8. do not touch file extension
      brename -p ".+" -r "{nr}" -f .mkv -f .mp4 -e
  9. only list paths that match pattern (-l)
      brename -i -f '.docx?$' -p . -R -l

  More examples: https://github.com/shenwei356/brename

Flags:
  -d, --dry-run                   print rename operations but do not run
  -F, --exclude-filters strings   exclude file filter(s) (regular expression, case ignored). multiple values supported, e.g., -F ".html" -F ".htm", but ATTENTION: comma in filter is treated as separater of multiple filters
  -h, --help                      help for brename
  -i, --ignore-case               ignore case
  -e, --ignore-ext                ignore file extension. i.e., replacement does not change file extension
  -f, --include-filters strings   include file filter(s) (regular expression, case ignored). multiple values supported, e.g., -f ".html" -f ".htm", but ATTENTION: comma in filter is treated as separater of multiple filters (default [.])
  -D, --including-dir             rename directories
  -K, --keep-key                  keep the key as value when no value found for the key
  -I, --key-capt-idx int          capture variable index of key (1-based) (default 1)
  -m, --key-miss-repl string      replacement for key with no corresponding value
  -k, --kv-file string            tab-delimited key-value file for replacing key with value when using "{kv}" in -r (--replacement)
  -l, --list                      only list paths that match pattern
  -a, --list-abs                  list absolute path, using along with -l/--list
  -p, --pattern string            search pattern (regular expression)
  -R, --recursive                 rename recursively
  -r, --replacement string        replacement. capture variables supported.  e.g. $1 represents the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character. Ascending integer is also supported by "{nr}"
  -n, --start-num int             starting number when using {nr} in replacement (default 1)
  -v, --verbose int               verbose level (0 for all, 1 for warning and error, 2 for only error)
  -V, --version                   print version information and check for update

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

1. Rename with number (-r `{nr}`)

        $ brename -p '(.+)\.' -r 'pic-{nr}.' -f .jpg
        [INFO] checking: [ ok ] 'AA.jpg' -> 'pic-1.jpg'
        [INFO] checking: [ ok ] 'b.jpg' -> 'pic-2.jpg'
        [INFO] 2 path(s) to be renamed

1. Replace submatch with corresponding value via key-value file (`-k/--kv-file`)

        $ more kv.tsv
        a       一
        b       二
        c       三

        $ brename -p '^(\w)' -r '{kv}' -k kv.tsv -K -i  -d
        [INFO] read key-value file: kv.tsv
        [INFO] 3 pairs of key-value loaded
        [INFO] checking: [ ok ] 'AA.jpg' -> '一A.jpg'
        [INFO] checking: [ ok ] 'b.jpg' -> '二.jpg'
        [WARN] checking: [ unchanged ] 'hello b.html' -> 'hello b.html'
        [WARN] checking: [ unchanged ] 'kv.tsv' -> 'kv.tsv'

1. Do not touch file extension (`-e/--ignore-ext`)

        $ brename -p '.+(S\d+E\d+).+' -r '$1' -d -e
        [INFO] checking: [ ok ] 'XXXXXX.S01E01.XXXX.720p.WEB-DL.AAC2.0.H.264-MC.ass' -> 'S01E01.ass'
        [INFO] checking: [ ok ] 'XXXXXX.S01E01.XXXX.720p.WEB-DL.AAC2.0.H.264-MC.mkv' -> 'S01E01.mkv'
        [INFO] 2 path(s) to be renamed

1. Auto mkdir

        $ brename -f .txt -p '-' -r '/'
        [INFO] checking: [ ok ] 'a-b-c.txt' -> 'a/b/c.txt'
        [INFO] 1 path(s) to be renamed
        [INFO] renamed: 'a-b-c.txt' -> 'a/b/c.txt'
        [INFO] 1 path(s) renamed

        $ tree a
        a
        └── b
            └── c.txt

1. only list paths that match pattern (`-l` and `-a`)

        $ brename -p '.gz$' -R -l
        binaries/brename_darwin_386.tar.gz
        binaries/brename_darwin_amd64.tar.gz
        binaries/brename_linux_386.tar.gz
        binaries/brename_linux_amd64.tar.gz
        binaries/brename_windows_386.exe.tar.gz
        binaries/brename_windows_amd64.exe.tar.gz
        
        $ brename -p '.gz$' -R -l
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_darwin_386.tar.gz
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_darwin_amd64.tar.gz
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_linux_386.tar.gz
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_linux_amd64.tar.gz
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_windows_386.exe.tar.gz
        /home/shenwei/project/src/github.com/shenwei356/brename/binaries/brename_windows_amd64.exe.tar.gz


## Real-world examples

1. Replace matches with corresponding pairing values

    1. Original files

            $ tree
            .
            ├── barcodes.tsv
            ├── tag_ATGCGTA.fasta
            ├── tag_CCCCCCC.fasta
            ├── tag_CGACGTC.fasta
            ├── tag_TCATAGC.fasta
            └── tag_TCTATAG.fasta

    1. Tab-delimited key-value file. Notice that `CCCCCCC` is not in it.

            $ cat barcodes.tsv
            CGACGTC S1
            ATGCGTA S2
            TCTATAG S4
            TCATAGC S3

    1. Renaming tag as sample names, marking `unknown` for non-existing tag.

            $ brename -p 'tag_(\w+)' -r '{kv}' -k barcodes.tsv -m unknown -d
            [INFO] read key-value file: barcodes.tsv
            [INFO] 4 pairs of key-value loaded
            [INFO] checking: [ ok ] 'tag_ATGCGTA.fasta' -> 'S2.fasta'
            [INFO] checking: [ ok ] 'tag_CCCCCCC.fasta' -> 'unknown.fasta'
            [INFO] checking: [ ok ] 'tag_CGACGTC.fasta' -> 'S1.fasta'
            [INFO] checking: [ ok ] 'tag_TCATAGC.fasta' -> 'S3.fasta'
            [INFO] checking: [ ok ] 'tag_TCTATAG.fasta' -> 'S4.fasta'
            [INFO] 5 path(s) to be renamed


1. Renaming PDF files for compatibility (moving from `EXT4` to `NTFS` file system):

    1. Original files

            $ tree -Q
            .
            ├── "0000 Test.pdf"
            ├── "2016 SeqKit _ A Cross-Platform and Ultrafast Toolkit for FASTA\342\201\204Q File Manipulation .pdf"
            ├── "metagenomics"
            │   ├── "2017 16S rRNA gene sequencing and healthy reference ranges for 28 clinically relevant microbial taxa from the human gut microbiome .pdf"
            │   ├── "2017 De novo assembly of viral quasispecies using overlap graphs .pdf"
            │   └── "2017 Tracking microbial colonization in fecal microbiota transplantation experiments via genome-resolved metagenomics .pdf"
            ├── "test2222222222222222222211111111122222222222222222233333333.pdf"
            └── "test.pdf"


    1. Removing "\n"

            $ brename -p "\n" -r " " -R

    1. Replacing ":" with "_"

            $ brename -p ":" -r " _" -R

    1. Shortening file names (prefering spliting with space)

            $ brename -R -f .pdf -i -p "^(.{30,50})[ \.].*.pdf" -r "\$1.pdf" -d

    1. Result

            $ tree -Q
            .
            ├── "0000 Test.pdf"
            ├── "2016 SeqKit _ A Cross-Platform and Ultrafast.pdf"
            ├── "metagenomics"
            │   ├── "2017 16S rRNA gene sequencing and healthy.pdf"
            │   ├── "2017 De novo assembly of viral quasispecies using.pdf"
            │   └── "2017 Tracking microbial colonization in fecal.pdf"
            ├── "test2222222222222222222211111111122222222222222222233333333.pdf"
            └── "test.pdf"

## Contact

[Create an issue](https://github.com/shenwei356/brename/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)
