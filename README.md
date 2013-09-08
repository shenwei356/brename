brename
===============

Recursively batch rename files and directories by regular expression.

通过正则表达式递归、批量重命名文件和文件夹。

Install
-------
This package is "go-gettable", just:

    go get github.com/shenwei356/brename

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
    
    brename -s "\.jpeg$" -r ".jpg"
    
Have a Try
----------
You can compile by yourself or just download the executable files immediately.

- [brename.exe](https://github.com/shenwei356/brename/blob/master/brename.win.tar.gz?raw=true) for Windows.
- [brename_x86_64](https://github.com/shenwei356/brename/blob/master/brename.x86_64.tar.gz?raw=true) and [brename_x86](https://github.com/shenwei356/brename/blob/master/brename.x86.tar.gz?raw=true) for 64bit and 32bit Linux.
    
Copyright (c) 2013, Wei Shen (shenwei356@gmail.com)

[MIT License](https://github.com/shenwei356/brename/blob/master/LICENSE)