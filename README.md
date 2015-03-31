brename
=======

Recursively batch rename files and directories by regular expression.

通过正则表达式递归、批量重命名文件和文件夹。

Install
-------

[Download](https://github.com/shenwei356/brename/releases)

This package is "go-gettable", just:

    go get github.com/shenwei356/brename
    go install github.com/shenwei356/brename

Usage
-----
    
    Usage: brename -s <regexp> -r <replacement> [-R] [-D] [path...]

    Options:
      -D=false: Rename directories
      -R=false: Recursively rename
      -r="": Replacement
      -s="": Regular expression


Example
-------
    
  1. a.jpeg -> a.jpg

    brename -s '\.jpeg$' -r '.jpg'

  2. ab.png -> abab.png

    brename -s '([ab]+)' -r '$1$1'

    
Copyright (c) 2014, Wei Shen (shenwei356@gmail.com)

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)
